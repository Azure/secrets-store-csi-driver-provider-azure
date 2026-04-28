package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/date"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/utils"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

const (
	// Pod Identity podNameHeader
	podNameHeader = "podname"
	// Pod Identity podNamespaceHeader
	podNamespaceHeader = "podns"

	// For Azure AD Workload Identity, the audience recommended for use is
	// "api://AzureADTokenExchange"
	DefaultTokenAudience = "api://AzureADTokenExchange" // nolint

	// For AKS Identity Binding, the audience is "api://AKSIdentityBinding"
	IdentityBindingTokenAudience = "api://AKSIdentityBinding" // nolint:gosec
)

var (
	// ErrServiceAccountTokensNotFound is returned when the service account token is not found
	ErrServiceAccountTokensNotFound = errors.New("service account tokens not found")

	// proxyTransport is the transport for identity binding proxy.
	// Set via SetProxyTransport during initialization; error is checked
	// lazily when identity binding is actually used.
	proxyTransport    policy.Transporter
	proxyTransportErr error
)

// SetProxyTransport sets the identity binding proxy transport.
// This must be called exactly once from main() before the gRPC server starts.
// It is not goroutine-safe; the single-write-at-init pattern ensures safety.
func SetProxyTransport(t policy.Transporter, err error) {
	proxyTransport = t
	proxyTransportErr = err
}

