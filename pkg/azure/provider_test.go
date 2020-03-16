package azure

import (
	"context"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/azure"
)

func TestGetVaultDNSSuffix(t *testing.T) {
	cases := []struct {
		cloudName string
		expected  *string
	}{
		{
			cloudName: "",
			expected:  &azure.PublicCloud.KeyVaultDNSSuffix,
		},
		{
			cloudName: "AZURECHINACLOUD",
			expected:  &azure.ChinaCloud.KeyVaultDNSSuffix,
		},
		{
			cloudName: "AZUREGERMANCLOUD",
			expected:  &azure.GermanCloud.KeyVaultDNSSuffix,
		},
		{
			cloudName: "AZUREPUBLICCLOUD",
			expected:  &azure.PublicCloud.KeyVaultDNSSuffix,
		},
		{
			cloudName: "AZUREUSGOVERNMENTCLOUD",
			expected:  &azure.USGovernmentCloud.KeyVaultDNSSuffix,
		},
	}

	for _, tc := range cases {
		actual, _ := GetVaultDNSSuffix(tc.cloudName)
		if !strings.EqualFold(*tc.expected, *actual) {
			t.Fatalf("expected: %v, got: %v", *tc.expected, *actual)
		}
	}
}

func TestGetVaultURL(t *testing.T) {
	testEnvs := []string{"", "AZUREPUBLICCLOUD", "AZURECHINACLOUD", "AZUREGERMANCLOUD", "AZUREUSGOVERNMENTCLOUD"}
	vaultDNSSuffix := []string{"vault.azure.net", "vault.azure.net", "vault.azure.cn", "vault.microsoftazure.de", "vault.usgovcloudapi.net"}

	cases := []struct {
		desc        string
		vaultName   string
		expectedErr bool
	}{
		{
			desc:        "vault name > 24",
			vaultName:   "longkeyvaultnamewhichisnotvalid",
			expectedErr: true,
		},
		{
			desc:        "vault name < 3",
			vaultName:   "kv",
			expectedErr: true,
		},
		{
			desc:        "vault name contains non alpha-numeric chars",
			vaultName:   "kv_test",
			expectedErr: true,
		},
		{
			desc:        "valid vault name in public cloud",
			vaultName:   "testkv",
			expectedErr: false,
		},
	}

	for i, tc := range cases {
		t.Log(i, tc.desc)
		p, err := NewProvider()
		if err != nil {
			t.Fatalf("expected nil err, got: %v", err)
		}
		p.KeyvaultName = tc.vaultName

		for idx := range testEnvs {
			vaultURL, err := p.getVaultURL(context.Background(), testEnvs[idx])
			if tc.expectedErr && err == nil || !tc.expectedErr && err != nil {
				t.Fatalf("expected error: %v, got error: %v", tc.expectedErr, err)
			}
			expectedURL := "https://" + tc.vaultName + "." + vaultDNSSuffix[idx] + "/"
			if !tc.expectedErr && expectedURL != *vaultURL {
				t.Fatalf("expected vault url: %s, got: %s", expectedURL, *vaultURL)
			}
		}
	}
}
