package azure

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	yaml "gopkg.in/yaml.v2"

	kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/pkg/errors"
)

// Type of Azure Key Vault objects
const (
	// VaultObjectTypeSecret secret vault object type
	VaultObjectTypeSecret string = "secret"
	// VaultObjectTypeKey key vault object type
	VaultObjectTypeKey string = "key"
	// VaultObjectTypeCertificate certificate vault object type
	VaultObjectTypeCertificate string = "cert"
	// OAuthGrantTypeServicePrincipal for client credentials flow
	OAuthGrantTypeServicePrincipal OAuthGrantType = iota
	// OAuthGrantTypeDeviceFlow for device-auth flow
	OAuthGrantTypeDeviceFlow
	// Pod Identity nmiendpoint
	nmiendpoint = "http://localhost:2579/host/token/"
	// Pod Identity podnameheader
	podnameheader = "podname"
	// Pod Identity podnsheader
	podnsheader = "podns"
)

// NMIResponse is the response received from aad-pod-identity
type NMIResponse struct {
	Token    adal.Token `json:"token"`
	ClientID string     `json:"clientid"`
}

// OAuthGrantType specifies which grant type to use.
type OAuthGrantType int

// AuthGrantType ...
func AuthGrantType() OAuthGrantType {
	return OAuthGrantTypeServicePrincipal
}

// Provider implements the secrets-store-csi-driver provider interface
type Provider struct {
	// the name of the Azure Key Vault instance
	KeyvaultName string
	// the name of the Azure Key Vault objects, since attributes can only be strings, this will be mapped to StringArray, which is an array of KeyVaultObject
	Objects []KeyVaultObject
	// tenantID in AAD
	TenantID string
	// POD AAD Identity flag
	UsePodIdentity bool
	// VM managed identity flag
	UseVMManagedIdentity bool
	// User Assign Identity
	UserAssignedIdentityID string
	// AAD app client secret (if not using POD AAD Identity)
	AADClientSecret string
	// AAD app client secret id (if not using POD AAD Identity)
	AADClientID string
	// the name of the pod (if using POD AAD Identity)
	PodName string
	// the namespace of the pod (if using POD AAD Identity)
	PodNamespace string
}

// KeyVaultObject holds keyvault object related config
type KeyVaultObject struct {
	// the name of the Azure Key Vault objects
	ObjectName string `json:"objectName" yaml:"objectName"`
	// the filename the object will be written to
	ObjectAlias string `json:"objectAlias" yaml:"objectAlias"`
	// the version of the Azure Key Vault objects
	ObjectVersion string `json:"objectVersion" yaml:"objectVersion"`
	// the type of the Azure Key Vault objects
	ObjectType string `json:"objectType" yaml:"objectType"`
}

// StringArray ...
type StringArray struct {
	Array []string `json:"array" yaml:"array"`
}

// NewProvider creates a new Azure Key Vault Provider.
func NewProvider() (*Provider, error) {
	log.Debugf("NewAzureProvider")
	var p Provider
	return &p, nil
}

// ParseAzureEnvironment returns azure environment by name
func ParseAzureEnvironment(cloudName string) (*azure.Environment, error) {
	var env azure.Environment
	var err error
	if cloudName == "" {
		env = azure.PublicCloud
	} else {
		env, err = azure.EnvironmentFromName(cloudName)
	}
	return &env, err
}

// GetKeyvaultToken retrieves a new service principal token to access keyvault
func (p *Provider) GetKeyvaultToken(grantType OAuthGrantType, cloudName string) (authorizer autorest.Authorizer, err error) {
	env, err := ParseAzureEnvironment(cloudName)
	if err != nil {
		return nil, err
	}

	kvEndPoint := env.KeyVaultEndpoint
	if '/' == kvEndPoint[len(kvEndPoint)-1] {
		kvEndPoint = kvEndPoint[:len(kvEndPoint)-1]
	}

	servicePrincipalToken, err := p.GetServicePrincipalToken(env, kvEndPoint)
	if err != nil {
		return nil, err
	}
	authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	return authorizer, nil
}

