//go:build e2e
// +build e2e

package keyvault

import (
	"context"
	"fmt"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type Client interface {
	// SetSecret sets the secret in key vault
	SetSecret(name, value string) error
	// DeleteSecret deletes the secret in key vault
	DeleteSecret(name string) error
}

type client struct {
	config        *framework.Config
	secretsClient *azsecrets.Client
}

func NewClient(config *framework.Config) Client {
	opts := &azidentity.ClientSecretCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: cloud.Configuration{
				ActiveDirectoryAuthorityHost: azure.PublicCloud.ActiveDirectoryEndpoint,
			},
		},
	}

	cred, err := azidentity.NewClientSecretCredential(config.TenantID, config.AzureClientID, config.AzureClientSecret, opts)
	Expect(err).To(BeNil())

	c, err := azsecrets.NewClient(getVaultURL(config.KeyvaultName), cred, nil)
	Expect(err).To(BeNil())

	return &client{
		config:        config,
		secretsClient: c,
	}
}

// SetSecret sets the secret in key vault
func (c *client) SetSecret(name, value string) error {
	params := azsecrets.SetSecretParameters{
		Value: to.StringPtr(value),
	}

	By(fmt.Sprintf("Setting secret \"%s\" in keyvault \"%s\"", name, c.config.KeyvaultName))
	_, err := c.secretsClient.SetSecret(context.Background(), name, params, &azsecrets.SetSecretOptions{})
	return err
}

// DeleteSecret deletes the secret in key vault
func (c *client) DeleteSecret(name string) error {
	By(fmt.Sprintf("Deleting secret \"%s\" in keyvault \"%s\"", name, c.config.KeyvaultName))
	_, err := c.secretsClient.DeleteSecret(context.Background(), name, &azsecrets.DeleteSecretOptions{})
	return err
}

func getVaultURL(vaultName string) string {
	return fmt.Sprintf("https://%s.%s/", vaultName, azure.PublicCloud.KeyVaultDNSSuffix)
}
