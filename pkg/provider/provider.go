package provider

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
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

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/pkg/errors"
	"golang.org/x/crypto/pkcs12"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

// Provider implements the secrets-store-csi-driver provider interface
type Interface interface {
	GetSecretsStoreObjectContent(ctx context.Context, attrib, secrets map[string]string, defaultFilePermission os.FileMode) ([]types.SecretFile, error)
}

type provider struct {
	reporter metrics.StatsReporter

	constructPEMChain              bool
	writeCertAndKeyInSeparateFiles bool

	defaultCloudEnvironment azure.Environment
}

// mountConfig holds the information for the mount event
type mountConfig struct {
	// the name of the Azure Key Vault instance
	keyvaultName string
	// the type of azure cloud based on azure go sdk
	azureCloudEnvironment azure.Environment
	// authConfig is the config parameters for accessing Key Vault
	authConfig auth.Config
	// tenantID in AAD
	tenantID string
	// podName is the pod name
	podName string
	// podNamespace is the pod namespace
	podNamespace string
}

type keyvaultObject struct {
	content        string
	fileNameSuffix string
	version        string
}

// NewProvider creates a new provider
func NewProvider(constructPEMChain, writeCertAndKeyInSeparateFiles bool, defaultCloudEnvironment azure.Environment) Interface {
	return &provider{
		reporter:                       metrics.NewStatsReporter(),
		constructPEMChain:              constructPEMChain,
		writeCertAndKeyInSeparateFiles: writeCertAndKeyInSeparateFiles,
		defaultCloudEnvironment:        defaultCloudEnvironment,
	}
}

// parseAzureEnvironment returns azure environment by name
func (p *provider) parseAzureEnvironment(cloudName string) (azure.Environment, error) {
	if cloudName == "" {
		return p.defaultCloudEnvironment, nil
	}
	return azure.EnvironmentFromName(cloudName)
}

