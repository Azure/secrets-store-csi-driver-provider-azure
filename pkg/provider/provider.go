package provider

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/auth"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/metrics"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/types"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/version"

	kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/pkg/errors"
	"golang.org/x/crypto/pkcs12"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

var (
	ConstructPEMChain = flag.Bool("construct-pem-chain", true, "explicitly reconstruct the pem chain in the order: SERVER, INTERMEDIATE, ROOT")
)

// Provider implements the secrets-store-csi-driver provider interface
type Provider struct {
	reporter metrics.StatsReporter
}

// mountConfig holds the information for the mount event
type mountConfig struct {
	// the name of the Azure Key Vault instance
	keyvaultName string
	// the type of azure cloud based on azure go sdk
	azureCloudEnvironment *azure.Environment
	// authConfig is the config parameters for accessing Key Vault
	authConfig auth.Config
	// tenantID in AAD
	tenantID string
	// podName is the pod name
	podName string
	// podNamespace is the pod namespace
	podNamespace string
}

// NewProvider creates a new provider
func NewProvider() *Provider {
	return &Provider{
		reporter: metrics.NewStatsReporter(),
	}
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

func (mc *mountConfig) initializeKvClient(ctx context.Context) (*kv.BaseClient, error) {
	kvClient := kv.New()
	kvEndpoint := strings.TrimSuffix(mc.azureCloudEnvironment.KeyVaultEndpoint, "/")

	err := kvClient.AddToUserAgent(version.GetUserAgent())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add user agent to keyvault client")
	}

	kvClient.Authorizer, err = mc.GetAuthorizer(ctx, kvEndpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get authorizer for keyvault client")
	}
	return &kvClient, nil
}

func (mc *mountConfig) getVaultURL() (vaultURL *string, err error) {
	// Key Vault name must be a 3-24 character string
	if len(mc.keyvaultName) < 3 || len(mc.keyvaultName) > 24 {
		return nil, errors.Errorf("Invalid vault name: %q, must be between 3 and 24 chars", mc.keyvaultName)
	}
	// See docs for validation spec: https://docs.microsoft.com/en-us/azure/key-vault/about-keys-secrets-and-certificates#objects-identifiers-and-versioning
	isValid := regexp.MustCompile(`^[-A-Za-z0-9]+$`).MatchString
	if !isValid(mc.keyvaultName) {
		return nil, errors.Errorf("Invalid vault name: %q, must match [-a-zA-Z0-9]{3,24}", mc.keyvaultName)
	}

	vaultDNSSuffixValue := mc.azureCloudEnvironment.KeyVaultDNSSuffix
	vaultURI := "https://" + mc.keyvaultName + "." + vaultDNSSuffixValue + "/"
	return &vaultURI, nil
}

// GetAuthorizer returns an Azure authorizer based on the provided azure identity
func (mc *mountConfig) GetAuthorizer(ctx context.Context, resource string) (autorest.Authorizer, error) {
	return mc.authConfig.GetAuthorizer(ctx, mc.podName, mc.podNamespace, resource, mc.azureCloudEnvironment.ActiveDirectoryEndpoint, mc.tenantID, types.PodIdentityNMIPort)
}

