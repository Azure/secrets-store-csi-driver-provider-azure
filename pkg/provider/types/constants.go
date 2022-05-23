package types

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
	TenantIDParameter = "tenantId"
	// CloudEnvFileNameParameter is the name of the cloud env file name parameter
	CloudEnvFileNameParameter = "cloudEnvFileName"
	// ClientIDParameter is the name of the client ID parameter
	// This clientID is used for workload identity
	ClientIDParameter = "clientID"
	// ObjectsParameter is the name of the objects parameter
	ObjectsParameter = "objects"
)