func (p *Provider) initializeKvClient(cloudName string) (*kv.BaseClient, error) {
	kvClient := kv.New()
	token, err := p.GetKeyvaultToken(AuthGrantType(), cloudName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get key vault token")
	}

	kvClient.Authorizer = token
	return &kvClient, nil
}

// GetCredential gets clientid and clientsecret
func GetCredential(secrets map[string]string) (string, string, error) {
	if secrets == nil {
		return "", "", fmt.Errorf("unexpected: getCredential failed, nodePublishSecretRef secret is not provided")
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

func (p *Provider) getVaultURL(ctx context.Context, cloudName string) (vaultURL *string, err error) {
	log.Debugf("vaultName: %s", p.KeyvaultName)

	// Key Vault name must be a 3-24 character string
	if len(p.KeyvaultName) < 3 || len(p.KeyvaultName) > 24 {
		return nil, errors.Errorf("Invalid vault name: %q, must be between 3 and 24 chars", p.KeyvaultName)
	}
	// See docs for validation spec: https://docs.microsoft.com/en-us/azure/key-vault/about-keys-secrets-and-certificates#objects-identifiers-and-versioning
	isValid := regexp.MustCompile(`^[-A-Za-z0-9]+$`).MatchString
	if !isValid(p.KeyvaultName) {
		return nil, errors.Errorf("Invalid vault name: %q, must match [-a-zA-Z0-9]{3,24}", p.KeyvaultName)
	}

	vaultDnsSuffix, err := GetVaultDNSSuffix(cloudName)
	if err != nil {
		return nil, err
	}
	vaultDnsSuffixValue := *vaultDnsSuffix
	vaultUri := "https://" + p.KeyvaultName + "." + vaultDnsSuffixValue + "/"
	return &vaultUri, nil
}

// GetServicePrincipalToken creates a new service principal token based on the configuration
func (p *Provider) GetServicePrincipalToken(env *azure.Environment, resource string) (*adal.ServicePrincipalToken, error) {
	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, p.TenantID)
	if err != nil {
		return nil, fmt.Errorf("creating the OAuth config: %v", err)
	}

	// For usepodidentity mode, the CSI driver makes an authorization request to fetch token for a resource from the NMI host endpoint (http://127.0.0.1:2579/host/token/).
	// The request includes the pod namespace `podns` and the pod name `podname` in the request header and the resource endpoint of the resource requesting the token.
	// The NMI server identifies the pod based on the `podns` and `podname` in the request header and then queries k8s (through MIC) for a matching azure identity.
	// Then nmi makes an adal request to get a token for the resource in the request, returns the `token` and the `clientid` as a response to the CSI request.

	if p.UsePodIdentity {
		log.Infof("azure: using pod identity to retrieve token")

		endpoint := fmt.Sprintf("%s?resource=%s", nmiendpoint, resource)
		client := &http.Client{}
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Add(podnsheader, p.PodNamespace)
		req.Header.Add(podnameheader, p.PodName)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			var nmiResp = new(NMIResponse)
			err = json.Unmarshal(bodyBytes, &nmiResp)
			if err != nil {
				return nil, err
			}

			log.Infof("accesstoken: %s", RedacteClientID(nmiResp.Token.AccessToken))
			log.Infof("clientid: %s", RedacteClientID(nmiResp.ClientID))

			token := nmiResp.Token
			clientID := nmiResp.ClientID

			if token.AccessToken == "" || clientID == "" {
				return nil, fmt.Errorf("nmi did not return expected values in response: token and clientid")
			}

			spt, err := adal.NewServicePrincipalTokenFromManualToken(*oauthConfig, clientID, resource, token, nil)
			if err != nil {
				return nil, err
			}
			return spt, nil
		}

		err = fmt.Errorf("nmi response failed with status code: %d", resp.StatusCode)
		return nil, err
	}

	if p.UseVMManagedIdentity {
		msiEndpoint, err := adal.GetMSIVMEndpoint()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get managed identity (MSI) endpoint")
		}

		if p.UserAssignedIdentityID != "" {
			log.Infof("azure: using user assigned managed identity %s to retrieve access token", RedacteClientID(p.UserAssignedIdentityID))
			return adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(
				msiEndpoint,
				resource,
				p.UserAssignedIdentityID)
		}

		log.Infof("azure: using system assigned managed identity to retrieve access token")
		return adal.NewServicePrincipalTokenFromMSI(
			msiEndpoint,
			resource)
	}

	// When CSI driver is using a Service Principal clientid + client secret to retrieve token for resource
	if len(p.AADClientSecret) > 0 {
		log.Infof("azure: using client_id+client_secret to retrieve access token")
		return adal.NewServicePrincipalToken(
			*oauthConfig,
			p.AADClientID,
			p.AADClientSecret,
			resource)
	}
	return nil, fmt.Errorf("No credentials provided for AAD application %s", p.AADClientID)
}