func (mc *mountConfig) initializeKvClient(vaultURI string) (KeyVault, error) {
	kvEndpoint := strings.TrimSuffix(mc.azureCloudEnvironment.KeyVaultEndpoint, "/")

	cred, err := mc.authConfig.GetCredential(mc.podName, mc.podNamespace, kvEndpoint, mc.azureCloudEnvironment.ActiveDirectoryEndpoint, mc.tenantID, types.PodIdentityNMIPort)
	if err != nil {
		return nil, err
	}
	return NewClient(cred, vaultURI)
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

// GetSecretsStoreObjectContent gets the objects (secret, key, certificate) from keyvault and returns the content
// to the CSI driver. The driver will write the content to the file system.
func (p *provider) GetSecretsStoreObjectContent(ctx context.Context, attrib, secrets map[string]string, defaultFilePermission os.FileMode) ([]types.SecretFile, error) {
	keyvaultName := types.GetKeyVaultName(attrib)
	cloudName := types.GetCloudName(attrib)
	userAssignedIdentityID := types.GetUserAssignedIdentityID(attrib)
	tenantID := types.GetTenantID(attrib)
	cloudEnvFileName := types.GetCloudEnvFileName(attrib)
	podName := types.GetPodName(attrib)
	podNamespace := types.GetPodNamespace(attrib)
	saName := types.GetServiceAccountName(attrib)

	usePodIdentity, err := types.GetUsePodIdentity(attrib)
	if err != nil {
		return nil, fmt.Errorf("failed to parse usePodIdentity flag, error: %w", err)
	}
	useVMManagedIdentity, err := types.GetUseVMManagedIdentity(attrib)
	if err != nil {
		return nil, fmt.Errorf("failed to parse useVMManagedIdentity flag, error: %w", err)
	}
	usePodServiceAccountAnnotation, err := types.GetUsePodServiceAccountAnnotation(attrib)
	if err != nil {
		return nil, fmt.Errorf("failed to parse usePodServiceAccountAnnotation flag, error: %w", err)
	}

	// attributes for workload identity
	var workloadIdentityClientID string
	if usePodServiceAccountAnnotation {
		kubernetesHelper := NewKubernetesHelper(podNamespace, saName)
		workloadIdentityClientID, err = kubernetesHelper.GetServiceAccountClientID()
		if err != nil {
			return nil, fmt.Errorf("failed to get service account client id, error: %w", err)
		}
	} else {
		workloadIdentityClientID = types.GetClientID(attrib)
	}
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
	azureCloudEnv, err := p.parseAzureEnvironment(cloudName)
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

		if err = validate(keyVaultObject); err != nil {
			return nil, wrapObjectTypeError(err, keyVaultObject.ObjectType, keyVaultObject.ObjectName, keyVaultObject.ObjectVersion)
		}

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
	kvClient, err := mc.initializeKvClient(*vaultURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get keyvault client")
	}

	files := []types.SecretFile{}
	for _, keyVaultObject := range keyVaultObjects {
		klog.V(5).InfoS("fetching object from key vault", "objectName", keyVaultObject.ObjectName, "objectType", keyVaultObject.ObjectType, "keyvault", mc.keyvaultName, "pod", klog.ObjectRef{Namespace: podNamespace, Name: podName})

		resolvedKvObjects, err := p.resolveObjectVersions(ctx, kvClient, keyVaultObject)
		if err != nil {
			return nil, err
		}

		for _, resolvedKvObject := range resolvedKvObjects {
			// fetch the object from Key Vault
			result, err := p.getKeyVaultObjectContent(ctx, kvClient, resolvedKvObject)
			if err != nil {
				return nil, err
			}

			for idx := range result {
				r := result[idx]
				objectContent, err := getContentBytes(r.content, resolvedKvObject.ObjectType, resolvedKvObject.ObjectEncoding)
				if err != nil {
					return nil, err
				}

				// objectUID is a unique identifier in the format <object type>/<object name>
				// This is the object id the user sees in the SecretProviderClassPodStatus
				objectUID := resolvedKvObject.GetObjectUID()
				file := types.SecretFile{
					Path:    resolvedKvObject.GetFileName() + r.fileNameSuffix,
					Content: objectContent,
					UID:     objectUID,
					Version: r.version,
				}
				// the validity of file permission is already checked in the validate function above
				file.FileMode, _ = resolvedKvObject.GetFilePermission(defaultFilePermission)

				files = append(files, file)
				klog.V(5).InfoS("added file to the gRPC response", "file", file.Path, "pod", klog.ObjectRef{Namespace: podNamespace, Name: podName})
			}
		}
	}

	return files, nil
}

func (p *provider) resolveObjectVersions(ctx context.Context, kvClient KeyVault, kvObject types.KeyVaultObject) (versions []types.KeyVaultObject, err error) {
	if kvObject.IsSyncingSingleVersion() {
		// version history less than or equal to 1 means only sync the latest and
		// don't add anything to the file name
		return []types.KeyVaultObject{kvObject}, nil
	}

	kvObjectVersions, err := p.getKeyVaultObjectVersions(ctx, kvClient, kvObject)
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
	baseFileName := kvObject.GetFileName()
	objects := []types.KeyVaultObject{}

	sort.Sort(kvObjectVersions)

	// if we're being asked for the latest, then there's no need to skip any versions
	foundFirst := kvObject.ObjectVersion == "" || kvObject.ObjectVersion == "latest"

	for _, objectVersion := range kvObjectVersions {
		foundFirst = foundFirst || objectVersion.Version == kvObject.ObjectVersion

		if foundFirst {
			length := len(objects)
			newObject := kvObject

			newObject.ObjectAlias = filepath.Join(baseFileName, strconv.Itoa(length))
			newObject.ObjectVersion = objectVersion.Version

			objects = append(objects, newObject)

			if length+1 > int(kvObject.ObjectVersionHistory) {
				break
			}
		}
	}

	return objects
}

func (p *provider) getKeyVaultObjectVersions(ctx context.Context, kvClient KeyVault, kvObject types.KeyVaultObject) (versions types.KeyVaultObjectVersionList, err error) {
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
		return getSecretVersions(ctx, kvClient, kvObject)
	case types.VaultObjectTypeKey:
		return getKeyVersions(ctx, kvClient, kvObject)
	case types.VaultObjectTypeCertificate:
		return getCertificateVersions(ctx, kvClient, kvObject)
	default:
		err := errors.Errorf("Invalid vaultObjectTypes. Should be secret, key, or cert")
		return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}
}

func getSecretVersions(ctx context.Context, kvClient KeyVault, kvObject types.KeyVaultObject) ([]types.KeyVaultObjectVersion, error) {
	return kvClient.GetSecretVersions(ctx, kvObject.ObjectName)
}

func getKeyVersions(ctx context.Context, kvClient KeyVault, kvObject types.KeyVaultObject) ([]types.KeyVaultObjectVersion, error) {
	return kvClient.GetKeyVersions(ctx, kvObject.ObjectName)
}

