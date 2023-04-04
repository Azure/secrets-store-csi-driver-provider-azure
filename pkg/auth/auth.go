package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/utils"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest/adal"
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
	DefaultTokenAudience = "api://AzureADTokenExchange" //nolint
)

var (
	// ErrServiceAccountTokensNotFound is returned when the service account token is not found
	ErrServiceAccountTokensNotFound = errors.New("service account tokens not found")
)

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

// PodIdentityResponse is the response received from aad-pod-identity when requesting token
// on behalf of the pod
type PodIdentityResponse struct {
	Token    adal.Token `json:"token"`
	ClientID string     `json:"clientid"`
}

// Config is the required parameters for auth config
type Config struct {
	// UsePodIdentity is set to true if access mode is using aad-pod-identity
	UsePodIdentity bool
	// UseVMManagedIdentity is set to true if access mode is using managed identity
	UseVMManagedIdentity bool
	// UserAssignedIdentityID is the user-assigned managed identity clientID
	UserAssignedIdentityID string
	// AADClientSecret is the client secret for SP access mode
	AADClientSecret string
	// AADClientID is the clientID for SP access mode
	AADClientID string
	// WorkloadIdentityClientID is the clientID for workload identity
	// this clientID can be an Azure AD Application or a Managed identity
	// NOTE: workload identity federation with managed identity is currently not supported
	WorkloadIdentityClientID string
	// WorkloadIdentityToken is the service account token for workload identity
	// this token will be exchanged for an Azure AD Token based on the federated identity credential
	// this service account token is associated with the workload requesting the volume mount
	WorkloadIdentityToken string
}

// SATokens represents the service account tokens sent as part of the MountRequest
type SATokens struct {
	APIAzureADTokenExchange struct {
		Token               string    `json:"token"`
		ExpirationTimestamp time.Time `json:"expirationTimestamp"`
	} `json:"api://AzureADTokenExchange"`
}

type workloadIdentityCredential struct {
	assertion string
	cred      *azidentity.ClientAssertionCredential
}

type workloadIdentityCredentialOptions struct {
	azcore.ClientOptions
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
	usePodIdentity,
	useVMManagedIdentity bool,
	userAssignedIdentityID,
	workloadIdentityClientID,
	workloadIdentityToken string,
	secrets map[string]string) (Config, error) {
	config := Config{}
	// aad-pod-identity and user assigned managed identity modes are currently mutually exclusive
	if usePodIdentity && useVMManagedIdentity {
		return config, fmt.Errorf("cannot enable both pod identity and user-assigned managed identity")
	}
	useWorkloadIdentity := len(workloadIdentityClientID) > 0 && len(workloadIdentityToken) > 0

	if !usePodIdentity && !useVMManagedIdentity && !useWorkloadIdentity {
		var err error
		if config.AADClientID, config.AADClientSecret, err = getCredential(secrets); err != nil {
			return config, err
		}
	}

	config.UsePodIdentity = usePodIdentity
	config.UseVMManagedIdentity = useVMManagedIdentity
	config.UserAssignedIdentityID = userAssignedIdentityID
	config.WorkloadIdentityClientID = workloadIdentityClientID
	config.WorkloadIdentityToken = workloadIdentityToken

	return config, nil
}

// GetCredential returns the azure credential to use based on the auth config
func (c Config) GetCredential(podName, podNamespace, resource, aadEndpoint, tenantID, nmiPort string) (azcore.TokenCredential, error) {
	// use switch case to ensure only one of the identity modes is enabled
	switch {
	case c.UsePodIdentity:
		return getPodIdentityTokenCredential(podName, podNamespace, resource, tenantID, nmiPort)
	case c.UseVMManagedIdentity:
		return getManagedIdentityTokenCredential(c.UserAssignedIdentityID)
	case len(c.AADClientSecret) > 0 && len(c.AADClientID) > 0:
		return getServicePrincipalTokenCredential(c.AADClientID, c.AADClientSecret, aadEndpoint, tenantID)
	case len(c.WorkloadIdentityClientID) > 0 && len(c.WorkloadIdentityToken) > 0:
		return getWorkloadIdentityTokenCredential(c.WorkloadIdentityClientID, c.WorkloadIdentityToken, aadEndpoint, tenantID)
	default:
		return nil, fmt.Errorf("no identity mode is enabled")
	}
}

func newWorkloadIdentityCredential(tenantID, clientID, assertion string, options *workloadIdentityCredentialOptions) (azcore.TokenCredential, error) {
	w := &workloadIdentityCredential{assertion: assertion}
	cred, err := azidentity.NewClientAssertionCredential(tenantID, clientID, w.getAssertion, &azidentity.ClientAssertionCredentialOptions{ClientOptions: options.ClientOptions})
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
	opts := &azidentity.ManagedIdentityCredentialOptions{
		ID: azidentity.ClientID(identityClientID),
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
		return "", "", fmt.Errorf("could not find clientid in secrets(%v)", secrets)
	}
	if clientSecret == "" {
		return "", "", fmt.Errorf("could not find clientsecret in secrets(%v)", secrets)
	}
	return clientID, clientSecret, nil
}

// ParseServiceAccountToken parses the bound service account token from the tokens
// passed from driver as part of MountRequest.
// ref: https://kubernetes-csi.github.io/docs/token-requests.html
func ParseServiceAccountToken(saTokens string) (string, error) {
	klog.V(5).InfoS("parsing service account token for workload identity")
	if len(saTokens) == 0 {
		return "", ErrServiceAccountTokensNotFound
	}

	// Bound token is of the format:
	// "csi.storage.k8s.io/serviceAccount.tokens": {
	//  <audience>: {
	//    'token': <token>,
	//    'expirationTimestamp': <expiration timestamp in RFC3339 format>,
	//  },
	//  ...
	// }
	tokens := SATokens{}
	if err := json.Unmarshal([]byte(saTokens), &tokens); err != nil {
		return "", fmt.Errorf("failed to unmarshal service account tokens, error: %w", err)
	}
	klog.V(5).InfoS("successfully unmarshaled service account tokens")
	if tokens.APIAzureADTokenExchange.Token == "" {
		return "", fmt.Errorf("token for audience %s not found", DefaultTokenAudience)
	}
	return tokens.APIAzureADTokenExchange.Token, nil
}

func getScope(resource string) string {
	scope := resource
	if !strings.HasSuffix(resource, "/.default") {
		scope += "/.default"
	}
	return scope
}
