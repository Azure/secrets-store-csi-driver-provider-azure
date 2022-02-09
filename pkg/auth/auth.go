package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/utils"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

const (
	// Pod Identity podNameHeader
	podNameHeader = "podname"
	// Pod Identity podNamespaceHeader
	podNamespaceHeader = "podns"

	// the format for expires_on in UTC with AM/PM
	expiresOnDateFormatPM = "1/2/2006 15:04:05 PM +00:00"
	// the format for expires_on in UTC without AM/PM
	expiresOnDateFormat = "1/2/2006 15:04:05 +00:00"
)

var (
	// ErrServiceAccountTokensNotFound is returned when the service account token is not found
	ErrServiceAccountTokensNotFound = errors.New("service account tokens not found")
)

// authResult contains the subset of results from token acquisition operation in ConfidentialClientApplication
// For details see https://aka.ms/msal-net-authenticationresult
type authResult struct {
	accessToken    string
	expiresOn      time.Time
	grantedScopes  []string
	declinedScopes []string
}

// NMIResponse is the response received from aad-pod-identity when requesting token
// on behalf of the pod
type NMIResponse struct {
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

// GetAuthorizer returns an Azure authorizer based on the provided azure identity
func (c Config) GetAuthorizer(ctx context.Context, podName, podNamespace, resource, aadEndpoint, tenantID, nmiPort string) (autorest.Authorizer, error) {
	if c.UsePodIdentity {
		return getAuthorizerForPodIdentity(podName, podNamespace, resource, aadEndpoint, tenantID, nmiPort)
	}
	if c.UseVMManagedIdentity {
		return getAuthorizerForManagedIdentity(resource, c.UserAssignedIdentityID)
	}
	if len(c.AADClientSecret) > 0 && len(c.AADClientID) > 0 {
		return getAuthorizerForServicePrincipal(c.AADClientID, c.AADClientSecret, resource, aadEndpoint, tenantID)
	}
	if len(c.WorkloadIdentityToken) > 0 && len(c.WorkloadIdentityClientID) > 0 {
		return getAuthorizerForWorkloadIdentity(ctx, c.WorkloadIdentityClientID, c.WorkloadIdentityToken, resource, aadEndpoint, tenantID)
	}

	return nil, fmt.Errorf("no valid identity access mode specified")
}

func getAuthorizerForWorkloadIdentity(ctx context.Context, clientID, signedAssertion, resource, aadEndpoint, tenantID string) (autorest.Authorizer, error) {
	cred, err := confidential.NewCredFromAssertion(signedAssertion)
	if err != nil {
		return nil, fmt.Errorf("failed to create confidential creds: %w", err)
	}
	confidentialClientApp, err := confidential.New(clientID, cred,
		confidential.WithAuthority(fmt.Sprintf("%s%s/oauth2/token", aadEndpoint, tenantID)))
	if err != nil {
		return nil, fmt.Errorf("failed to create confidential client app: %w", err)
	}
	scope := strings.TrimSuffix(resource, "/")
	// .default needs to be added to the scope
	if !strings.HasSuffix(resource, ".default") {
		scope += "/.default"
	}
	result, err := confidentialClientApp.AcquireTokenByCredential(ctx, []string{scope})
	if err != nil {
		return nil, fmt.Errorf("failed to acquire token: %w", err)
	}

	token := adal.Token{
		AccessToken: result.AccessToken,
		Resource:    resource,
		Type:        "Bearer",
	}
	token.ExpiresOn, err = parseExpiresOn(result.ExpiresOn.UTC().Local().Format(expiresOnDateFormat))
	if err != nil {
		return nil, fmt.Errorf("failed to parse expires_on: %w", err)
	}
	return autorest.NewBearerAuthorizer(authResult{
		accessToken:    result.AccessToken,
		expiresOn:      result.ExpiresOn,
		grantedScopes:  result.GrantedScopes,
		declinedScopes: result.DeclinedScopes,
	}), nil
}

// OAuthToken implements the OAuthTokenProvider interface.  It returns the current access token.
func (ar authResult) OAuthToken() string {
	return ar.accessToken
}

func getAuthorizerForServicePrincipal(clientID, clientSecret, resource, aadEndpoint, tenantID string) (autorest.Authorizer, error) {
	oauthConfig, err := adal.NewOAuthConfig(aadEndpoint, tenantID)
	if err != nil {
		return nil, err
	}
	spt, err := adal.NewServicePrincipalToken(
		*oauthConfig,
		clientID,
		clientSecret,
		resource)
	if err != nil {
		return nil, err
	}
	return autorest.NewBearerAuthorizer(spt), nil
}

func getAuthorizerForManagedIdentity(resource, identityClientID string) (autorest.Authorizer, error) {
	managedIdentityOpts := &adal.ManagedIdentityOptions{ClientID: identityClientID}
	spt, err := adal.NewServicePrincipalTokenFromManagedIdentity(resource, managedIdentityOpts)
	if err != nil {
		return nil, err
	}
	return autorest.NewBearerAuthorizer(spt), nil
}

func getAuthorizerForPodIdentity(podName, podNamespace, resource, aadEndpoint, tenantID, nmiPort string) (autorest.Authorizer, error) {
	// For usePodIdentity mode, the CSI driver makes an authorization request to fetch token for a resource from the NMI host endpoint (http://127.0.0.1:2579/host/token/).
	// The request includes the pod namespace `podns` and the pod name `podname` in the request header and the resource endpoint of the resource requesting the token.
	// The NMI server identifies the pod based on the `podns` and `podname` in the request header and then queries k8s (through MIC) for a matching azure identity.
	// Then nmi makes an adal request to get a token for the resource in the request, returns the `token` and the `clientid` as a response to the CSI request.
	klog.V(5).InfoS("using pod identity to retrieve token", "pod", klog.ObjectRef{Namespace: podNamespace, Name: podName})
	// pod name and namespace are required for the Key Vault provider to request a token
	// on behalf of the application pod
	if len(podName) == 0 || len(podNamespace) == 0 {
		return nil, fmt.Errorf("pod information is not available. deploy a CSIDriver object to set podInfoOnMount: true")
	}

	endpoint := fmt.Sprintf("http://localhost:%s/host/token/?resource=%s", nmiPort, resource)
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(podNamespaceHeader, podNamespace)
	req.Header.Add(podNameHeader, podName)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nmi response failed with status code: %d, response body: %+v", resp.StatusCode, string(bodyBytes))
	}

	var nmiResp = new(NMIResponse)
	err = json.Unmarshal(bodyBytes, &nmiResp)
	if err != nil {
		return nil, err
	}
	klog.V(5).InfoS("successfully acquired access token", "accessToken", utils.RedactClientID(nmiResp.Token.AccessToken), "clientID", utils.RedactClientID(nmiResp.ClientID), "pod", klog.ObjectRef{Namespace: podNamespace, Name: podName})

	token, clientID := nmiResp.Token, nmiResp.ClientID
	if token.AccessToken == "" || clientID == "" {
		return nil, fmt.Errorf("nmi did not return expected values in response: token and clientid")
	}

	oauthConfig, err := adal.NewOAuthConfig(aadEndpoint, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth config: %w", err)
	}
	spt, err := adal.NewServicePrincipalTokenFromManualToken(*oauthConfig, clientID, resource, token, nil)
	if err != nil {
		return nil, err
	}
	return autorest.NewBearerAuthorizer(spt), nil
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

// Vendored from https://github.com/Azure/go-autorest/blob/def88ef859fb980eff240c755a70597bc9b490d0/autorest/adal/token.go
// converts expires_on to the number of seconds
func parseExpiresOn(s string) (json.Number, error) {
	// convert the expiration date to the number of seconds from now
	timeToDuration := func(t time.Time) json.Number {
		dur := t.Sub(time.Now().UTC())
		return json.Number(strconv.FormatInt(int64(dur.Round(time.Second).Seconds()), 10))
	}
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		// this is the number of seconds case, no conversion required
		return json.Number(s), nil
	} else if eo, err := time.Parse(expiresOnDateFormatPM, s); err == nil {
		return timeToDuration(eo), nil
	} else if eo, err := time.Parse(expiresOnDateFormat, s); err == nil {
		return timeToDuration(eo), nil
	} else {
		// unknown format
		return json.Number(""), err
	}
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
	tokens := make(map[string]interface{})
	if err := json.Unmarshal([]byte(saTokens), &tokens); err != nil {
		return "", fmt.Errorf("failed to unmarshal service account tokens, error: %w", err)
	}
	klog.V(5).InfoS("successfully unmarshalled service account tokens", "tokens", len(tokens))
	// For Azure AD Workload Identity, the audience recommended for use is
	// "api://AzureADTokenExchange"
	audience := "api://AzureADTokenExchange"
	if _, ok := tokens[audience]; !ok {
		return "", fmt.Errorf("token for audience %s not found", audience)
	}
	token, ok := tokens[audience].(map[string]interface{})["token"].(string)
	if !ok {
		return "", fmt.Errorf("token for audience %s not found", audience)
	}
	return token, nil
}