func getCertificateVersions(ctx context.Context, kvClient KeyVault, kvObject types.KeyVaultObject) ([]types.KeyVaultObjectVersion, error) {
	return kvClient.GetCertificateVersions(ctx, kvObject.ObjectName)
}

// getKeyVaultObjectContent gets content of the keyvault object
func (p *provider) getKeyVaultObjectContent(ctx context.Context, kvClient KeyVault, kvObject types.KeyVaultObject) (result []keyvaultObject, err error) {
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
		return p.getSecret(ctx, kvClient, kvObject)
	case types.VaultObjectTypeKey:
		return p.getKey(ctx, kvClient, kvObject)
	case types.VaultObjectTypeCertificate:
		return p.getCertificate(ctx, kvClient, kvObject)
	default:
		err := errors.Errorf("Invalid vaultObjectTypes. Should be secret, key, or cert")
		return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}
}

// getSecret retrieves the secret from the vault
func (p *provider) getSecret(ctx context.Context, kvClient KeyVault, kvObject types.KeyVaultObject) ([]keyvaultObject, error) {
	secret, err := kvClient.GetSecret(ctx, kvObject.ObjectName, kvObject.ObjectVersion)
	if err != nil {
		return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}
	if secret.Value == nil {
		return nil, errors.Errorf("secret value is nil")
	}
	if secret.ID == nil {
		return nil, errors.Errorf("secret id is nil")
	}
	content := *secret.Value
	id := *secret.ID
	version := id.Version()
	result := []keyvaultObject{}
	// if the secret is part of a certificate, then we need to convert the certificate and key to PEM format
	if secret.Kid != nil && len(*secret.Kid) > 0 {
		switch *secret.ContentType {
		case types.CertTypePem:
		case types.CertTypePfx:
			// object format requested is pfx, then return the content as is
			if strings.EqualFold(kvObject.ObjectFormat, types.ObjectFormatPFX) {
				break
			}
			// convert to pem as that's the default object format for this provider
			if content, err = p.decodePKCS12(*secret.Value); err != nil {
				return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
			}
		default:
			err := errors.Errorf("failed to get certificate. unknown content type '%s'", *secret.ContentType)
			return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}

		if p.writeCertAndKeyInSeparateFiles {
			// when writeCertAndKeyInSeparateFiles feature flag is enabled, we write the cert and key in separate files
			// with suffixes .crt and .key respectively. These files are written in addition to the default file which
			// contains the cert and key in a single file to maintain backward compatibility with the existing behavior.
			cert, key := splitCertAndKey(content)
			result = append(result,
				keyvaultObject{version: version, content: cert, fileNameSuffix: ".crt"},
				keyvaultObject{version: version, content: key, fileNameSuffix: ".key"},
			)
		}
	}

	result = append(result, keyvaultObject{content: content, version: version})
	return result, nil
}

// getKey retrieves the key from the vault
func (p *provider) getKey(ctx context.Context, kvClient KeyVault, kvObject types.KeyVaultObject) ([]keyvaultObject, error) {
	keybundle, err := kvClient.GetKey(ctx, kvObject.ObjectName, kvObject.ObjectVersion)
	if err != nil {
		return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}
	if keybundle.Key == nil {
		return nil, errors.Errorf("key value is nil")
	}
	if keybundle.Key.KID == nil {
		return nil, errors.Errorf("key id is nil")
	}

	id := *keybundle.Key.KID
	version := id.Version()
	// for object type "key" the public key is written to the file in PEM format
	switch *keybundle.Key.Kty {
	case azkeys.JSONWebKeyTypeRSA, azkeys.JSONWebKeyTypeRSAHSM:
		nb := keybundle.Key.N
		eb := keybundle.Key.E

		e := new(big.Int).SetBytes(eb).Int64()
		pKey := &rsa.PublicKey{
			N: new(big.Int).SetBytes(nb),
			E: int(e),
		}
		derBytes, err := x509.MarshalPKIXPublicKey(pKey)
		if err != nil {
			return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
		pubKeyBlock := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: derBytes,
		}
		var pemData []byte
		pemData = append(pemData, pem.EncodeToMemory(pubKeyBlock)...)
		return []keyvaultObject{{content: string(pemData), version: version}}, nil
	case azkeys.JSONWebKeyTypeEC, azkeys.JSONWebKeyTypeECHSM:
		xb := keybundle.Key.X
		yb := keybundle.Key.Y

		crv, err := getCurve(*keybundle.Key.Crv)
		if err != nil {
			return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
		pKey := &ecdsa.PublicKey{
			X:     new(big.Int).SetBytes(xb),
			Y:     new(big.Int).SetBytes(yb),
			Curve: crv,
		}
		derBytes, err := x509.MarshalPKIXPublicKey(pKey)
		if err != nil {
			return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
		}
		pubKeyBlock := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: derBytes,
		}
		var pemData []byte
		pemData = append(pemData, pem.EncodeToMemory(pubKeyBlock)...)
		return []keyvaultObject{{content: string(pemData), version: version}}, nil
	default:
		err := errors.Errorf("failed to get key. key type '%s' currently not supported", *keybundle.Key.Kty)
		return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}
}

