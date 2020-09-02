package azure

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/crypto/pkcs12"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	yaml "gopkg.in/yaml.v2"

	kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/version"

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
	podnsheader     = "podns"
	certTypePem     = "application/x-pem-file"
	certTypePfx     = "application/x-pkcs12"
	certificateType = "CERTIFICATE"
	objectFormatPEM = "pem"
	objectFormatPFX = "pfx"
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
	// the type of azure cloud based on azure go sdk
	AzureCloudEnvironment *azure.Environment
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
	// EnvironmentFilepathName captures the name of the environment variable containing the path to the file
	// to be used while populating the Azure Environment.
	EnvironmentFilepathName string
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
	// the format of the Azure Key Vault objects
	// supported formats are PEM, PFX
	ObjectFormat string `json:"objectFormat" yaml:"objectFormat"`
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
func (p *Provider) GetKeyvaultToken(grantType OAuthGrantType) (authorizer autorest.Authorizer, err error) {
	err = adal.AddToUserAgent(version.GetUserAgent())
	if err != nil {
		return nil, errors.Wrap(err, "failed to add user agent to adal")
	}
	kvEndPoint := p.AzureCloudEnvironment.KeyVaultEndpoint
	if '/' == kvEndPoint[len(kvEndPoint)-1] {
		kvEndPoint = kvEndPoint[:len(kvEndPoint)-1]
	}
	servicePrincipalToken, err := p.GetServicePrincipalToken(kvEndPoint)
	if err != nil {
		return nil, err
	}
	authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)
	return authorizer, nil
}

func (p *Provider) initializeKvClient() (*kv.BaseClient, error) {
	kvClient := kv.New()
	err := kvClient.AddToUserAgent(version.GetUserAgent())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add user agent to keyvault client")
	}
	token, err := p.GetKeyvaultToken(AuthGrantType())
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

func (p *Provider) getVaultURL(ctx context.Context) (vaultURL *string, err error) {
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

	vaultDnsSuffixValue := p.AzureCloudEnvironment.KeyVaultDNSSuffix
	vaultUri := "https://" + p.KeyvaultName + "." + vaultDnsSuffixValue + "/"
	return &vaultUri, nil
}

// GetServicePrincipalToken creates a new service principal token based on the configuration
func (p *Provider) GetServicePrincipalToken(resource string) (*adal.ServicePrincipalToken, error) {
	oauthConfig, err := adal.NewOAuthConfig(p.AzureCloudEnvironment.ActiveDirectoryEndpoint, p.TenantID)
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

			log.Infof("accesstoken: %s", RedactClientID(nmiResp.Token.AccessToken))
			log.Infof("clientid: %s", RedactClientID(nmiResp.ClientID))

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
			log.Infof("azure: using user assigned managed identity %s to retrieve access token", RedactClientID(p.UserAssignedIdentityID))
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
	cloudName := attrib["cloudName"]
	usePodIdentityStr := attrib["usePodIdentity"]
	useVMManagedIdentityStr := attrib["useVMManagedIdentity"]
	userAssignedIdentityID := attrib["userAssignedIdentityID"]
	tenantID := attrib["tenantId"]
	p.PodName = attrib["csi.storage.k8s.io/pod.name"]
	p.PodNamespace = attrib["csi.storage.k8s.io/pod.namespace"]
	cloudEnvFileName := attrib["cloudEnvFileName"]

	if keyvaultName == "" {
		return fmt.Errorf("keyvaultName is not set")
	}
	if tenantID == "" {
		return fmt.Errorf("tenantId is not set")
	}

	err = setAzureEnvironmentFilePath(cloudEnvFileName)
	if err != nil {
		return fmt.Errorf("failed to set AZURE_ENVIRONMENT_FILEPATH env to %s, error %+v", cloudEnvFileName, err)
	}
	azureCloudEnv, err := ParseAzureEnvironment(cloudName)
	if err != nil {
		return fmt.Errorf("cloudName %s is not valid, error: %v", cloudName, err)
	}

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
		log.Infof("mounting secrets store object content for %s/%s", p.PodNamespace, p.PodName)
	} else if useVMManagedIdentity {
		log.Infof("using vm managed identity to access keyvault")
	}
	if useVMManagedIdentity {
		log.Infof("using vmss user identity to access keyvault")
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
	p.AzureCloudEnvironment = azureCloudEnv
	p.UsePodIdentity = usePodIdentity
	p.UseVMManagedIdentity = useVMManagedIdentity
	p.UserAssignedIdentityID = userAssignedIdentityID
	p.TenantID = tenantID

	for _, keyVaultObject := range keyVaultObjects {
		log.Infof("fetching object: %s, type: %s from key vault", keyVaultObject.ObjectName, keyVaultObject.ObjectType)
		if err := validateObjectFormat(keyVaultObject.ObjectFormat, keyVaultObject.ObjectType); err != nil {
			return wrapObjectTypeError(err, keyVaultObject.ObjectType, keyVaultObject.ObjectName, keyVaultObject.ObjectVersion)
		}
		content, err := p.GetKeyVaultObjectContent(ctx, keyVaultObject)
		if err != nil {
			return err
		}
		objectContent := []byte(content)
		fileName := keyVaultObject.ObjectName
		if keyVaultObject.ObjectAlias != "" {
			fileName = keyVaultObject.ObjectAlias
		}
		if err := ioutil.WriteFile(filepath.Join(targetPath, fileName), objectContent, permission); err != nil {
			return errors.Wrapf(err, "failed to mount %s at %s", fileName, targetPath)
		}
		log.Infof("successfully mounted %s", fileName)
	}

	return nil
}

