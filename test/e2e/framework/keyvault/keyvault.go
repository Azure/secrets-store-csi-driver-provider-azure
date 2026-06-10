//go:build e2e
// +build e2e

package keyvault

import (
	"context"
	"fmt"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/types"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type Client interface {
	// SetSecret sets the secret in key vault
	SetSecret(name, value string) error
	// GetSecretVersions gets the enabled versions of the secret in key vault
	GetSecretVersions(name string) ([]types.KeyVaultObjectVersion, error)
	// DeleteSecret deletes the secret in key vault
	DeleteSecret(name string) error
}

type client struct {
	config        *framework.Config
	secretsClient *azsecrets.Client
}

func NewClient(config *framework.Config) Client {
	// Use Azure CLI credential so the test reuses the `az login` (WIF) session
	// performed by the pipeline. This avoids depending on a pool-level managed
	// identity, which is no longer available on the CI agents.
	cred, err := azidentity.NewAzureCLICredential(nil)
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

// GetSecretVersions gets the enabled versions of the secret in key vault
func (c *client) GetSecretVersions(name string) ([]types.KeyVaultObjectVersion, error) {
	By(fmt.Sprintf("Getting versions for secret \"%s\" in keyvault \"%s\"", name, c.config.KeyvaultName))

	pager := c.secretsClient.NewListSecretVersionsPager(name, &azsecrets.ListSecretVersionsOptions{})
	versions := []types.KeyVaultObjectVersion{}
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			return nil, err
		}

		for _, secret := range page.SecretListResult.Value {
			if secret.Attributes == nil {
				continue
			}
			if secret.Attributes.Enabled != nil && !*secret.Attributes.Enabled {
				continue
			}

			id := *secret.ID
			created := date.UnixEpoch()
			if secret.Attributes.Created != nil {
				created = *secret.Attributes.Created
			}

			versions = append(versions, types.KeyVaultObjectVersion{
				Version: id.Version(),
				Created: created,
			})
		}
	}

	return versions, nil
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
