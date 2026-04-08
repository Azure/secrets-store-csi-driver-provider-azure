// Package identitybinding provides internal utilities for identity binding
// transport configuration. This package is internal to prevent external
// consumption of implementation details.
package identitybinding

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base32"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"k8s.io/klog/v2"
)

const (
	// defaultTokenProxyURL is the well-known Kubernetes API server endpoint
	// used as the token proxy for identity binding.
	defaultTokenProxyURL = "https://kubernetes.default.svc.cluster.local" // nolint:gosec // not a credential

	// defaultCAFile is the standard Kubernetes service account CA certificate path.
	defaultCAFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

	sniHashSalt = "identity-binding"
	sniPrefix   = "a"
	sniSuffix   = ".ests.aks"

	apiServerSANPrefix = "hcp-kubernetes."
	apiServerSANSuffix = ".svc.cluster.local"
)

// Config holds the configuration for identity binding proxy transport.
type Config struct {
	// TokenProxyURL is the URL of the token proxy endpoint.
	// If empty, defaults to the Kubernetes API server endpoint.
	TokenProxyURL string
	// SNIName is the TLS server name for the proxy connection.
	// If empty, it is computed from the API server's serving certificate.
	SNIName string
}

// CreateProxyTransport creates a proxy transport by leveraging the SDK's
// internal proxy configuration via a throwaway WorkloadIdentityCredential.
//
// NOT goroutine-safe: this function temporarily sets and unsets process-wide
// environment variables. It must be called from main() before any concurrent
// activity (e.g., before the gRPC server starts).
func CreateProxyTransport(cfg Config) (policy.Transporter, error) {
	tokenProxyURL := cfg.TokenProxyURL
	if tokenProxyURL == "" {
		tokenProxyURL = defaultTokenProxyURL
	}

	sniName := cfg.SNIName
	if sniName == "" {
		var err error
		sniName, err = computeSNIName()
		if err != nil {
			return nil, fmt.Errorf("compute SNI name: %w", err)
		}
	}

	klog.InfoS("configuring identity binding proxy transport",
		"tokenProxyURL", tokenProxyURL, "sniName", sniName)

	// Temporarily set env vars that the SDK's internal proxy configuration reads.
	// Save and restore all env vars we touch to be hermetic.
	envKeys := []string{"AZURE_KUBERNETES_TOKEN_PROXY", "AZURE_KUBERNETES_SNI_NAME", "AZURE_KUBERNETES_CA_FILE"}
	savedEnv := make(map[string]string, len(envKeys))
	savedExists := make(map[string]bool, len(envKeys))
	for _, k := range envKeys {
		if v, ok := os.LookupEnv(k); ok {
			savedEnv[k] = v
			savedExists[k] = true
		}
	}
	defer func() {
		for _, k := range envKeys {
			if savedExists[k] {
				os.Setenv(k, savedEnv[k])
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	os.Setenv("AZURE_KUBERNETES_TOKEN_PROXY", tokenProxyURL)
	os.Setenv("AZURE_KUBERNETES_SNI_NAME", sniName)

	// Only set CA_FILE if the file exists. In some environments (e.g., managed
	// deployments where SNI and proxy URL are provided via flags), the default
	// CA file may not be present or a different trust root is used.
	if _, err := os.Stat(defaultCAFile); err == nil {
		os.Setenv("AZURE_KUBERNETES_CA_FILE", defaultCAFile)
	} else {
		os.Unsetenv("AZURE_KUBERNETES_CA_FILE")
	}

	// Create a throwaway WorkloadIdentityCredential with proxy enabled.
	// This triggers the SDK's internal proxy configuration which builds the
	// transport and sets it on the credential's ClientOptions.Transport.
	// Dummy values are fine: the SDK only checks they're non-empty during construction.
	// The token file is never read during construction.
	wic, err := azidentity.NewWorkloadIdentityCredential(&azidentity.WorkloadIdentityCredentialOptions{
		ClientID:                 "placeholder",
		TenantID:                 "placeholder",
		TokenFilePath:            os.DevNull, // never read during construction
		DisableInstanceDiscovery: true,
		EnableAzureProxy:         true,
	})
	if err != nil {
		return nil, fmt.Errorf("create throwaway credential for proxy transport extraction: %w", err)
	}

	return extractProxyTransport(wic)
}

// computeSNIName computes the identity binding SNI name by connecting to the
// API server, extracting a cluster identifier from the serving certificate,
// and computing a hash-based server name.
func computeSNIName() (string, error) {
	clusterID, err := extractClusterID()
	if err != nil {
		return "", fmt.Errorf("extract cluster ID from API server certificate: %w", err)
	}
	klog.V(5).InfoS("extracted cluster ID from API server certificate")
	return resolveServerName(clusterID), nil
}

// extractClusterID connects to the Kubernetes API server via TLS and extracts
// a cluster identifier from the serving certificate's SANs.
func extractClusterID() (string, error) {
	caCert, err := os.ReadFile(defaultCAFile)
	if err != nil {
		return "", fmt.Errorf("read CA file %s: %w (use --sni-name flag to skip automatic SNI detection)", defaultCAFile, err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return "", fmt.Errorf("no valid certificates found in %s", defaultCAFile)
	}

	apiServerPort := os.Getenv("KUBERNETES_SERVICE_PORT")
	if apiServerPort == "" {
		apiServerPort = "443"
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", "kubernetes.default.svc.cluster.local:"+apiServerPort, &tls.Config{
		RootCAs:    caPool,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return "", fmt.Errorf("TLS connect to API server: %w", err)
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return "", fmt.Errorf("no certificates presented by API server")
	}

	leaf := certs[0]
	for _, san := range leaf.DNSNames {
		if strings.HasPrefix(san, apiServerSANPrefix) && strings.HasSuffix(san, apiServerSANSuffix) {
			id := san[len(apiServerSANPrefix) : len(san)-len(apiServerSANSuffix)]
			if id == "" {
				return "", fmt.Errorf("empty cluster ID in SAN %q", san)
			}
			return id, nil
		}
	}

	return "", fmt.Errorf("cluster ID SAN not found in API server certificate (use --sni-name flag to provide SNI name explicitly)")
}

// resolveServerName generates an SNI server name from a cluster identifier.
// The generated name is RFC 1035 compliant.
func resolveServerName(clusterID string) string {
	hashedBytes := sha256.Sum256([]byte(sniHashSalt + clusterID))
	hashedID := base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(hashedBytes[:])
	return strings.ToLower(sniPrefix + hashedID + sniSuffix)
}