// Token encapsulates the access token used to authorize Azure requests.
// https://docs.microsoft.com/en-us/azure/active-directory/develop/v1-oauth2-client-creds-grant-flow#service-to-service-access-token-response
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`

	ExpiresIn json.Number `json:"expires_in"`
	ExpiresOn json.Number `json:"expires_on"`
	NotBefore json.Number `json:"not_before"`

	Resource string `json:"resource"`
	Type     string `json:"token_type"`
}

// Expires returns the time.Time when the Token expires.
func (t Token) Expires() time.Time {
	s, err := t.ExpiresOn.Float64()
	if err != nil {
		s = -3600
	}

	expiration := date.NewUnixTimeFromSeconds(s)

	return time.Time(expiration).UTC()
}

// PodIdentityResponse is the response received from aad-pod-identity when requesting token
// on behalf of the pod
type PodIdentityResponse struct {
	Token    Token  `json:"token"`
	ClientID string `json:"clientid"`
}

// IdentityMode represents the authentication mode used to access Azure resources.
// Only one mode can be active at a time.
type IdentityMode int

const (
	// IdentityModeNone indicates no explicit identity mode is set.
	// Falls back to workload identity (if clientID+token are present) or service principal.
	IdentityModeNone IdentityMode = iota
	// IdentityModePodIdentity uses aad-pod-identity via NMI.
	IdentityModePodIdentity
	// IdentityModeVMManagedIdentity uses VM/VMSS managed identity.
	IdentityModeVMManagedIdentity
	// IdentityModeAzureTokenProxy uses identity binding via the Azure token proxy.
	IdentityModeAzureTokenProxy
)

// String returns a human-readable name for the identity mode.
func (m IdentityMode) String() string {
	switch m {
	case IdentityModeNone:
		return "None"
	case IdentityModePodIdentity:
		return "PodIdentity"
	case IdentityModeVMManagedIdentity:
		return "VMManagedIdentity"
	case IdentityModeAzureTokenProxy:
		return "AzureTokenProxy"
	default:
		return fmt.Sprintf("Unknown(%d)", int(m))
	}
}

// Config is the required parameters for auth config
type Config struct {
	// IdentityMode specifies which authentication mode to use.
	IdentityMode IdentityMode
	// UserAssignedIdentityID is the user-assigned managed identity clientID
	UserAssignedIdentityID string
	// AADClientSecret is the client secret for SP access mode
	AADClientSecret string
	// AADClientID is the clientID for SP access mode
	AADClientID string
	// WorkloadIdentityClientID is the clientID used for both workload identity and identity binding.
	// This clientID can be an Azure AD Application or a Managed identity.
	WorkloadIdentityClientID string
	// ServiceAccountToken is the service account token for workload identity or identity binding.
	// For workload identity, this is the token with audience "api://AzureADTokenExchange".
	// For identity binding, this is the token with audience "api://AKSIdentityBinding".
	// The token will be exchanged for an Azure AD Token.
	ServiceAccountToken string
}

// saToken represents a single service account token entry in the CSI token request.
type saToken struct {
	Token               string    `json:"token"`
	ExpirationTimestamp time.Time `json:"expirationTimestamp"`
}

type workloadIdentityCredential struct {
	assertion string
	cred      *azidentity.ClientAssertionCredential
}

type workloadIdentityCredentialOptions struct {
	azcore.ClientOptions
	DisableInstanceDiscovery bool
}

type podIdentityCredential struct {
	podName      string
	podNamespace string
	resource     string
	tenantID     string
	nmiPort      string
}

// NewConfig returns new auth config
func NewConfig(
	mode IdentityMode,
	userAssignedIdentityID,
	workloadIdentityClientID,
	serviceAccountToken string,
	secrets map[string]string) (Config, error) {
	config := Config{
		IdentityMode:             mode,
		UserAssignedIdentityID:   userAssignedIdentityID,
		WorkloadIdentityClientID: workloadIdentityClientID,
		ServiceAccountToken:      serviceAccountToken,
	}

	useWorkloadIdentity := len(workloadIdentityClientID) > 0 && len(serviceAccountToken) > 0

	if mode == IdentityModeNone && !useWorkloadIdentity {
		var err error
		if config.AADClientID, config.AADClientSecret, err = getCredential(secrets); err != nil {
			return config, err
		}
	}

	return config, nil
}

// GetCredential returns the azure credential to use based on the auth config
func (c Config) GetCredential(podName, podNamespace, resource, aadEndpoint, tenantID, nmiPort string) (azcore.TokenCredential, error) {
	switch c.IdentityMode {
	case IdentityModePodIdentity:
		return getPodIdentityTokenCredential(podName, podNamespace, resource, tenantID, nmiPort)
	case IdentityModeVMManagedIdentity:
		return getManagedIdentityTokenCredential(c.UserAssignedIdentityID)
	case IdentityModeAzureTokenProxy:
		if len(c.WorkloadIdentityClientID) == 0 || len(c.ServiceAccountToken) == 0 {
			return nil, fmt.Errorf("workload identity client ID and service account token are required for identity binding")
		}
		return getIdentityBindingTokenCredential(c.WorkloadIdentityClientID, c.ServiceAccountToken, aadEndpoint, tenantID)
	case IdentityModeNone:
		// Try workload identity, then service principal
		if len(c.WorkloadIdentityClientID) > 0 && len(c.ServiceAccountToken) > 0 {
			return getWorkloadIdentityTokenCredential(c.WorkloadIdentityClientID, c.ServiceAccountToken, aadEndpoint, tenantID)
		}
		if len(c.AADClientSecret) > 0 && len(c.AADClientID) > 0 {
			return getServicePrincipalTokenCredential(c.AADClientID, c.AADClientSecret, aadEndpoint, tenantID)
		}
		return nil, fmt.Errorf("no identity mode is enabled")
	default:
		return nil, fmt.Errorf("unknown identity mode: %s", c.IdentityMode)
	}
}

func newWorkloadIdentityCredential(tenantID, clientID, assertion string, options *workloadIdentityCredentialOptions) (azcore.TokenCredential, error) {
	w := &workloadIdentityCredential{assertion: assertion}
	cred, err := azidentity.NewClientAssertionCredential(tenantID, clientID, w.getAssertion, &azidentity.ClientAssertionCredentialOptions{
		ClientOptions:            options.ClientOptions,
		DisableInstanceDiscovery: options.DisableInstanceDiscovery,
	})
	if err != nil {
		return nil, err
	}
	w.cred = cred
	return w, nil
}

func (w *workloadIdentityCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return w.cred.GetToken(ctx, opts)
}

func (w *workloadIdentityCredential) getAssertion(context.Context) (string, error) {
	return w.assertion, nil
}

func getWorkloadIdentityTokenCredential(clientID, signedAssertion, aadEndpoint, tenantID string) (azcore.TokenCredential, error) {
	opts := &workloadIdentityCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: cloud.Configuration{
				ActiveDirectoryAuthorityHost: aadEndpoint,
			},
		},
	}
	return newWorkloadIdentityCredential(tenantID, clientID, signedAssertion, opts)
}

func getIdentityBindingTokenCredential(clientID, signedAssertion, aadEndpoint, tenantID string) (azcore.TokenCredential, error) {
	klog.V(5).InfoS("using identity binding (azure token proxy) to retrieve token", "clientID", clientID)

	// Check if the proxy transport was successfully initialized
	if proxyTransportErr != nil {
		return nil, fmt.Errorf("proxy transport not available: %w", proxyTransportErr)
	}
	if proxyTransport == nil {
		return nil, fmt.Errorf("identity binding proxy transport not initialized (call SetProxyTransport during startup)")
	}

	opts := &workloadIdentityCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: cloud.Configuration{
				ActiveDirectoryAuthorityHost: aadEndpoint,
			},
		},
		// DisableInstanceDiscovery must be true when using the proxy to avoid
		// unnecessary instance discovery calls that don't work through the proxy
		DisableInstanceDiscovery: true,
	}

	// Use the proxy transport extracted from the SDK via reflect
	opts.ClientOptions.Transport = proxyTransport

	return newWorkloadIdentityCredential(tenantID, clientID, signedAssertion, opts)
}

func getServicePrincipalTokenCredential(clientID, secret, aadEndpoint, tenantID string) (azcore.TokenCredential, error) {
	opts := &azidentity.ClientSecretCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: cloud.Configuration{
				ActiveDirectoryAuthorityHost: aadEndpoint,
			},
		},
	}
	return azidentity.NewClientSecretCredential(tenantID, clientID, secret, opts)
}

func getManagedIdentityTokenCredential(identityClientID string) (azcore.TokenCredential, error) {
	opts := &azidentity.ManagedIdentityCredentialOptions{}
	if len(identityClientID) > 0 {
		opts.ID = azidentity.ClientID(identityClientID)
	}
	return azidentity.NewManagedIdentityCredential(opts)
}

func (c *podIdentityCredential) GetToken(ctx context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	// For usePodIdentity mode, the CSI driver makes an authorization request to fetch token for a resource from the NMI host endpoint (http://127.0.0.1:2579/host/token/).
	// The request includes the pod namespace `podns` and the pod name `podname` in the request header and the resource endpoint of the resource requesting the token.
	// The NMI server identifies the pod based on the `podns` and `podname` in the request header and then queries k8s (through MIC) for a matching azure identity.
	// Then nmi makes an adal request to get a token for the resource in the request, returns the `token` and the `clientid` as a response to the CSI request.
	klog.V(5).InfoS("using pod identity to retrieve token", "pod", klog.ObjectRef{Namespace: c.podNamespace, Name: c.podName})

	endpoint := fmt.Sprintf("http://localhost:%s/host/token/?resource=%s", c.nmiPort, c.resource)
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return azcore.AccessToken{}, err
	}
	req.Header.Add(podNamespaceHeader, c.podNamespace)
	req.Header.Add(podNameHeader, c.podName)
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return azcore.AccessToken{}, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return azcore.AccessToken{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return azcore.AccessToken{}, fmt.Errorf("nmi response failed with status code: %d, response body: %+v", resp.StatusCode, string(bodyBytes))
	}

	podIdentityResponse := &PodIdentityResponse{}
	if err = json.Unmarshal(bodyBytes, &podIdentityResponse); err != nil {
		return azcore.AccessToken{}, err
	}
	klog.V(5).InfoS("successfully acquired access token", "accessToken", utils.RedactSecureString(podIdentityResponse.Token.AccessToken), "clientID", utils.RedactSecureString(podIdentityResponse.ClientID), "pod", klog.ObjectRef{Namespace: c.podNamespace, Name: c.podName})

	token, clientID := podIdentityResponse.Token, podIdentityResponse.ClientID
	if token.AccessToken == "" || clientID == "" {
		return azcore.AccessToken{}, fmt.Errorf("nmi did not return expected values in response: token and clientid")
	}

	return azcore.AccessToken{
		Token:     token.AccessToken,
		ExpiresOn: podIdentityResponse.Token.Expires(),
	}, nil
}

func getPodIdentityTokenCredential(podName, podNamespace, resource, tenantID, nmiPort string) (azcore.TokenCredential, error) {
	if len(podName) == 0 || len(podNamespace) == 0 {
		return nil, fmt.Errorf("pod information is not available. deploy a CSIDriver object to set podInfoOnMount: true")
	}
	return &podIdentityCredential{
		podName:      podName,
		podNamespace: podNamespace,
		resource:     resource,
		tenantID:     tenantID,
		nmiPort:      nmiPort,
	}, nil
}

// getCredential gets clientid and clientsecret from the secrets
func getCredential(secrets map[string]string) (string, string, error) {
	if secrets == nil {
		return "", "", fmt.Errorf("failed to get credentials, nodePublishSecretRef secret is not set")
	}

	var clientID, clientSecret string
	for k, v := range secrets {
		switch strings.ToLower(k) {
		case "clientid":
			clientID = v
		case "clientsecret":
			clientSecret = v
		}
	}

	if clientID == "" {
		return "", "", fmt.Errorf("could not find clientid in secrets")
	}
	if clientSecret == "" {
		return "", "", fmt.Errorf("could not find clientsecret in secrets")
	}
	return clientID, clientSecret, nil
}

// ParseServiceAccountToken parses the bound service account token for the
// workload identity audience from the tokens passed from driver as part of MountRequest.
// ref: https://kubernetes-csi.github.io/docs/token-requests.html
func ParseServiceAccountToken(saTokens string) (string, error) {
	return parseTokenForAudience(saTokens, DefaultTokenAudience)
}

// ParseIdentityBindingToken parses the service account token for the
// identity binding audience from the tokens passed from driver as part of MountRequest.
func ParseIdentityBindingToken(saTokens string) (string, error) {
	return parseTokenForAudience(saTokens, IdentityBindingTokenAudience)
}

// parseTokenForAudience extracts a service account token for a specific audience
// from the JSON-encoded token map sent by the CSI driver.
func parseTokenForAudience(saTokens, audience string) (string, error) {
	klog.V(5).InfoS("parsing service account token", "audience", audience)
	if len(saTokens) == 0 {
		return "", ErrServiceAccountTokensNotFound
	}

	var tokens map[string]saToken
	if err := json.Unmarshal([]byte(saTokens), &tokens); err != nil {
		return "", fmt.Errorf("failed to unmarshal service account tokens, error: %w", err)
	}

	entry, ok := tokens[audience]
	if !ok || entry.Token == "" {
		return "", fmt.Errorf("token for audience %s not found", audience)
	}
	return entry.Token, nil
}

func getScope(resource string) string {
	scope := resource
	if !strings.HasSuffix(resource, "/.default") {
		scope += "/.default"
	}
	return scope
}