// GetSecretsStoreObjectContent gets the objects (secret, key, certificate) from keyvault and returns the content
// to the CSI driver. The driver will write the content to the file system.
func (p *Provider) GetSecretsStoreObjectContent(ctx context.Context, attrib, secrets map[string]string, targetPath string, defaultFilePermission os.FileMode) ([]types.SecretFile, error) {
	keyvaultName := types.GetKeyVaultName(attrib)
	cloudName := types.GetCloudName(attrib)
	userAssignedIdentityID := types.GetUserAssignedIdentityID(attrib)
	tenantID := types.GetTenantID(attrib)
	cloudEnvFileName := types.GetCloudEnvFileName(attrib)
	podName := types.GetPodName(attrib)
	podNamespace := types.GetPodNamespace(attrib)

	usePodIdentity, err := types.GetUsePodIdentity(attrib)
	if err != nil {
		return nil, fmt.Errorf("failed to parse usePodIdentity flag, error: %w", err)
	}
	useVMManagedIdentity, err := types.GetUseVMManagedIdentity(attrib)
	if err != nil {
		return nil, fmt.Errorf("failed to parse useVMManagedIdentity flag, error: %w", err)
	}

	// attributes for workload identity
	workloadIdentityClientID := types.GetClientID(attrib)
	saTokens := types.GetServiceAccountTokens(attrib)

	if keyvaultName == "" {
		return nil, fmt.Errorf("keyvaultName is not set")
	}
	if tenantID == "" {
		return nil, fmt.Errorf("tenantId is not set")
	}

	err = setAzureEnvironmentFilePath(cloudEnvFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to set AZURE_ENVIRONMENT_FILEPATH env to %s, error %w", cloudEnvFileName, err)
	}
	azureCloudEnv, err := ParseAzureEnvironment(cloudName)
	if err != nil {
		return nil, fmt.Errorf("cloudName %s is not valid, error: %w", cloudName, err)
	}

	// parse bound service account tokens for workload identity only if the clientID is set
	var workloadIdentityToken string
	if workloadIdentityClientID != "" {
		if workloadIdentityToken, err = auth.ParseServiceAccountToken(saTokens); err != nil {
			return nil, fmt.Errorf("failed to parse workload identity tokens, error: %w", err)
		}
	}

	authConfig, err := auth.NewConfig(usePodIdentity, useVMManagedIdentity, userAssignedIdentityID, workloadIdentityClientID, workloadIdentityToken, secrets)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth config, error: %w", err)
	}

	mc := &mountConfig{
		keyvaultName:          keyvaultName,
		azureCloudEnvironment: azureCloudEnv,
		authConfig:            authConfig,
		tenantID:              tenantID,
		podName:               podName,
		podNamespace:          podNamespace,
	}

	objectsStrings := types.GetObjects(attrib)
	if objectsStrings == "" {
		return nil, fmt.Errorf("objects is not set")
	}
	klog.V(2).InfoS("objects string defined in secret provider class", "objects", objectsStrings, "pod", klog.ObjectRef{Namespace: podNamespace, Name: podName})

	objects, err := types.GetObjectsArray(objectsStrings)
	if err != nil {
		return nil, fmt.Errorf("failed to yaml unmarshal objects, error: %w", err)
	}
	klog.V(2).InfoS("unmarshaled objects yaml array", "objectsArray", objects.Array, "pod", klog.ObjectRef{Namespace: podNamespace, Name: podName})

	keyVaultObjects := []types.KeyVaultObject{}
	for i, object := range objects.Array {
		var keyVaultObject types.KeyVaultObject
		err = yaml.Unmarshal([]byte(object), &keyVaultObject)
		if err != nil {
			return nil, fmt.Errorf("unmarshal failed for keyVaultObjects at index %d, error: %w", i, err)
		}
		// remove whitespace from all fields in keyVaultObject
		formatKeyVaultObject(&keyVaultObject)
		keyVaultObjects = append(keyVaultObjects, keyVaultObject)
	}

	klog.V(5).InfoS("unmarshaled key vault objects", "keyVaultObjects", keyVaultObjects, "count", len(keyVaultObjects), "pod", klog.ObjectRef{Namespace: podNamespace, Name: podName})

	if len(keyVaultObjects) == 0 {
		return nil, nil
	}

	vaultURL, err := mc.getVaultURL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get vault")
	}
	klog.V(2).InfoS("vault url", "vaultName", mc.keyvaultName, "vaultURL", *vaultURL, "pod", klog.ObjectRef{Namespace: podNamespace, Name: podName})

	// the keyvault name is per SPC and we don't need to recreate the client for every single keyvault object defined
	kvClient, err := mc.initializeKvClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get keyvault client")
	}

	files := []types.SecretFile{}
	for _, keyVaultObject := range keyVaultObjects {
		klog.V(5).InfoS("fetching object from key vault", "objectName", keyVaultObject.ObjectName, "objectType", keyVaultObject.ObjectType, "keyvault", mc.keyvaultName, "pod", klog.ObjectRef{Namespace: podNamespace, Name: podName})
		if err := validateObjectFormat(keyVaultObject.ObjectFormat, keyVaultObject.ObjectType); err != nil {
			return nil, wrapObjectTypeError(err, keyVaultObject.ObjectType, keyVaultObject.ObjectName, keyVaultObject.ObjectVersion)
		}
		if err := validateObjectEncoding(keyVaultObject.ObjectEncoding, keyVaultObject.ObjectType); err != nil {
			return nil, wrapObjectTypeError(err, keyVaultObject.ObjectType, keyVaultObject.ObjectName, keyVaultObject.ObjectVersion)
		}
		filePermission, err := validateFilePermission(keyVaultObject.FilePermission, defaultFilePermission)
		if err != nil {
			return nil, err
		}

		resolvedKvObjects, err := p.resolveObjectVersions(ctx, kvClient, keyVaultObject, *vaultURL)
		if err != nil {
			return nil, err
		}

		for _, resolvedKvObject := range resolvedKvObjects {
			fileName := resolvedKvObject.ObjectName

			if resolvedKvObject.ObjectAlias == "" {
				fileName = resolvedKvObject.ObjectAlias
			}

			if err := validateFileName(fileName); err != nil {
				return nil, wrapObjectTypeError(err, resolvedKvObject.ObjectType, resolvedKvObject.ObjectName, resolvedKvObject.ObjectVersion)
			}

			content, newObjectVersion, err := p.getKeyVaultObjectContent(ctx, kvClient, resolvedKvObject, *vaultURL)
			if err != nil {
				return nil, err
			}

			objectContent, err := getContentBytes(content, resolvedKvObject.ObjectType, resolvedKvObject.ObjectEncoding)
			if err != nil {
				return nil, err
			}

			// objectUID is a unique identifier in the format <object type>/<object name>
			// This is the object id the user sees in the SecretProviderClassPodStatus
			objectUID := getObjectUID(resolvedKvObject.ObjectName, resolvedKvObject.ObjectType)

			// these files will be returned to the CSI driver as part of gRPC response
			files = append(files, types.SecretFile{
				Path:     fileName,
				Content:  objectContent,
				FileMode: filePermission,
				UID:      objectUID,
				Version:  newObjectVersion,
			})
			klog.V(5).InfoS("added file to the gRPC response", "file", fileName, "pod", klog.ObjectRef{Namespace: podNamespace, Name: podName})
		}
	}

	return files, nil
}