// GetKeyVaultObjectContent get content of the keyvault object
func (p *Provider) GetKeyVaultObjectContent(ctx context.Context, kvObject KeyVaultObject) (content string, err error) {
	vaultURL, err := p.getVaultURL(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to get vault")
	}

	kvClient, err := p.initializeKvClient()
	if err != nil {
		return "", errors.Wrap(err, "failed to get keyvaultClient")
	}

	switch kvObject.ObjectType {
	case VaultObjectTypeSecret:
		secret, err := kvClient.GetSecret(ctx, *vaultURL, kvObject.ObjectName, kvObject.ObjectVersion)
		if err != nil {
			return "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
		content := *secret.Value
		// if the secret is part of a certificate, then we need to convert the certificate and key to PEM format
		if secret.Kid != nil && len(*secret.Kid) > 0 {
			switch *secret.ContentType {
			case certTypePem:
				return content, nil
			case certTypePfx:
				// object format requested is pfx, then return the content as is
				if strings.EqualFold(kvObject.ObjectFormat, objectFormatPFX) {
					return content, err
				}
				// convert to pem as that's the default object format for this provider
				content, err := decodePKCS12(*secret.Value)
				if err != nil {
					return "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
				}
				return content, nil
			default:
				err := errors.Errorf("failed to get certificate. unknown content type '%s'", *secret.ContentType)
				return "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
			}
		}
		return content, nil
	case VaultObjectTypeKey:
		keybundle, err := kvClient.GetKey(ctx, *vaultURL, kvObject.ObjectName, kvObject.ObjectVersion)
		if err != nil {
			return "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
		// for object type "key" the public key is written to the file in PEM format
		switch keybundle.Key.Kty {
		case kv.RSA:
			// decode the base64 bytes for n
			nb, err := base64.RawURLEncoding.DecodeString(*keybundle.Key.N)
			if err != nil {
				return "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
			}
			// decode the base64 bytes for e
			eb, err := base64.RawURLEncoding.DecodeString(*keybundle.Key.E)
			if err != nil {
				return "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
			}
			e := new(big.Int).SetBytes(eb).Int64()
			pKey := &rsa.PublicKey{
				N: new(big.Int).SetBytes(nb),
				E: int(e),
			}
			derBytes, err := x509.MarshalPKIXPublicKey(pKey)
			if err != nil {
				return "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
			}
			pubKeyBlock := &pem.Block{
				Type:  "PUBLIC KEY",
				Bytes: derBytes,
			}
			var pemData []byte
			pemData = append(pemData, pem.EncodeToMemory(pubKeyBlock)...)
			return string(pemData), nil
		case kv.EC:
			// decode the base64 bytes for x
			xb, err := base64.RawURLEncoding.DecodeString(*keybundle.Key.X)
			if err != nil {
				return "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
			}
			// decode the base64 bytes for y
			yb, err := base64.RawURLEncoding.DecodeString(*keybundle.Key.Y)
			if err != nil {
				return "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
			}
			crv, err := getCurve(keybundle.Key.Crv)
			if err != nil {
				return "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
			}
			pKey := &ecdsa.PublicKey{
				X:     new(big.Int).SetBytes(xb),
				Y:     new(big.Int).SetBytes(yb),
				Curve: crv,
			}
			derBytes, err := x509.MarshalPKIXPublicKey(pKey)
			if err != nil {
				return "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
			}
			pubKeyBlock := &pem.Block{
				Type:  "PUBLIC KEY",
				Bytes: derBytes,
			}
			var pemData []byte
			pemData = append(pemData, pem.EncodeToMemory(pubKeyBlock)...)
			return string(pemData), nil
		default:
			err := errors.Errorf("failed to get key. key type '%s' currently not supported", keybundle.Key.Kty)
			return "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
	case VaultObjectTypeCertificate:
		// for object type "cert" the certificate is written to the file in PEM format
		certbundle, err := kvClient.GetCertificate(ctx, *vaultURL, kvObject.ObjectName, kvObject.ObjectVersion)
		if err != nil {
			return "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
		certBlock := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: *certbundle.Cer,
		}
		var pemData []byte
		pemData = append(pemData, pem.EncodeToMemory(certBlock)...)
		return string(pemData), nil
	default:
		err := errors.Errorf("Invalid vaultObjectTypes. Should be secret, key, or cert")
		return "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}
}

func wrapObjectTypeError(err error, objectType, objectName, objectVersion string) error {
	return errors.Wrapf(err, "failed to get objectType:%s, objectName:%s, objectVersion:%s", objectType, objectName, objectVersion)
}

//RedactClientID Apply regex to a sensitive string and return the redacted value
func RedactClientID(sensitiveString string) string {
	r, _ := regexp.Compile(`^(\S{4})(\S|\s)*(\S{4})$`)
	return r.ReplaceAllString(sensitiveString, "$1##### REDACTED #####$3")
}

// decodePkcs12 decodes PKCS#12 client certificates by extracting the public certificates, the private
// keys and converts it to PEM format
func decodePKCS12(value string) (content string, err error) {
	pfxRaw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	// using ToPEM to extract more than one certificate and key in pfxData
	pemBlock, err := pkcs12.ToPEM(pfxRaw, "")
	if err != nil {
		return "", err
	}

	var pemKeyData, pemCertData, pemData []byte
	for _, block := range pemBlock {
		// PEM block encoded form contains the headers
		//    -----BEGIN Type-----
		//    Headers
		//    base64-encoded Bytes
		//    -----END Type-----
		// Setting headers to nil to ensure no headers included in the encoded block
		block.Headers = make(map[string]string)
		if block.Type == certificateType {
			pemCertData = append(pemCertData, pem.EncodeToMemory(block)...)
		} else {
			key, err := parsePrivateKey(block.Bytes)
			if err != nil {
				return "", err
			}
			// pkcs1 RSA private key PEM file is specific for RSA keys. RSA is not used exclusively inside X509
			// and SSL/TLS, a more generic key format is available in the form of PKCS#8 that identifies the type
			// of private key and contains the relevant data.
			// Converting to pkcs8 private key as ToPEM uses pkcs1
			// The driver determines the key type from the pkcs8 form of the key and marshals appropriately
			block.Bytes, err = x509.MarshalPKCS8PrivateKey(key)
			if err != nil {
				return "", err
			}
			pemKeyData = append(pemKeyData, pem.EncodeToMemory(block)...)
		}
	}
	pemData = append(pemData, pemKeyData...)
	pemData = append(pemData, pemCertData...)
	return string(pemData), nil
}

func getCurve(crv kv.JSONWebKeyCurveName) (elliptic.Curve, error) {
	switch crv {
	case kv.P256:
		return elliptic.P256(), nil
	case kv.P384:
		return elliptic.P384(), nil
	case kv.P521:
		return elliptic.P521(), nil
	default:
		return nil, fmt.Errorf("curve %s is not suppported", crv)
	}
}

func parsePrivateKey(block []byte) (interface{}, error) {
	if key, err := x509.ParsePKCS1PrivateKey(block); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(block); err == nil {
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(block); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("failed to parse key for type pkcs1, pkcs8 or ec")
}

// setAzureEnvironmentFilePath sets the AZURE_ENVIRONMENT_FILEPATH env var which is used by
// go-autorest for AZURESTACKCLOUD
func setAzureEnvironmentFilePath(envFileName string) error {
	if envFileName == "" {
		return nil
	}
	log.Infof("setting AZURE_ENVIRONMENT_FILEPATH to %s for custom cloud", envFileName)
	return os.Setenv(azure.EnvironmentFilepathName, envFileName)
}

// validateObjectFormat checks if the object format is valid and is supported
// for the given object type
func validateObjectFormat(objectFormat, objectType string) error {
	if len(objectFormat) == 0 {
		return nil
	}
	if !strings.EqualFold(objectFormat, objectFormatPEM) && !strings.EqualFold(objectFormat, objectFormatPFX) {
		return fmt.Errorf("Invalid objectFormat: %v, should be PEM or PFX", objectFormat)
	}
	// Azure Key Vault returns the base64 encoded binary content only for type secret
	// for types cert/key, the content is always in pem format
	if objectFormat == objectFormatPFX && objectType != VaultObjectTypeSecret {
		return fmt.Errorf("PFX format only supported for objectType: secret")
	}
	return nil
}