// MountSecretsStoreObjectContent mounts content of the secrets store object to target path
func (p *Provider) MountSecretsStoreObjectContent(ctx context.Context, attrib map[string]string, secrets map[string]string, targetPath string, permission os.FileMode) (err error) {
	keyvaultName := attrib["keyvaultName"]
	usePodIdentityStr := attrib["usePodIdentity"]
	useVMManagedIdentityStr := attrib["useVMManagedIdentity"]
	userAssignedIdentityID := attrib["userAssignedIdentityID"]
	tenantID := attrib["tenantId"]
	p.PodName = attrib["csi.storage.k8s.io/pod.name"]
	p.PodNamespace = attrib["csi.storage.k8s.io/pod.namespace"]

	if keyvaultName == "" {
		return fmt.Errorf("keyvaultName is not set")
	}
	if tenantID == "" {
		return fmt.Errorf("tenantId is not set")
	}
	// defaults
	usePodIdentity := false
	if usePodIdentityStr == "true" {
		usePodIdentity = true
	}

	useVMManagedIdentity := false
	if useVMManagedIdentityStr == "true" {
		useVMManagedIdentity = true
	}

	if usePodIdentity && useVMManagedIdentity {
		return fmt.Errorf("cannot enable both pod identity and assigned user identity")
	}

	log.Infof("mounting secrets store object content for %s/%s", p.PodNamespace, p.PodName)
	if !usePodIdentity && !useVMManagedIdentity {
		log.Infof("not using pod identity or vm assigned user identity to access keyvault")
		p.AADClientID, p.AADClientSecret, err = GetCredential(secrets)
		if err != nil {
			log.Infof("missing client credential to access keyvault")
			return err
		}
	}
	if usePodIdentity {
		log.Infof("using pod identity to access keyvault")
		if p.PodName == "" || p.PodNamespace == "" {
			return fmt.Errorf("pod information is not available. deploy a CSIDriver object to set podInfoOnMount")
		}
	} else if useVMManagedIdentity {
		log.Infof("using vm managed identity to access keyvault")
	}

	objectsStrings := attrib["objects"]
	if objectsStrings == "" {
		return fmt.Errorf("objects is not set")
	}
	log.Infof("objects: %s", objectsStrings)

	var objects StringArray
	err = yaml.Unmarshal([]byte(objectsStrings), &objects)
	if err != nil {
		log.Infof("unmarshal failed for objects")
		return err
	}
	log.Debugf("objects array: %v", objects.Array)
	keyVaultObjects := []KeyVaultObject{}
	for i, object := range objects.Array {
		var keyVaultObject KeyVaultObject
		err = yaml.Unmarshal([]byte(object), &keyVaultObject)
		if err != nil {
			log.Infof("unmarshal failed for keyVaultObjects at index %d", i)
			return err
		}
		keyVaultObjects = append(keyVaultObjects, keyVaultObject)
	}

	log.Infof("unmarshaled keyVaultObjects: %v", keyVaultObjects)
	log.Infof("keyVaultObjects len: %d", len(keyVaultObjects))

	if len(keyVaultObjects) == 0 {
		return fmt.Errorf("objects array is empty")
	}
	p.KeyvaultName = keyvaultName
	p.UsePodIdentity = usePodIdentity
	p.UseVMManagedIdentity = useVMManagedIdentity
	p.UserAssignedIdentityID = userAssignedIdentityID
	p.TenantID = tenantID

	for _, keyVaultObject := range keyVaultObjects {
		content, err := p.GetKeyVaultObjectContent(ctx, keyVaultObject.ObjectType, keyVaultObject.ObjectName, keyVaultObject.ObjectVersion)
		if err != nil {
			return err
		}
		objectContent := []byte(content)
		fileName := keyVaultObject.ObjectName
		if keyVaultObject.ObjectAlias != "" {
			fileName = keyVaultObject.ObjectAlias
		}
		if err := ioutil.WriteFile(filepath.Join(targetPath, fileName), objectContent, permission); err != nil {
			return errors.Wrapf(err, "secrets store csi driver failed to mount %s at %s", fileName, targetPath)
		}
		log.Infof("secrets store csi driver mounted %s", fileName)
		log.Infof("Mount point: %s", targetPath)
	}

	return nil
}