func (p *Provider) resolveObjectVersions(ctx context.Context, kvClient *kv.BaseClient, kvObject types.KeyVaultObject, vaultURL string) (versions []types.KeyVaultObject, err error) {
	if kvObject.ObjectVersionHistory <= 1 {
		// version history less than or equal to 1 means only sync the latest and
		// don't add anything to the file name
		return []types.KeyVaultObject{kvObject}, nil
	}

	kvObjectVersions, err := p.getKeyVaultObjectVersions(ctx, kvClient, kvObject, vaultURL)
	if err != nil {
		return nil, err
	}

	return getLatestNKeyVaultObjects(kvObject, kvObjectVersions), nil
}

/*
Given a base key vault object and a list of object versions and their created dates, find
the latest kvObject.ObjectVersionHistory versions and return key vault objects with the
appropriate alias and version.

The alias is determine by the index of the version starting with 0 at the specified version (or
latest if no version is specified).
*/
func getLatestNKeyVaultObjects(kvObject types.KeyVaultObject, kvObjectVersions types.KeyVaultObjectVersionList) []types.KeyVaultObject {
	baseFileName := kvObject.ObjectName
	if kvObject.ObjectAlias != "" {
		baseFileName = kvObject.ObjectAlias
	}

	objects := []types.KeyVaultObject{}

	sort.Sort(kvObjectVersions)

	// if we're being asked for the latest, then there's no need to skip any versions
	var foundFirst = kvObject.ObjectVersion == "" || kvObject.ObjectVersion == "latest"

	for _, objectVersion := range kvObjectVersions {
		foundFirst = foundFirst || objectVersion.Version == kvObject.ObjectVersion

		if foundFirst {
			length := len(objects)
			newObject := kvObject

			newObject.ObjectAlias = fmt.Sprintf("%s/%d", baseFileName, length)
			newObject.ObjectVersion = objectVersion.Version

			objects = append(objects, newObject)

			if length+1 > int(kvObject.ObjectVersionHistory) {
				break
			}
		}
	}

	return objects
}