// getCertificate retrieves the certificate from the vault
func (p *provider) getCertificate(ctx context.Context, kvClient KeyVault, kvObject types.KeyVaultObject) ([]keyvaultObject, error) {
	// for object type "cert" the certificate is written to the file in PEM format
	certbundle, err := kvClient.GetCertificate(ctx, kvObject.ObjectName, kvObject.ObjectVersion)
	if err != nil {
		return nil, wrapObjectTypeError(err, kvObject.ObjectType, kvObject.ObjectName, kvObject.ObjectVersion)
	}
	if certbundle.CER == nil {
		return nil, errors.Errorf("certificate value is nil")
	}
	if certbundle.ID == nil {
		return nil, errors.Errorf("certificate id is nil")
	}

	id := *certbundle.ID
	version := id.Version()

	certBlock := &pem.Block{
		Type:  types.CertificateType,
		Bytes: certbundle.CER,
	}
	var pemData []byte
	pemData = append(pemData, pem.EncodeToMemory(certBlock)...)
	return []keyvaultObject{{content: string(pemData), version: version}}, nil
}

func wrapObjectTypeError(err error, objectType, objectName, objectVersion string) error {
	return errors.Wrapf(err, "failed to get objectType:%s, objectName:%s, objectVersion:%s", objectType, objectName, objectVersion)
}

// decodePkcs12 decodes PKCS#12 client certificates by extracting the public certificates, the private
// keys and converts it to PEM format
func (p *provider) decodePKCS12(value string) (content string, err error) {
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
	if p.constructPEMChain {
		pemCertData, err = fetchCertChains(pemCertData)
		if err != nil {
			return "", err
		}
	}

	pemData = append(pemData, pemKeyData...)
	pemData = append(pemData, pemCertData...)
	return string(pemData), nil
}

func getCurve(crv azkeys.JSONWebKeyCurveName) (elliptic.Curve, error) {
	switch crv {
	case azkeys.JSONWebKeyCurveNameP256:
		return elliptic.P256(), nil
	case azkeys.JSONWebKeyCurveNameP384:
		return elliptic.P384(), nil
	case azkeys.JSONWebKeyCurveNameP521:
		return elliptic.P521(), nil
	default:
		return nil, fmt.Errorf("curve %s is not supported", crv)
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

type node struct {
	cert     *x509.Certificate
	parent   *node
	isParent bool
}

// implementation xref: https://social.technet.microsoft.com/wiki/contents/articles/3147.pki-certificate-chaining-engine-cce.aspx#Building_the_Certificate_Chain
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

			// a leaf cert SubjectKeyId is optional per RFC3280
			if nodes[i].cert.AuthorityKeyId == nil && nodes[j].cert.SubjectKeyId == nil {
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

	if len(nodes) != len(newCertChain) {
		klog.Warning("certificate chain is not complete due to missing intermediate/root certificates in the cert from key vault")
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

// splitCertAndKey takes the given data and splits it into cert and key
// this function doesn't check if the returned cert and key is not empty as this
// can't be enforced. It is possible the secret in the key vault only contains the
// cert or key.
func splitCertAndKey(certAndKey string) (certs string, privKey string) {
	// split the cert and key for PEM format
	// This does not handle the case where cert and key is in PFX format
	// TODO(aramase) consider adding support for PFX format if there is an ask
	var cert, key []byte
	data := []byte(certAndKey)
	for {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type == types.CertificateType {
			cert = append(cert, pem.EncodeToMemory(block)...)
		} else {
			key = append(key, pem.EncodeToMemory(block)...)
		}
		data = rest
	}

	certs = string(cert)
	privKey = string(key)
	return certs, privKey
}