// GetKeyVaultObjectContent get content of the keyvault object
func (p *Provider) GetKeyVaultObjectContent(ctx context.Context, objectType string, objectName string, objectVersion string) (content string, err error) {
	vaultURL, err := p.getVaultURL(ctx, "")
	if err != nil {
		return "", errors.Wrap(err, "failed to get vault")
	}

	kvClient, err := p.initializeKvClient("")
	if err != nil {
		return "", errors.Wrap(err, "failed to get keyvaultClient")
	}

	switch objectType {
	case VaultObjectTypeSecret:
		secret, err := kvClient.GetSecret(ctx, *vaultURL, objectName, objectVersion)
		if err != nil {
			return "", wrapObjectTypeError(err, objectType, objectName, objectVersion)
		}
		return *secret.Value, nil
	case VaultObjectTypeKey:
		keybundle, err := kvClient.GetKey(ctx, *vaultURL, objectName, objectVersion)
		if err != nil {
			return "", wrapObjectTypeError(err, objectType, objectName, objectVersion)
		}
		// NOTE: we are writing the RSA modulus content of the key
		return *keybundle.Key.N, nil
	case VaultObjectTypeCertificate:
		certbundle, err := kvClient.GetCertificate(ctx, *vaultURL, objectName, objectVersion)
		if err != nil {
			return "", wrapObjectTypeError(err, objectType, objectName, objectVersion)
		}
		return string(*certbundle.Cer), nil
	default:
		err := errors.Errorf("Invalid vaultObjectTypes. Should be secret, key, or cert")
		return "", wrapObjectTypeError(err, objectType, objectName, objectVersion)
	}
}

func wrapObjectTypeError(err error, objectType string, objectName string, objectVersion string) error {
	return errors.Wrapf(err, "failed to get objectType:%s, objectName:%s, objectVersion:%s", objectType, objectName, objectVersion)
}

func GetVaultDNSSuffix(cloudName string) (vaultTld *string, err error) {
	environment, err := ParseAzureEnvironment(cloudName)
	if err != nil {
		return nil, err
	}

	return &environment.KeyVaultDNSSuffix, nil
}

//RedacteString Apply regex to a sensitive string and return the redacted value
func RedacteClientID(sensitiveString string) string {
	r, _ := regexp.Compile(`^(\S{4})(\S|\s)*(\S{4})$`)
	return r.ReplaceAllString(sensitiveString, "$1##### REDACTED #####$3")
}
