//go:build e2e
// +build e2e

package keyvault

import (
	"context"
	"fmt"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"

	kv "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type Client interface {
	// SetSecret sets the secret in key vault
	SetSecret(name, value string) error
	// DeleteSecret deletes the secret in key vault
	DeleteSecret(name string) error
}

type client struct {
	config         *framework.Config
	keyvaultClient kv.BaseClient
}

func NewClient(config *framework.Config) Client {
	kvClient := kv.New()
	kvEndPoint := azure.PublicCloud.KeyVaultEndpoint
	if '/' == kvEndPoint[len(kvEndPoint)-1] {
		kvEndPoint = kvEndPoint[:len(kvEndPoint)-1]
	}

	oauthConfig, err := getOAuthConfig(azure.PublicCloud, config.TenantID)
	Expect(err).To(BeNil())

	armSpt, err := adal.NewServicePrincipalToken(*oauthConfig, config.AzureClientID, config.AzureClientSecret, kvEndPoint)
	Expect(err).To(BeNil())
	kvClient.Authorizer = autorest.NewBearerAuthorizer(armSpt)

	return &client{
		config:         config,
		keyvaultClient: kvClient,
	}
}

// SetSecret sets the secret in key vault
func (c *client) SetSecret(name, value string) error {
	By(fmt.Sprintf("Setting secret \"%s\" in keyvault \"%s\"", name, c.config.KeyvaultName))
	_, err := c.keyvaultClient.SetSecret(context.Background(), getVaultURL(c.config.KeyvaultName), name, kv.SecretSetParameters{
		Value: to.StringPtr(value),
	})
	return err
}

// DeleteSecret deletes the secret in key vault
func (c *client) DeleteSecret(name string) error {
	By(fmt.Sprintf("Deleting secret \"%s\" in keyvault \"%s\"", name, c.config.KeyvaultName))
	_, err := c.keyvaultClient.DeleteSecret(context.Background(), getVaultURL(c.config.KeyvaultName), name)
	return err
}

func getOAuthConfig(env azure.Environment, tenantID string) (*adal.OAuthConfig, error) {
	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	return oauthConfig, nil
}

func getVaultURL(vaultName string) string {
	return fmt.Sprintf("https://%s.%s/", vaultName, azure.PublicCloud.KeyVaultDNSSuffix)
}
