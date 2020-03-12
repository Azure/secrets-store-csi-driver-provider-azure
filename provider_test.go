package main

import (
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
