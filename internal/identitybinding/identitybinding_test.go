package identitybinding

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/stretchr/testify/require"
)

// verifiedAzidentityVersion is the azidentity module version that the reflect
// field chain in extractProxyTransport was verified against. If the SDK is
// updated, this test fails as a reminder to re-verify the field chain.
const verifiedAzidentityVersion = "v1.14.0-beta.2.0.20260124023332-4c5175309ebb"

func TestAzidentityVersionNotChanged(t *testing.T) {
	// Read go.sum from the repo root to find the pinned azidentity version.
	// This is more reliable than debug.ReadBuildInfo() which may not list
	// deps of the parent module when running tests in internal/ packages.
	data, err := os.ReadFile("../../go.sum")
	require.NoError(t, err, "failed to read go.sum — run tests from the repo root")

	var version string
	for _, line := range strings.Split(string(data), "\n") {
		// go.sum lines: <module> <version> <hash>
		// Also has lines with /go.mod suffix — skip those
		if strings.HasPrefix(line, "github.com/Azure/azure-sdk-for-go/sdk/azidentity ") && !strings.Contains(line, "/go.mod") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				version = parts[1]
				break
			}
		}
	}

	require.NotEmpty(t, version, "azidentity dependency not found in go.sum")
	require.Equal(t, verifiedAzidentityVersion, version,
		"azidentity version changed — re-verify the reflect field chain in extractProxyTransport and update verifiedAzidentityVersion")
}

func TestExtractProxyTransport(t *testing.T) {
	t.Setenv("AZURE_KUBERNETES_TOKEN_PROXY", "https://localhost:8443")

	wic, err := azidentity.NewWorkloadIdentityCredential(&azidentity.WorkloadIdentityCredentialOptions{
		ClientID:                 "placeholder",
		TenantID:                 "placeholder",
		TokenFilePath:            "/dev/null",
		DisableInstanceDiscovery: true,
		EnableAzureProxy:         true,
	})
	require.NoError(t, err)

	transport, err := extractProxyTransport(wic)
	require.NoError(t, err)
	require.NotNil(t, transport)
}

// TestExtractProxyTransport_FieldChainValid validates each field in the reflect
// chain exists with expected characteristics. If the SDK renames or restructures
// any field, this test pinpoints exactly which one changed.
func TestExtractProxyTransport_FieldChainValid(t *testing.T) {
	t.Setenv("AZURE_KUBERNETES_TOKEN_PROXY", "https://localhost:8443")

	wic, err := azidentity.NewWorkloadIdentityCredential(&azidentity.WorkloadIdentityCredentialOptions{
		ClientID:                 "placeholder",
		TenantID:                 "placeholder",
		TokenFilePath:            "/dev/null",
		DisableInstanceDiscovery: true,
		EnableAzureProxy:         true,
	})
	require.NoError(t, err)

	v := reflect.ValueOf(wic).Elem()

	cred := v.FieldByName("cred")
	require.True(t, cred.IsValid(), "WorkloadIdentityCredential must have a 'cred' field")
	require.Equal(t, reflect.Pointer, cred.Kind())
	require.False(t, cred.IsNil())

	client := cred.Elem().FieldByName("client")
	require.True(t, client.IsValid(), "ClientAssertionCredential must have a 'client' field")
	require.Equal(t, reflect.Pointer, client.Kind())
	require.False(t, client.IsNil())

	opts := client.Elem().FieldByName("opts")
	require.True(t, opts.IsValid(), "confidentialClient must have an 'opts' field")
	require.Equal(t, reflect.Struct, opts.Kind())

	co := opts.FieldByName("ClientOptions")
	require.True(t, co.IsValid(), "confidentialClientOptions must have 'ClientOptions'")
	require.Equal(t, reflect.Struct, co.Kind())

	tf := co.FieldByName("Transport")
	require.True(t, tf.IsValid(), "ClientOptions must have 'Transport'")
	require.Equal(t, reflect.Interface, tf.Kind())
	require.False(t, tf.IsNil(), "Transport must not be nil when proxy is enabled")
}

func TestExtractProxyTransport_NilCredential(t *testing.T) {
	t.Setenv("AZURE_KUBERNETES_TOKEN_PROXY", "")

	wic, err := azidentity.NewWorkloadIdentityCredential(&azidentity.WorkloadIdentityCredentialOptions{
		ClientID:                 "placeholder",
		TenantID:                 "placeholder",
		TokenFilePath:            "/dev/null",
		DisableInstanceDiscovery: true,
		EnableAzureProxy:         false,
	})
	require.NoError(t, err)

	transport, err := extractProxyTransport(wic)
	require.Error(t, err)
	require.Nil(t, transport)
	require.Contains(t, err.Error(), "no transport configured")
}

func TestResolveServerName(t *testing.T) {
	tests := []struct {
		name      string
		clusterID string
	}{
		{"typical ID", "696995ccb303450001b31a18"},
		{"another ID", "abc123def456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sni := resolveServerName(tt.clusterID)

			require.True(t, strings.HasPrefix(sni, "a"), "SNI must start with 'a'")
			require.True(t, strings.HasSuffix(sni, ".ests.aks"), "SNI must end with '.ests.aks'")
			require.Equal(t, strings.ToLower(sni), sni, "SNI must be lowercase")

			firstLabel := strings.TrimSuffix(sni, ".ests.aks")
			require.Len(t, firstLabel, 53, "first label should be 53 chars: 'a' + 52 base32hex")

			require.Equal(t, sni, resolveServerName(tt.clusterID), "must be deterministic")
		})
	}

	require.NotEqual(t, resolveServerName("id1"), resolveServerName("id2"),
		"different inputs must produce different SNIs")
}