func (p *Provider) getKeyVaultObjectVersions(ctx context.Context, kvClient *kv.BaseClient, kvObject types.KeyVaultObject, vaultURL string) (versions types.KeyVaultObjectVersionList, err error) {
	start := time.Now()
	defer func() {
		var errMsg string
		if err != nil {
			errMsg = err.Error()
		}
		p.reporter.ReportKeyvaultRequest(ctx, time.Since(start).Seconds(), kvObject.ObjectType, kvObject.ObjectName, errMsg)
	}()

	switch kvObject.ObjectType {
	case types.VaultObjectTypeSecret:
		return getSecretVersions(ctx, kvClient, vaultURL, kvObject)
	case types.VaultObjectTypeKey:
		return getKeyVersions(ctx, kvClient, vaultURL, kvObject)
	case types.VaultObjectTypeCertificate:
		return getCertificateVersions(ctx, kvClient, vaultURL, kvObject)
	default:
		err := errors.Errorf("Invalid vaultObjectTypes. Should be secret, key, or cert")
		return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}
}

func getSecretVersions(ctx context.Context, kvClient *kv.BaseClient, vaultURL string, kvObject types.KeyVaultObject) ([]types.KeyVaultObjectVersion, error) {
	kvVersionsList, err := kvClient.GetSecretVersions(ctx, vaultURL, kvObject.ObjectName, nil)
	if err != nil {
		return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}

	secretVersions := types.KeyVaultObjectVersionList{}

	for notDone := true; notDone; notDone = kvVersionsList.NotDone() {
		for _, secret := range kvVersionsList.Values() {
			objectVersion := getObjectVersion(*secret.ID)
			created := date.UnixEpoch()

			if secret.Attributes != nil {
				created = time.Time(*secret.Attributes.Created)
			}

			if secret.Attributes.Enabled != nil && *secret.Attributes.Enabled {
				secretVersions = append(secretVersions, types.KeyVaultObjectVersion{
					Version: objectVersion,
					Created: created,
				})
			}
		}

		err = kvVersionsList.NextWithContext(ctx)
		if err != nil {
			return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
	}

	return secretVersions, nil
}

func getKeyVersions(ctx context.Context, kvClient *kv.BaseClient, vaultURL string, kvObject types.KeyVaultObject) ([]types.KeyVaultObjectVersion, error) {
	kvVersionsList, err := kvClient.GetKeyVersions(ctx, vaultURL, kvObject.ObjectName, nil)
	if err != nil {
		return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}

	secretVersions := types.KeyVaultObjectVersionList{}

	for notDone := true; notDone; notDone = kvVersionsList.NotDone() {
		for _, secret := range kvVersionsList.Values() {
			objectVersion := getObjectVersion(*secret.Kid)
			created := date.UnixEpoch()

			if secret.Attributes != nil {
				created = time.Time(*secret.Attributes.Created)
			}

			if secret.Attributes.Enabled != nil && *secret.Attributes.Enabled {
				secretVersions = append(secretVersions, types.KeyVaultObjectVersion{
					Version: objectVersion,
					Created: created,
				})
			}
		}

		err = kvVersionsList.NextWithContext(ctx)
		if err != nil {
			return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
	}

	return secretVersions, nil
}

func getCertificateVersions(ctx context.Context, kvClient *kv.BaseClient, vaultURL string, kvObject types.KeyVaultObject) ([]types.KeyVaultObjectVersion, error) {
	kvVersionsList, err := kvClient.GetCertificateVersions(ctx, vaultURL, kvObject.ObjectName, nil)
	if err != nil {
		return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}

	secretVersions := types.KeyVaultObjectVersionList{}

	for notDone := true; notDone; notDone = kvVersionsList.NotDone() {
		for _, secret := range kvVersionsList.Values() {
			objectVersion := getObjectVersion(*secret.ID)
			created := date.UnixEpoch()

			if secret.Attributes != nil {
				created = time.Time(*secret.Attributes.Created)
			}

			if secret.Attributes.Enabled != nil && *secret.Attributes.Enabled {
				secretVersions = append(secretVersions, types.KeyVaultObjectVersion{
					Version: objectVersion,
					Created: created,
				})
			}
		}

		err = kvVersionsList.NextWithContext(ctx)
		if err != nil {
			return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
	}

	return secretVersions, nil
}

// GetKeyVaultObjectContent get content of the keyvault object
func (p *Provider) getKeyVaultObjectContent(ctx context.Context, kvClient *kv.BaseClient, kvObject types.KeyVaultObject, vaultURL string) (content, version string, err error) {
	start := time.Now()
	defer func() {
		var errMsg string
		if err != nil {
			errMsg = err.Error()
		}
		p.reporter.ReportKeyvaultRequest(ctx, time.Since(start).Seconds(), kvObject.ObjectType, kvObject.ObjectName, errMsg)
	}()

	switch kvObject.ObjectType {
	case types.VaultObjectTypeSecret:
		return getSecret(ctx, kvClient, vaultURL, kvObject)
	case types.VaultObjectTypeKey:
		return getKey(ctx, kvClient, vaultURL, kvObject)
	case types.VaultObjectTypeCertificate:
		return getCertificate(ctx, kvClient, vaultURL, kvObject)
	default:
		err := errors.Errorf("Invalid vaultObjectTypes. Should be secret, key, or cert")
		return "", "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}
}

// getSecret retrieves the secret from the vault
func getSecret(ctx context.Context, kvClient *kv.BaseClient, vaultURL string, kvObject types.KeyVaultObject) (string, string, error) {
	secret, err := kvClient.GetSecret(ctx, vaultURL, kvObject.ObjectName, kvObject.ObjectVersion)
	if err != nil {
		return "", "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}
	if secret.Value == nil {
		return "", "", errors.Errorf("secret value is nil")
	}
	if secret.ID == nil {
		return "", "", errors.Errorf("secret id is nil")
	}
	content := *secret.Value
	version := getObjectVersion(*secret.ID)
	// if the secret is part of a certificate, then we need to convert the certificate and key to PEM format
	if secret.Kid != nil && len(*secret.Kid) > 0 {
		switch *secret.ContentType {
		case types.CertTypePem:
			return content, version, nil
		case types.CertTypePfx:
			// object format requested is pfx, then return the content as is
			if strings.EqualFold(kvObject.ObjectFormat, types.ObjectFormatPFX) {
				return content, version, err
			}
			// convert to pem as that's the default object format for this provider
			content, err := decodePKCS12(*secret.Value)
			if err != nil {
				return "", "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
			}
			return content, version, nil
		default:
			err := errors.Errorf("failed to get certificate. unknown content type '%s'", *secret.ContentType)
			return "", "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
	}
	return content, version, nil
}

// getKey retrieves the key from the vault
func getKey(ctx context.Context, kvClient *kv.BaseClient, vaultURL string, kvObject types.KeyVaultObject) (string, string, error) {
	keybundle, err := kvClient.GetKey(ctx, vaultURL, kvObject.ObjectName, kvObject.ObjectVersion)
	if err != nil {
		return "", "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}
	if keybundle.Key == nil {
		return "", "", errors.Errorf("key value is nil")
	}
	if keybundle.Key.Kid == nil {
		return "", "", errors.Errorf("key id is nil")
	}
	version := getObjectVersion(*keybundle.Key.Kid)
	// for object type "key" the public key is written to the file in PEM format
	switch keybundle.Key.Kty {
	case kv.RSA, kv.RSAHSM:
		// decode the base64 bytes for n
		nb, err := base64.RawURLEncoding.DecodeString(*keybundle.Key.N)
		if err != nil {
			return "", "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
		// decode the base64 bytes for e
		eb, err := base64.RawURLEncoding.DecodeString(*keybundle.Key.E)
		if err != nil {
			return "", "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
		e := new(big.Int).SetBytes(eb).Int64()
		pKey := &rsa.PublicKey{
			N: new(big.Int).SetBytes(nb),
			E: int(e),
		}
		derBytes, err := x509.MarshalPKIXPublicKey(pKey)
		if err != nil {
			return "", "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
		pubKeyBlock := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: derBytes,
		}
		var pemData []byte
		pemData = append(pemData, pem.EncodeToMemory(pubKeyBlock)...)
		return string(pemData), version, nil
	case kv.EC, kv.ECHSM:
		// decode the base64 bytes for x
		xb, err := base64.RawURLEncoding.DecodeString(*keybundle.Key.X)
		if err != nil {
			return "", "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
		// decode the base64 bytes for y
		yb, err := base64.RawURLEncoding.DecodeString(*keybundle.Key.Y)
		if err != nil {
			return "", "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
		crv, err := getCurve(keybundle.Key.Crv)
		if err != nil {
			return "", "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
		pKey := &ecdsa.PublicKey{
			X:     new(big.Int).SetBytes(xb),
			Y:     new(big.Int).SetBytes(yb),
			Curve: crv,
		}
		derBytes, err := x509.MarshalPKIXPublicKey(pKey)
		if err != nil {
			return "", "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
		pubKeyBlock := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: derBytes,
		}
		var pemData []byte
		pemData = append(pemData, pem.EncodeToMemory(pubKeyBlock)...)
		return string(pemData), version, nil
	default:
		err := errors.Errorf("failed to get key. key type '%s' currently not supported", keybundle.Key.Kty)
		return "", "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}
}

// getCertificate retrieves the certificate from the vault
func getCertificate(ctx context.Context, kvClient *kv.BaseClient, vaultURL string, kvObject types.KeyVaultObject) (string, string, error) {
	// for object type "cert" the certificate is written to the file in PEM format
	certbundle, err := kvClient.GetCertificate(ctx, vaultURL, kvObject.ObjectName, kvObject.ObjectVersion)
	if err != nil {
		return "", "", wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}
	if certbundle.Cer == nil {
		return "", "", errors.Errorf("certificate value is nil")
	}
	if certbundle.ID == nil {
		return "", "", errors.Errorf("certificate id is nil")
	}
	version := getObjectVersion(*certbundle.ID)

	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: *certbundle.Cer,
	}
	var pemData []byte
	pemData = append(pemData, pem.EncodeToMemory(certBlock)...)
	return string(pemData), version, nil
}

func wrapObjectTypeError(err error, objectType, objectName, objectVersion string) error {
	return errors.Wrapf(err, "failed to get objectType:%s, objectName:%s, objectVersion:%s", objectType, objectName, objectVersion)
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
		if block.Type == types.CertificateType {
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

	// construct the pem chain in the order
	// SERVER, INTERMEDIATE, ROOT
	if *ConstructPEMChain {
		pemCertData, err = fetchCertChains(pemCertData)
		if err != nil {
			return "", err
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
	klog.V(5).InfoS("setting AZURE_ENVIRONMENT_FILEPATH for custom cloud", "fileName", envFileName)
	return os.Setenv(azure.EnvironmentFilepathName, envFileName)
}

// validateObjectFormat checks if the object format is valid and is supported
// for the given object type
func validateObjectFormat(objectFormat, objectType string) error {
	if len(objectFormat) == 0 {
		return nil
	}
	if !strings.EqualFold(objectFormat, types.ObjectFormatPEM) && !strings.EqualFold(objectFormat, types.ObjectFormatPFX) {
		return fmt.Errorf("invalid objectFormat: %v, should be PEM or PFX", objectFormat)
	}
	// Azure Key Vault returns the base64 encoded binary content only for type secret
	// for types cert/key, the content is always in pem format
	if objectFormat == types.ObjectFormatPFX && objectType != types.VaultObjectTypeSecret {
		return fmt.Errorf("PFX format only supported for objectType: secret")
	}
	return nil
}

// getObjectVersion parses the id to retrieve the version
// of object fetched
// example id format - https://kindkv.vault.azure.net/secrets/actual/1f304204f3624873aab40231241243eb
// TODO (aramase) follow up on https://github.com/Azure/azure-rest-api-specs/issues/10825 to provide
// a native way to obtain the version
func getObjectVersion(id string) string {
	splitID := strings.Split(id, "/")
	return splitID[len(splitID)-1]
}

// getObjectUID returns UID for the object with the format
// <object type>/<object name>
func getObjectUID(objectName, objectType string) string {
	return fmt.Sprintf("%s/%s", objectType, objectName)
}

// validateObjectEncoding checks if the object encoding is valid and is supported
// for the given object type
func validateObjectEncoding(objectEncoding, objectType string) error {
	if len(objectEncoding) == 0 {
		return nil
	}

	// ObjectEncoding is supported only for secret types
	if objectType != types.VaultObjectTypeSecret {
		return fmt.Errorf("objectEncoding only supported for objectType: secret")
	}

	if !strings.EqualFold(objectEncoding, types.ObjectEncodingHex) && !strings.EqualFold(objectEncoding, types.ObjectEncodingBase64) && !strings.EqualFold(objectEncoding, types.ObjectEncodingUtf8) {
		return fmt.Errorf("invalid objectEncoding: %v, should be hex, base64 or utf-8", objectEncoding)
	}

	return nil
}

// getContentBytes takes the given content string and returns the bytes to write to disk
// If an encoding is specified it will decode the string first
func getContentBytes(content, objectType, objectEncoding string) ([]byte, error) {
	if !strings.EqualFold(objectType, types.VaultObjectTypeSecret) || len(objectEncoding) == 0 || strings.EqualFold(objectEncoding, types.ObjectEncodingUtf8) {
		return []byte(content), nil
	}

	if strings.EqualFold(objectEncoding, types.ObjectEncodingBase64) {
		return base64.StdEncoding.DecodeString(content)
	}

	if strings.EqualFold(objectEncoding, types.ObjectEncodingHex) {
		return hex.DecodeString(content)
	}

	return make([]byte, 0), fmt.Errorf("invalid objectEncoding. Should be utf-8, base64, or hex")
}

// formatKeyVaultObject formats the fields in KeyVaultObject
func formatKeyVaultObject(object *types.KeyVaultObject) {
	if object == nil {
		return
	}
	objectPtr := reflect.ValueOf(object)
	objectValue := objectPtr.Elem()

	for i := 0; i < objectValue.NumField(); i++ {
		field := objectValue.Field(i)
		if field.Type() != reflect.TypeOf("") {
			continue
		}
		str := field.Interface().(string)
		str = strings.TrimSpace(str)
		field.SetString(str)
	}
}

// This validate will make sure fileName:
// 1. is not abs path
// 2. does not contain any '..' elements
// 3. does not start with '..'
// These checks have been implemented based on -
// [validateLocalDescendingPath] https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/validation/validation.go#L1158-L1170
// [validatePathNoBacksteps] https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/validation/validation.go#L1172-L1186
func validateFileName(fileName string) error {
	if len(fileName) == 0 {
		return fmt.Errorf("file name must not be empty")
	}
	// is not abs path
	if filepath.IsAbs(fileName) {
		return fmt.Errorf("file name must be a relative path")
	}
	// does not have any element which is ".."
	parts := strings.Split(filepath.ToSlash(fileName), "/")
	for _, item := range parts {
		if item == ".." {
			return fmt.Errorf("file name must not contain '..'")
		}
	}
	// fallback logic if .. is missed in the previous check
	if strings.Contains(fileName, "..") {
		return fmt.Errorf("file name must not contain '..'")
	}
	return nil
}

type node struct {
	cert     *x509.Certificate
	parent   *node
	isParent bool
}

func fetchCertChains(data []byte) ([]byte, error) {
	var newCertChain []*x509.Certificate
	var pemData []byte
	nodes := make([]*node, 0)

	for {
		// decode pem to der first
		block, rest := pem.Decode(data)
		data = rest

		if block == nil {
			break
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return pemData, err
		}
		// this should not be the case because ParseCertificate should return a non nil
		// certificate when there is no error.
		if cert == nil {
			return pemData, fmt.Errorf("certificate is nil")
		}
		nodes = append(nodes, &node{
			cert:     cert,
			parent:   nil,
			isParent: false,
		})
	}

	// at the end of this computation, the output will be a single linked list
	// the tail of the list will be the root node (which has no parents)
	// the head of the list will be the leaf node (whose parent will be intermediate certs)
	// (head) leaf -> intermediates -> root (tail)
	for i := range nodes {
		for j := range nodes {
			// ignore same node to prevent generating a cycle
			if i == j {
				continue
			}
			// if ith node AuthorityKeyId is same as jth node SubjectKeyId, jth node was used
			// to sign the ith certificate
			if string(nodes[i].cert.AuthorityKeyId) == string(nodes[j].cert.SubjectKeyId) {
				nodes[j].isParent = true
				nodes[i].parent = nodes[j]
				break
			}
		}
	}

	var leaf *node
	for i := range nodes {
		if !nodes[i].isParent {
			// this is the leaf node as it's not a parent for any other node
			// TODO (aramase) handle errors if there are more than 1 leaf nodes
			leaf = nodes[i]
			break
		}
	}

	if leaf == nil {
		return nil, fmt.Errorf("no leaf found")
	}

	processedNodes := 0
	// iterate through the directed list and append the nodes to new cert chain
	for leaf != nil {
		processedNodes++
		// ensure we aren't stuck in a cyclic loop
		if processedNodes > len(nodes) {
			return pemData, fmt.Errorf("constructing chain resulted in cycle")
		}
		newCertChain = append(newCertChain, leaf.cert)
		leaf = leaf.parent
	}

	for _, cert := range newCertChain {
		b := &pem.Block{
			Type:  types.CertificateType,
			Bytes: cert.Raw,
		}
		pemData = append(pemData, pem.EncodeToMemory(b)...)
	}
	return pemData, nil
}

// validateFilePermission checks if the given file permission is correct octal number and returns
// a. decimal equivalent of the default file permission (0644) if file permission is not provided Or
// b. decimal equivalent Or
// c. error if it's not valid
func validateFilePermission(filePermission string, defaultFilePermission os.FileMode) (int32, error) {
	if filePermission == "" {
		return int32(defaultFilePermission), nil
	}

	permission, err := strconv.ParseInt(filePermission, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("file permission must be a valid octal number: %w", err)
	}

	return int32(permission), nil
}
