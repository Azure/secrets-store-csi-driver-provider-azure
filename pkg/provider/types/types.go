package types

import "time"

const (
	// VaultObjectTypeSecret secret vault object type
	VaultObjectTypeSecret = "secret"
	// VaultObjectTypeKey key vault object type
	VaultObjectTypeKey = "key"
	// VaultObjectTypeCertificate certificate vault object type
	VaultObjectTypeCertificate = "cert"

	CertTypePem = "application/x-pem-file"
	CertTypePfx = "application/x-pkcs12"

	CertificateType = "CERTIFICATE"

	ObjectFormatPEM = "pem"
	ObjectFormatPFX = "pfx"

	ObjectEncodingHex    = "hex"
	ObjectEncodingBase64 = "base64"
	ObjectEncodingUtf8   = "utf-8"

	// pod identity NMI port
	PodIdentityNMIPort = "2579"

	CSIAttributePodName              = "csi.storage.k8s.io/pod.name"
	CSIAttributePodNamespace         = "csi.storage.k8s.io/pod.namespace"
	CSIAttributeServiceAccountTokens = "csi.storage.k8s.io/serviceAccount.tokens" // nolint

	// KeyVaultNameParameter is the name of the key vault name parameter
	KeyVaultNameParameter = "keyvaultName"
	// CloudNameParameter is the name of the cloud name parameter
	CloudNameParameter = "cloudName"
	// UsePodIdentityParameter is the name of the use pod identity parameter
	UsePodIdentityParameter = "usePodIdentity"
	// UseVMManagedIdentityParameter is the name of the use VM managed identity parameter
	UseVMManagedIdentityParameter = "useVMManagedIdentity"
	// UserAssignedIdentityIDParameter is the name of the user assigned identity ID parameter
	UserAssignedIdentityIDParameter = "userAssignedIdentityID"
	// TenantIDParameter is the name of the tenant ID parameter
	// TODO(aramase): change this from tenantId to tenantID after v1.2 release
	// ref: https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/857
	TenantIDParameter = "tenantId"
	// CloudEnvFileNameParameter is the name of the cloud env file name parameter
	CloudEnvFileNameParameter = "cloudEnvFileName"
	// ClientIDParameter is the name of the client ID parameter
	// This clientID is used for workload identity
	ClientIDParameter = "clientID"
	// ObjectsParameter is the name of the objects parameter
	ObjectsParameter = "objects"
)

// KeyVaultObject holds keyvault object related config
type KeyVaultObject struct {
	// the name of the Azure Key Vault objects
	ObjectName string `json:"objectName" yaml:"objectName"`
	// the filename the object will be written to
	ObjectAlias string `json:"objectAlias" yaml:"objectAlias"`
	// the version of the Azure Key Vault objects
	ObjectVersion string `json:"objectVersion" yaml:"objectVersion"`
	// The number of versions to load for this secret starting at the latest version
	ObjectVersionHistory int32 `json:"objectVersionHistory" yaml:"objectVersionHistory"`
	// the type of the Azure Key Vault objects
	ObjectType string `json:"objectType" yaml:"objectType"`
	// the format of the Azure Key Vault objects
	// supported formats are PEM, PFX
	ObjectFormat string `json:"objectFormat" yaml:"objectFormat"`
	// The encoding of the object in KeyVault
	// Supported encodings are Base64, Hex, Utf-8
	ObjectEncoding string `json:"objectEncoding" yaml:"objectEncoding"`
	// FilePermission is the file permissions
	FilePermission string `json:"filePermission" yaml:"filePermission"`
}

// SecretFile holds content and metadata of a secret file that is sent
// back to the driver
type SecretFile struct {
	Content  []byte
	Path     string
	FileMode int32
	UID      string
	Version  string
}

// StringArray holds a list of strings
type StringArray struct {
	Array []string `json:"array" yaml:"array"`
}

// KeyVaultObjectVersion holds the version id and when that version was
// created for a specific version of a secret from KeyVault
type KeyVaultObjectVersion struct {
	Version string
	Created time.Time
}

// KeyVaultObjectVersionList holds a list of KeyVaultObjectVersion
type KeyVaultObjectVersionList []KeyVaultObjectVersion

func (list KeyVaultObjectVersionList) Len() int {
	return len(list)
}

func (list KeyVaultObjectVersionList) Less(i, j int) bool {
	return list[i].Created.After(list[j].Created)
}

func (list KeyVaultObjectVersionList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}
