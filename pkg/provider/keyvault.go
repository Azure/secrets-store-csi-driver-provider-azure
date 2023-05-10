package provider

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/Azure/go-autorest/autorest/date"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/types"
)

type KeyVault interface {
	GetSecret(ctx context.Context, name, version string) (*azsecrets.SecretBundle, error)
	GetSecretVersions(ctx context.Context, name string) ([]types.KeyVaultObjectVersion, error)
	GetKey(ctx context.Context, name, version string) (*azkeys.KeyBundle, error)
	GetKeyVersions(ctx context.Context, name string) ([]types.KeyVaultObjectVersion, error)
	GetCertificate(ctx context.Context, name, version string) (*azcertificates.CertificateBundle, error)
	GetCertificateVersions(ctx context.Context, name string) ([]types.KeyVaultObjectVersion, error)
}

// TODO(aramase): add user agent
type client struct {
	secrets *azsecrets.Client
	keys    *azkeys.Client
	certs   *azcertificates.Client
}

// NewClient creates a new KeyVault client
func NewClient(cred azcore.TokenCredential, vaultURI string) (KeyVault, error) {
	secrets, err := azsecrets.NewClient(vaultURI, cred, nil)
	if err != nil {
		return nil, err
	}
	keys, err := azkeys.NewClient(vaultURI, cred, nil)
	if err != nil {
		return nil, err
	}
	certs, err := azcertificates.NewClient(vaultURI, cred, nil)
	if err != nil {
		return nil, err
	}

	return &client{
		secrets: secrets,
		keys:    keys,
		certs:   certs,
	}, nil
}

func (c *client) GetSecret(ctx context.Context, name, version string) (*azsecrets.SecretBundle, error) {
	resp, err := c.secrets.GetSecret(ctx, name, version, &azsecrets.GetSecretOptions{})
	if err != nil {
		return nil, err
	}
	return &resp.SecretBundle, nil
}

func (c *client) GetKey(ctx context.Context, name, version string) (*azkeys.KeyBundle, error) {
	resp, err := c.keys.GetKey(ctx, name, version, &azkeys.GetKeyOptions{})
	if err != nil {
		return nil, err
	}
	return &resp.KeyBundle, nil
}

func (c *client) GetCertificate(ctx context.Context, name, version string) (*azcertificates.CertificateBundle, error) {
	resp, err := c.certs.GetCertificate(ctx, name, version, &azcertificates.GetCertificateOptions{})
	if err != nil {
		return nil, err
	}
	return &resp.CertificateBundle, nil
}

func (c *client) GetSecretVersions(ctx context.Context, name string) ([]types.KeyVaultObjectVersion, error) {
	pager := c.secrets.NewListSecretVersionsPager(name, &azsecrets.ListSecretVersionsOptions{})
	var versions []types.KeyVaultObjectVersion

	for pager.More() {
		page, err := pager.NextPage(ctx)
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

func (c *client) GetKeyVersions(ctx context.Context, name string) ([]types.KeyVaultObjectVersion, error) {
	pager := c.keys.NewListKeyVersionsPager(name, &azkeys.ListKeyVersionsOptions{})
	var versions []types.KeyVaultObjectVersion

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, key := range page.KeyListResult.Value {
			if key.Attributes == nil {
				continue
			}
			if key.Attributes.Enabled != nil && !*key.Attributes.Enabled {
				continue
			}

			id := *key.KID
			created := date.UnixEpoch()
			if key.Attributes.Created != nil {
				created = *key.Attributes.Created
			}

			versions = append(versions, types.KeyVaultObjectVersion{
				Version: id.Version(),
				Created: created,
			})
		}
	}

	return versions, nil
}

func (c *client) GetCertificateVersions(ctx context.Context, name string) ([]types.KeyVaultObjectVersion, error) {
	pager := c.certs.NewListCertificateVersionsPager(name, &azcertificates.ListCertificateVersionsOptions{})
	var versions []types.KeyVaultObjectVersion

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, cert := range page.CertificateListResult.Value {
			if cert.Attributes == nil {
				continue
			}
			if cert.Attributes.Enabled != nil && !*cert.Attributes.Enabled {
				continue
			}

			id := *cert.ID
			created := date.UnixEpoch()
			if cert.Attributes.Created != nil {
				created = *cert.Attributes.Created
			}

			versions = append(versions, types.KeyVaultObjectVersion{
				Version: id.Version(),
				Created: created,
			})
		}
	}

	return versions, nil
}
