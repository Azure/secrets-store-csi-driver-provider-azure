package provider

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/mock_keyvault"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/types"
)

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
		mc := &mountConfig{
			keyvaultName: tc.vaultName,
		}

		for idx := range testEnvs {
			azCloudEnv, err := parseAzureEnvironment(testEnvs[idx])
			if err != nil {
				t.Fatalf("Error parsing cloud environment %v", err)
			}
			mc.azureCloudEnvironment = azCloudEnv
			vaultURL, err := mc.getVaultURL()
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

func TestParseAzureEnvironment(t *testing.T) {
	envNamesArray := []string{"AZURECHINACLOUD", "AZUREGERMANCLOUD", "AZUREPUBLICCLOUD", "AZUREUSGOVERNMENTCLOUD", ""}
	for _, envName := range envNamesArray {
		azureEnv, err := parseAzureEnvironment(envName)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if strings.EqualFold(envName, "") && !strings.EqualFold(azureEnv.Name, "AZUREPUBLICCLOUD") {
			t.Fatalf("string doesn't match, expected AZUREPUBLICCLOUD, got %s", azureEnv.Name)
		} else if !strings.EqualFold(envName, "") && !strings.EqualFold(envName, azureEnv.Name) {
			t.Fatalf("string doesn't match, expected %s, got %s", envName, azureEnv.Name)
		}
	}

	wrongEnvName := "AZUREWRONGCLOUD"
	_, err := parseAzureEnvironment(wrongEnvName)
	if err == nil {
		t.Fatalf("expected error for wrong azure environment name")
	}
}

func TestDecodePKCS12(t *testing.T) {
	cases := []struct {
		desc         string
		value        string
		expectedCert string
		expectedKey  string
	}{
		{
			desc:  "one certificate and key in pfx data",
			value: "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",
			expectedCert: `-----BEGIN CERTIFICATE-----
MIIC2DCCAcACCQD9DZdcsr7kJDANBgkqhkiG9w0BAQsFADAuMRYwFAYDVQQDDA1k
ZW1vLnRlc3QuY29tMRQwEgYDVQQKDAtpbmdyZXNzLXRsczAeFw0yMDA1MjIxNjIz
MjZaFw0yMTA1MjIxNjIzMjZaMC4xFjAUBgNVBAMMDWRlbW8udGVzdC5jb20xFDAS
BgNVBAoMC2luZ3Jlc3MtdGxzMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKC
AQEAte0os8X6ZKbEUWoFJdSfcYoSovbxPBhtisEJd/U+oOK1jKH/HMBliTv+9l6O
vIhldtt48v57mk4P0M72KT8ulXcBasNV95DNnPsEpAqs7wKrhftleeDMKPnk8VvU
6jidPy6SO6Ntbp8tJchrbfMwZW7e2y+PVweKN8QwNECQPfygBtX8jP93CG6oYvK9
FDS45U1UcKUdxTLfXmSvORPo0HFEXLNvZxmdjSsrP0oSbasJfMr02DZb5/6MSxCb
J/FnPwdqXQH/cM6rQDLw2Is5iWn0QXPEYZMqYbtMAJoY0UEVHVHgIUb/HucQ+SEk
tt6kG3sIGKsKLiuymZGozRFNqwIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQCSZNbl
WFMjnZuGiFzIZoqfKOp/Dtw48poNJkrxMBJLkiciJD6drXj8vnTQrZUuR25TIiD/
Sq+cO+XVRcJKNP13FjFpRdyHYRtAze4TaQZSJlW2nyfeUtUQkwj2iMhv5l1UMnPG
7+Jxg56aA+IBvyE/tAQVvS0NPdq6Ht2MX6j40ERTXmS8qNdY6qi3ZCEAPazlNsUF
C6nLdViZ/vbQ+l6uEcNsEsPJ6SDTNKLkO9tU7pWCa6QBTncuFLbpDqr3Q+lvx4mv
MVw9RO3NiLuDiPQA0VfKSMrEJJUp4F88pbEax5nq525Rbp85RWkmVoc97UuFS+oc
ldGQrUHVb2/iI1fd
-----END CERTIFICATE-----
`,
			expectedKey: `-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC17SizxfpkpsRR
agUl1J9xihKi9vE8GG2KwQl39T6g4rWMof8cwGWJO/72Xo68iGV223jy/nuaTg/Q
zvYpPy6VdwFqw1X3kM2c+wSkCqzvAquF+2V54Mwo+eTxW9TqOJ0/LpI7o21uny0l
yGtt8zBlbt7bL49XB4o3xDA0QJA9/KAG1fyM/3cIbqhi8r0UNLjlTVRwpR3FMt9e
ZK85E+jQcURcs29nGZ2NKys/ShJtqwl8yvTYNlvn/oxLEJsn8Wc/B2pdAf9wzqtA
MvDYizmJafRBc8Rhkyphu0wAmhjRQRUdUeAhRv8e5xD5ISS23qQbewgYqwouK7KZ
kajNEU2rAgMBAAECggEBAK9MJxUapkxH+RDt1KoAN+aigZSv2ADtFNhHa0VAdal2
6jLpgbWFmhDjU6i3slfuIb6meePC3PzxTQIJ+l4COHPi6OWj9PkIeWdS5MTgWIQx
kW8Xr08CEhdFu5npv7408SgJSvTWY8Lc9BbdCM84LqD+dRTEvhzA8ikMDNq8f4CJ
hLreFUUl/udHacpMdE8mpB6vgCUliZEjBlHHC9qD2mDKgWb0cm4jkO9PcHxz8CXL
szcRV2vqTwvsJcZWcJwTzjhFxq/lUZrgbwpn60iKlov3BCRoTJBppOXi01giom3v
Wz7Y7DoFbHfizh6jyBrf3ODhKJQ3CGvS65QCS0aJ/kECgYEA4JuGC9DpQYmlzWbV
0CqJYnTcZKqcPQx/A1QZDKot0VWqF61vZIku5XuoGKGfY3eLwVZJJZqxoTlVTbuT
nNzYJe+EHzftRoUxUqXZtIh9VdirJMwCu4RMdwk705FA8+8FcTKXarKWBbAzUmFi
iINR2rlRJHVyh2cOA9hWPbEXX0sCgYEAz1qAYUIMBGnccY9mALGArWTmtyNN3DcB
9tl3/5SzfL1NlcOWsBXdZL61oTl9LhOjqHGZbf164/uFuKNHTsT1E63180UKujmV
TbHL6N6MrMctaJfgru3+XprTMd5pwjzd8huX603OtS8Gvn5gKdBRkG1ZI8CrfTl6
sJI9YRvl7yECgYEAjUIiptHHsVkhdrIDLL1j1BEM/x6xzk9KnkxIyMdKs4n9xJBm
K0N/xAHmMT+Mn6DyuzBKJqVIq84EETQ0XQYjxpABdyTUTHK+F22JItpogRIYaLcJ
zOcitAaRoriKsh+UO6IGyqrwYTl0vY3Ty2lTlIzSNGzND81HajGn43q56UsCgYEA
pGqArZ3vZXiDgdBQ82/MNrFxd/oYfOtpNVFPI2vHvrtkT8KdM9bCjGXkI4kwR17v
QFuDa4G49hm0+KkPm9f09LvV8CXo0a1jRA4dP/Nn3IC68tqrIEo6js15dWuEtK4K
1zUmC0DRDT3SvS38FmvGoRzzt7PIxyzSqjvrS5sRgcECgYAQ6b0YsM4p+89s4ALK
BPfGIKpoIEMKUcwiT3ovRrwIu1vbu70WRcYAi5do6rwOakp3FyUcQznkeZEOAQmc
xrBy8R64vg83WMuRITAqY6vartSa3oehqUHW0YbhGDVEtSrolXEs5elArUHbpYnX
SIVZww73PTGisLmXfIvKvr8GBA==
-----END PRIVATE KEY-----
`,
		},
		{
			desc:  "multiple certificates and one key in pfx data",
			value: `MIIHCgIBAzCCBtAGCSqGSIb3DQEHAaCCBsEEgga9MIIGuTCCBa8GCSqGSIb3DQEHBqCCBaAwggWcAgEAMIIFlQYJKoZIhvcNAQcBMBwGCiqGSIb3DQEMAQYwDgQIsQ14jUE4T4YCAggAgIIFaJNUlr3d5VUEaodVgXIJvqnL9bOzyr/Qo5I3SUrxXOoWTzHxgs0xHzsbA3PtYX0sHK4khZ82sXEdkrvmSfhgcmS0949r5qSO47lA7fh2yXIhMbg0mzhyAa9SSKLeoYpnIw1TgmoFpgUeIBUyLR3s7UXt5FFTsbjPtRXY1C7+memHZ921MA30rbAcKVLU89hZ/M4C6u0Gfi2OlFaJxmtGwbL6WSKsbKTHzySSdcUbzqrtPr41yijhEdv/vwGCYLx6qgViw8XpEpGQO66jWWaa5ajgN9pP7czOikMH3urEl20B6hJyo33Js7aWqJiUFOikbFs1UYqE+BIohJPfDJh57JSXsBSnKeee53ymebVhdshTb2zkrLoVYiIVtH76CDzUc/0IGDug5FhhDemDz5hkUvRnlcvwwii4ixJKHRIxHOEQ8jim1eXYImLmxwBEzxisQBPDxB7hUECmRQ1gYqtchbovdwiIXbR+lk7yyNmsm98oq4GofCMQbA3nAerpEV9oZNx2z/TdDKgbzpk5BLejtBYyO7sjWmm67RrzdjzTQVTzC7amnzn1Ip/mY6T3IlpwiZIg98VY1VWUB0HfWltA3krxoGTxW8N1jLVCDUe8EJNQmkAQh9y5PNfLCwHUjgXnU4cQEzHOe18EoVVWZ+YOJXjqJkShpg8mrCzDT8jHqWtJ6ncHnW1n5g6ROqRKnlXhzRNQV2CtCAedDbgGrK/ymIFrIXxa3QGTBHY/K/ZEMii5H0JlJT3Lek856rJNzlLfx/CD/d0Xnq/tDuy0aZ25zUTX4ZEIHbQRic53ujzUTeKlqA9BZKt8kZhdIEITi0ax45ATgoa/vcLPuwuVhkSIQXPPbK4XsQ5fOd8OZuES7YvWz/LaVmvQFUicWb4XgsurXVM2ytgTIiTdMcxCVJDFfUV/6jfVdZ5nZCdK23UaJjWbjqu9MgQmcRGEaiak0zxMVOpORMRXkiSarkzYrrxfv5xXcY0HeV7PPmMjXLp1yHHfY2XiZqPJs1Xv4f2iM3e8L/PDIgzG90SXOH6SL/56ss3XpPg/4cb25yK7m0xDdfgxXG+bNGSGVULXUUc0doCjo9SBQOgJlXB9XQ3N1op4tCDtIVlUgV8qAi8Nd+HBdg/Wo6dVPUEUkBCd4XkM3MFlVPkDwRhcN92XaUb2Rt86664EAJkS6yVh4PC5zi6Rx1lRzDeT1HywvnArKHAc3K0hWKGBtTbOMbcowSBPi0dNQZiWvgE7SHtB1qKxPelpxhu8vizYl8X8gSoy+trG3rCgiqUC6qF6Hwr8h+83sobGkU9QManBr3wg4E2pPy4zQ0o6mndzG+Vztbq4WbRteP1Vkr/eFmCvTRauGnjZGEveaPV48k7KXY69hHv09jFDTY9AU+ywOJvtpOIA5DLX6a5T3aHXirZlK+Jpb/fbddSoC8qMzn/DaHvVuJmGOgSE9jDbuoPQx9LQb7KZp2u6JcAwtv9BiZiqQ4Cjr5akddZTHfEa5xx4bgMECdQ8qw7xkmbXSYq2Af2Imdgh9uBkAPfAKrvUSZPCa9y2KQwSKfCqE85PYMjWd2lWtGr7eZ3PiwyV89r3D/5nxR1zJDFLMpl+diLgv2hUighiUd4v63lSxxnYfSK4Kc4Kio5VuYtY8jk0lowRRCmVjpdox3Y5br9XAxxG1ShXuBFci9g48auVtXLz90JmG9yjbQWzIut5lysmCwhbWSzjbW9/FUJAqz9JsgI9Q3wRNqK8Fd5xkOOBeQVJcLmZ25X6don2PKPT0hPsxYt067ZyQYc6Rcv1vRYsKNNTwe83Qoq2qLh6TURAuxU1pERFwl/ncFzf0BVoyZyTnoLabFfOVAkwggECBgkqhkiG9w0BBwGggfQEgfEwge4wgesGCyqGSIb3DQEMCgECoIG0MIGxMBwGCiqGSIb3DQEMAQMwDgQI50UUDLaw8dECAggABIGQHRm4079PB3AZSkqzZ3Ecrmt1fTPhgA7d1unatD/jNS2IzU+AQiSugAGO7+AsmKB0yAs6JA73mb4XPRqQd2gpJ8SilcLGI+ZSUXv/lRr2yQfPzZ2m7XGzm2eVrwgfVkTvl1//0iVBym+rj+k7LKQaUiuj+uUwq1QAzQUNQd8oshbmcU1HzLQKicSQ4QYRfHr7MSUwIwYJKoZIhvcNAQkVMRYEFBa4koFGJzKXD0GgYQta2xnmSS2xMDEwITAJBgUrDgMCGgUABBSRivYGiKYxZwnq2/98Ka/eGqEPhwQIXNw2IIK5QMwCAggA`,
			expectedCert: `-----BEGIN CERTIFICATE-----
MIIBqjCCAU+gAwIBAgIRAKJNuamTAo5J4rM3VWjDK5cwCgYIKoZIzj0EAwIwGjEY
MBYGA1UEAxMPaW50ZXJtZWRpYXRlLWNhMB4XDTIxMDgwOTE5NTkwMloXDTIxMDgx
MDE5NTkwMlowDjEMMAoGA1UEAxMDZm9vMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcD
QgAElqKRYGw0H9M5xPAy4ulv3PNnAkKuH5Sp++Zf/YdD8Ioj2iNExijdjvAML4Rh
sAxJZhYWLV3nW22kk+q21UQ1I6OBgTB/MA4GA1UdDwEB/wQEAwIHgDAdBgNVHSUE
FjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwHQYDVR0OBBYEFKG3lkIvQ5r9SCQziqJL
a5IBvn5dMB8GA1UdIwQYMBaAFKbwdJqSgN/FVvoKJTZwFGXc/veAMA4GA1UdEQQH
MAWCA2ZvbzAKBggqhkjOPQQDAgNJADBGAiEAnZBNyEOMen26N5eYvVU81zUebjca
gu/37qsELmGpmlcCIQDpv/levexCmAS+cna6+hx2XZlF5CufzBnGGF4pS87oxw==
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIBkDCCATegAwIBAgIRAMVDURSAQxm0HFJNnveSHbMwCgYIKoZIzj0EAwIwEjEQ
MA4GA1UEAxMHcm9vdC1jYTAeFw0yMTA4MDkxOTU0MjNaFw0zMTA4MDcxOTU0MjNa
MBoxGDAWBgNVBAMTD2ludGVybWVkaWF0ZS1jYTBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABMhw2HqcudX85glAogQ1iqUL4ntYdt73HjRhgZ5/uLFByKLjDkJriIlx
ZYSxcCiJ8BTWAVKp94M38DdC33iBBNajZjBkMA4GA1UdDwEB/wQEAwIBBjASBgNV
HRMBAf8ECDAGAQH/AgEAMB0GA1UdDgQWBBSm8HSakoDfxVb6CiU2cBRl3P73gDAf
BgNVHSMEGDAWgBST0EMXQTt8FI1eCm8X9jS6MNjeLjAKBggqhkjOPQQDAgNHADBE
AiA3t40JojHqLDER+dVJ7XdGk4Pxoxyn0IHloTHvL//nagIgN644i0E6RsyI3IBi
4r42rfgbnh9rz/fRcN7tANOyrPI=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIBaDCCAQ6gAwIBAgIRAKWgLws4aUKy51qL5cwuQ5QwCgYIKoZIzj0EAwIwEjEQ
MA4GA1UEAxMHcm9vdC1jYTAeFw0yMTA4MDkxOTU0MDZaFw0zMTA4MDcxOTU0MDZa
MBIxEDAOBgNVBAMTB3Jvb3QtY2EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQJ
6G2bvdIoY9h+R3raSTTGFQF85Bv+odMqo92t9WHEIvdDAuZ1i5Z5GKCWBEBoSRbM
U/EXAuOVDVSro4nsBrkZo0UwQzAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/BAgw
BgEB/wIBATAdBgNVHQ4EFgQUk9BDF0E7fBSNXgpvF/Y0ujDY3i4wCgYIKoZIzj0E
AwIDSAAwRQIhAK9eYLEdaJ3TRozlZlyLdYbKsxNswGK2KwTMxZBT/kd3AiBwNmYh
o4BmggmqQKVGvdcJzqJZRGYN5QZiHxJbZL77Pg==
-----END CERTIFICATE-----
`,
			expectedKey: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgRFqIa1jmKUCRekh0
mMfSMXxrxWv2gvQlvVO0g4+kF92hRANCAASWopFgbDQf0znE8DLi6W/c82cCQq4f
lKn75l/9h0PwiiPaI0TGKN2O8AwvhGGwDElmFhYtXedbbaST6rbVRDUj
-----END PRIVATE KEY-----
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			p := &provider{constructPEMChain: true}
			content, err := p.decodePKCS12(tc.value)
			if err != nil {
				t.Fatalf("expected nil err, got: %v", err)
			}
			if content != tc.expectedKey+tc.expectedCert {
				t.Fatalf("certificate and key mismatch")
			}
		})
	}
}

func TestParseAzureEnvironmentAzureStackCloud(t *testing.T) {
	azureStackCloudEnvName := "AZURESTACKCLOUD"
	file, err := os.CreateTemp("", "ut")
	defer os.Remove(file.Name())
	if err != nil {
		t.Fatalf("expected error to be nil, got: %+v", err)
	}
	_, err = io.WriteString(file, fmt.Sprintf(`{"name": "%s"}`, azureStackCloudEnvName))
	if err != nil {
		t.Fatalf("expected error to be nil, got: %+v", err)
	}
	_, err = parseAzureEnvironment(azureStackCloudEnvName)
	if err == nil {
		t.Fatalf("expected error to be not nil as AZURE_ENVIRONMENT_FILEPATH is not set")
	}

	err = setAzureEnvironmentFilePath(file.Name())
	defer os.Unsetenv(azure.EnvironmentFilepathName)
	if err != nil {
		t.Fatalf("expected error to be nil, got: %+v", err)
	}
	env, err := parseAzureEnvironment(azureStackCloudEnvName)
	if err != nil {
		t.Fatalf("expected error to be nil, got: %+v", err)
	}
	if env.Name != azureStackCloudEnvName {
		t.Fatalf("expected environment name to be '%s', got: '%s'", azureStackCloudEnvName, env.Name)
	}
}

func TestGetContentBytes(t *testing.T) {
	cases := []struct {
		desc           string
		objectContent  string
		objectEncoding string
		objectType     string
		expectedErr    error
		expectedValue  []byte
	}{
		{
			desc:           "No encoding specified for a secret",
			objectContent:  "abcdefg",
			objectEncoding: "",
			objectType:     "secret",
			expectedErr:    nil,
			expectedValue:  []byte{97, 98, 99, 100, 101, 102, 103},
		},
		{
			desc:           "Certificate object type",
			objectContent:  "foobar123",
			objectEncoding: "",
			objectType:     "cert",
			expectedErr:    nil,
			expectedValue:  []byte{102, 111, 111, 98, 97, 114, 49, 50, 51},
		},
		{
			desc:           "Key object type",
			objectContent:  "keyobjecttype",
			objectEncoding: "",
			objectType:     "key",
			expectedErr:    nil,
			expectedValue:  []byte{107, 101, 121, 111, 98, 106, 101, 99, 116, 116, 121, 112, 101},
		},
		{
			desc:           "UTF-8 encoding",
			objectContent:  "TestSecret1",
			objectEncoding: "utf-8",
			objectType:     "secret",
			expectedErr:    nil,
			expectedValue:  []byte{84, 101, 115, 116, 83, 101, 99, 114, 101, 116, 49},
		},
		{
			desc:           "Base64 encoding",
			objectContent:  "QmFzZTY0RW5jb2RlZFN0cmluZw==",
			objectEncoding: "base64",
			objectType:     "secret",
			expectedErr:    nil,
			expectedValue:  []byte{66, 97, 115, 101, 54, 52, 69, 110, 99, 111, 100, 101, 100, 83, 116, 114, 105, 110, 103},
		},
		{
			desc:           "Hex encoding",
			objectContent:  "486578456E636F646564537472696E67",
			objectEncoding: "hex",
			objectType:     "secret",
			expectedErr:    nil,
			expectedValue:  []byte{72, 101, 120, 69, 110, 99, 111, 100, 101, 100, 83, 116, 114, 105, 110, 103},
		},
		{
			desc:           "Invalid encoding",
			objectContent:  "TestSecret1",
			objectEncoding: "NotAnEncoding",
			objectType:     "secret",
			expectedErr:    fmt.Errorf("invalid objectEncoding. Should be utf-8, base64, or hex"),
			expectedValue:  []byte{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			actualValue, err := getContentBytes(tc.objectContent, tc.objectType, tc.objectEncoding)
			if tc.expectedErr != nil && err.Error() != tc.expectedErr.Error() || tc.expectedErr == nil && err != nil {
				t.Fatalf("expected err: %+v, got: %+v", tc.expectedErr, err)
			}
			if len(tc.expectedValue) > 0 {
				if !bytes.Equal(tc.expectedValue, actualValue) {
					t.Fatalf("Expected and actual byte values do not match.  Expected: %v  Actual: %v", string(tc.expectedValue), string(actualValue))
				}
			}
		})
	}
}

func TestGetLatestNKeyVaultObjects(t *testing.T) {
	now := time.Now()

	cases := []struct {
		desc            string
		kvObject        types.KeyVaultObject
		versions        types.KeyVaultObjectVersionList
		expectedObjects []types.KeyVaultObject
	}{
		{
			desc: "filename is name/index when no alias provided",
			kvObject: types.KeyVaultObject{
				ObjectName:           "secret1",
				ObjectVersion:        "latest",
				ObjectVersionHistory: 5,
			},
			versions: types.KeyVaultObjectVersionList{
				types.KeyVaultObjectVersion{
					Version: "a",
					Created: now.Add(time.Hour * 10),
				},
				types.KeyVaultObjectVersion{
					Version: "b",
					Created: now.Add(time.Hour * 9),
				},
				types.KeyVaultObjectVersion{
					Version: "c",
					Created: now.Add(time.Hour * 8),
				},
				types.KeyVaultObjectVersion{
					Version: "d",
					Created: now.Add(time.Hour * 7),
				},
				types.KeyVaultObjectVersion{
					Version: "e",
					Created: now.Add(time.Hour * 6),
				},
			},
			expectedObjects: []types.KeyVaultObject{
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "0"),
					ObjectVersion:        "a",
					ObjectVersionHistory: 5,
				},
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "1"),
					ObjectVersion:        "b",
					ObjectVersionHistory: 5,
				},
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "2"),
					ObjectVersion:        "c",
					ObjectVersionHistory: 5,
				},
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "3"),
					ObjectVersion:        "d",
					ObjectVersionHistory: 5,
				},
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "4"),
					ObjectVersion:        "e",
					ObjectVersionHistory: 5,
				},
			},
		},
		{
			desc: "sorts versions by descending created date",
			kvObject: types.KeyVaultObject{
				ObjectName:           "secret1",
				ObjectVersion:        "latest",
				ObjectVersionHistory: 5,
			},
			versions: types.KeyVaultObjectVersionList{
				types.KeyVaultObjectVersion{
					Version: "c",
					Created: now.Add(time.Hour * 8),
				},
				types.KeyVaultObjectVersion{
					Version: "e",
					Created: now.Add(time.Hour * 6),
				},
				types.KeyVaultObjectVersion{
					Version: "b",
					Created: now.Add(time.Hour * 9),
				},
				types.KeyVaultObjectVersion{
					Version: "a",
					Created: now.Add(time.Hour * 10),
				},
				types.KeyVaultObjectVersion{
					Version: "d",
					Created: now.Add(time.Hour * 7),
				},
			},
			expectedObjects: []types.KeyVaultObject{
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "0"),
					ObjectVersion:        "a",
					ObjectVersionHistory: 5,
				},
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "1"),
					ObjectVersion:        "b",
					ObjectVersionHistory: 5,
				},
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "2"),
					ObjectVersion:        "c",
					ObjectVersionHistory: 5,
				},
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "3"),
					ObjectVersion:        "d",
					ObjectVersionHistory: 5,
				},
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "4"),
					ObjectVersion:        "e",
					ObjectVersionHistory: 5,
				},
			},
		},
		{
			desc: "starts with latest version when no version specified",
			kvObject: types.KeyVaultObject{
				ObjectName:           "secret1",
				ObjectVersionHistory: 2,
			},
			versions: types.KeyVaultObjectVersionList{
				types.KeyVaultObjectVersion{
					Version: "a",
					Created: now.Add(time.Hour * 10),
				},
				types.KeyVaultObjectVersion{
					Version: "b",
					Created: now.Add(time.Hour * 9),
				},
			},
			expectedObjects: []types.KeyVaultObject{
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "0"),
					ObjectVersion:        "a",
					ObjectVersionHistory: 2,
				},
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "1"),
					ObjectVersion:        "b",
					ObjectVersionHistory: 2,
				},
			},
		},
		{
			desc: "fewer than ObjectVersionHistory results returns all versions",
			kvObject: types.KeyVaultObject{
				ObjectName:           "secret1",
				ObjectVersionHistory: 200,
			},
			versions: types.KeyVaultObjectVersionList{
				types.KeyVaultObjectVersion{
					Version: "a",
					Created: now.Add(time.Hour * 10),
				},
				types.KeyVaultObjectVersion{
					Version: "b",
					Created: now.Add(time.Hour * 9),
				},
			},
			expectedObjects: []types.KeyVaultObject{
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "0"),
					ObjectVersion:        "a",
					ObjectVersionHistory: 200,
				},
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "1"),
					ObjectVersion:        "b",
					ObjectVersionHistory: 200,
				},
			},
		},
		{
			desc: "starts at ObjectVersion when specified",
			kvObject: types.KeyVaultObject{
				ObjectName:           "secret1",
				ObjectVersion:        "c",
				ObjectVersionHistory: 5,
			},
			versions: types.KeyVaultObjectVersionList{
				types.KeyVaultObjectVersion{
					Version: "c",
					Created: now.Add(time.Hour * 8),
				},
				types.KeyVaultObjectVersion{
					Version: "e",
					Created: now.Add(time.Hour * 6),
				},
				types.KeyVaultObjectVersion{
					Version: "b",
					Created: now.Add(time.Hour * 9),
				},
				types.KeyVaultObjectVersion{
					Version: "a",
					Created: now.Add(time.Hour * 10),
				},
				types.KeyVaultObjectVersion{
					Version: "d",
					Created: now.Add(time.Hour * 7),
				},
			},
			expectedObjects: []types.KeyVaultObject{
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "0"),
					ObjectVersion:        "c",
					ObjectVersionHistory: 5,
				},
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "1"),
					ObjectVersion:        "d",
					ObjectVersionHistory: 5,
				},
				{
					ObjectName:           "secret1",
					ObjectAlias:          filepath.Join("secret1", "2"),
					ObjectVersion:        "e",
					ObjectVersionHistory: 5,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			actualObjects := getLatestNKeyVaultObjects(tc.kvObject, tc.versions)

			if !reflect.DeepEqual(actualObjects, tc.expectedObjects) {
				t.Fatalf("expected: %+v, but got: %+v", tc.expectedObjects, actualObjects)
			}
		})
	}
}

func TestFormatKeyVaultObject(t *testing.T) {
	cases := []struct {
		desc                   string
		keyVaultObject         types.KeyVaultObject
		expectedKeyVaultObject types.KeyVaultObject
	}{
		{
			desc: "leading and trailing whitespace trimmed from all fields",
			keyVaultObject: types.KeyVaultObject{
				ObjectName:     "secret1     ",
				ObjectVersion:  "",
				ObjectEncoding: "base64   ",
				ObjectType:     "  secret",
				ObjectAlias:    "",
			},
			expectedKeyVaultObject: types.KeyVaultObject{
				ObjectName:     "secret1",
				ObjectVersion:  "",
				ObjectEncoding: "base64",
				ObjectType:     "secret",
				ObjectAlias:    "",
			},
		},
		{
			desc: "no data loss for already sanitized object",
			keyVaultObject: types.KeyVaultObject{
				ObjectName:     "secret1",
				ObjectVersion:  "version1",
				ObjectEncoding: "base64",
				ObjectType:     "secret",
				ObjectAlias:    "alias",
			},
			expectedKeyVaultObject: types.KeyVaultObject{
				ObjectName:     "secret1",
				ObjectVersion:  "version1",
				ObjectEncoding: "base64",
				ObjectType:     "secret",
				ObjectAlias:    "alias",
			},
		},
		{
			desc: "no data loss for int properties",
			keyVaultObject: types.KeyVaultObject{
				ObjectName:           "secret1",
				ObjectVersion:        "latest",
				ObjectEncoding:       "base64",
				ObjectType:           "secret",
				ObjectAlias:          "alias",
				ObjectVersionHistory: 12,
			},
			expectedKeyVaultObject: types.KeyVaultObject{
				ObjectName:           "secret1",
				ObjectVersion:        "latest",
				ObjectEncoding:       "base64",
				ObjectType:           "secret",
				ObjectAlias:          "alias",
				ObjectVersionHistory: 12,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			formatKeyVaultObject(&tc.keyVaultObject)
			if !reflect.DeepEqual(tc.keyVaultObject, tc.expectedKeyVaultObject) {
				t.Fatalf("expected: %+v, but got: %+v", tc.expectedKeyVaultObject, tc.keyVaultObject)
			}
		})
	}
}

func TestFetchCertChain(t *testing.T) {
	rootCACert := `
-----BEGIN CERTIFICATE-----
MIIBeTCCAR6gAwIBAgIRAM3RAPH7k1Q+bICMC0mzKhkwCgYIKoZIzj0EAwIwGjEY
MBYGA1UEAxMPRXhhbXBsZSBSb290IENBMB4XDTIwMTIwMzAwMTAxNFoXDTMwMTIw
MTAwMTAxNFowGjEYMBYGA1UEAxMPRXhhbXBsZSBSb290IENBMFkwEwYHKoZIzj0C
AQYIKoZIzj0DAQcDQgAE1/AGExuSemtxPRzFECpefowtkcOQr7jaq355kfb2hUR2
LnMn+71fD4mZmMXT0kuxgeE2zC2CxOHdoJ/FmcQJxaNFMEMwDgYDVR0PAQH/BAQD
AgEGMBIGA1UdEwEB/wQIMAYBAf8CAQEwHQYDVR0OBBYEFKTuLl7BATUYGD6ZeUV3
2f8UAWoqMAoGCCqGSM49BAMCA0kAMEYCIQDEz2XKXPb0Q/Y40Gtxo8r6sa0Ra6U0
fpTPteqfpl8iGQIhAOo8tpUYiREVSYZu130fN0Gvy4WmJMFAi7JrVeSnZ7uP
-----END CERTIFICATE-----
`

	intermediateCert := `
-----BEGIN CERTIFICATE-----
MIIBozCCAUmgAwIBAgIRANEldEfXaQ+L2M1ahC6w4vAwCgYIKoZIzj0EAwIwGjEY
MBYGA1UEAxMPRXhhbXBsZSBSb290IENBMB4XDTIwMTIwMzAwMTAyNFoXDTMwMTIw
MTAwMTAyNFowJDEiMCAGA1UEAxMZRXhhbXBsZSBJbnRlcm1lZGlhdGUgQ0EgMTBZ
MBMGByqGSM49AgEGCCqGSM49AwEHA0IABOhTE8r5NpDIDF/6VLgPT+//0IR59Uzn
78JfV54E0qFA21khrcqc20/RJD+lyUv313gYQD9SxBXXxcGtl1OJ0s2jZjBkMA4G
A1UdDwEB/wQEAwIBBjASBgNVHRMBAf8ECDAGAQH/AgEAMB0GA1UdDgQWBBR+2JY0
VhjrWsrUng+V8dgeZBOGJzAfBgNVHSMEGDAWgBSk7i5ewQE1GBg+mXlFd9n/FAFq
KjAKBggqhkjOPQQDAgNIADBFAiB9EQB+siuNboL7k78CUzhZJ+5lD0cXUpGYGWYT
rxcX6QIhALGptitzrZ4z/MDMBPkan48bqk6O08e1tQ9dJOIoEKq7
-----END CERTIFICATE-----
`

	serverCert := `
-----BEGIN CERTIFICATE-----
MIIBwjCCAWmgAwIBAgIQGIPRUsQ/sFI1fkxZHCSU6jAKBggqhkjOPQQDAjAkMSIw
IAYDVQQDExlFeGFtcGxlIEludGVybWVkaWF0ZSBDQSAxMB4XDTIwMTIwMzAwMTAz
NloXDTIwMTIwNDAwMTAzNlowFjEUMBIGA1UEAxMLZXhhbXBsZS5jb20wWTATBgcq
hkjOPQIBBggqhkjOPQMBBwNCAAS0FvMzMHAfc6mOIEgijRngeRcNaDdp63AbCVeJ
tuKNX7j4KLbkQcACj6g+hblJu4NCJChFmeEYf8b7xw+q0dPOo4GKMIGHMA4GA1Ud
DwEB/wQEAwIHgDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwHQYDVR0O
BBYEFIRRQ0915ExZz30TeVhCpwgP3SEYMB8GA1UdIwQYMBaAFH7YljRWGOtaytSe
D5Xx2B5kE4YnMBYGA1UdEQQPMA2CC2V4YW1wbGUuY29tMAoGCCqGSM49BAMCA0cA
MEQCIH9NxXnWaip9fZyv9VJcfFz7tcdxTq10SrTO7gKhyJkpAiAljZFFK687kc6J
kzqEt441cQasPp5ohL5U4cJN6lAuwA==
-----END CERTIFICATE-----
`

	expectedCertChain := `-----BEGIN CERTIFICATE-----
MIIBwjCCAWmgAwIBAgIQGIPRUsQ/sFI1fkxZHCSU6jAKBggqhkjOPQQDAjAkMSIw
IAYDVQQDExlFeGFtcGxlIEludGVybWVkaWF0ZSBDQSAxMB4XDTIwMTIwMzAwMTAz
NloXDTIwMTIwNDAwMTAzNlowFjEUMBIGA1UEAxMLZXhhbXBsZS5jb20wWTATBgcq
hkjOPQIBBggqhkjOPQMBBwNCAAS0FvMzMHAfc6mOIEgijRngeRcNaDdp63AbCVeJ
tuKNX7j4KLbkQcACj6g+hblJu4NCJChFmeEYf8b7xw+q0dPOo4GKMIGHMA4GA1Ud
DwEB/wQEAwIHgDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwHQYDVR0O
BBYEFIRRQ0915ExZz30TeVhCpwgP3SEYMB8GA1UdIwQYMBaAFH7YljRWGOtaytSe
D5Xx2B5kE4YnMBYGA1UdEQQPMA2CC2V4YW1wbGUuY29tMAoGCCqGSM49BAMCA0cA
MEQCIH9NxXnWaip9fZyv9VJcfFz7tcdxTq10SrTO7gKhyJkpAiAljZFFK687kc6J
kzqEt441cQasPp5ohL5U4cJN6lAuwA==
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIBozCCAUmgAwIBAgIRANEldEfXaQ+L2M1ahC6w4vAwCgYIKoZIzj0EAwIwGjEY
MBYGA1UEAxMPRXhhbXBsZSBSb290IENBMB4XDTIwMTIwMzAwMTAyNFoXDTMwMTIw
MTAwMTAyNFowJDEiMCAGA1UEAxMZRXhhbXBsZSBJbnRlcm1lZGlhdGUgQ0EgMTBZ
MBMGByqGSM49AgEGCCqGSM49AwEHA0IABOhTE8r5NpDIDF/6VLgPT+//0IR59Uzn
78JfV54E0qFA21khrcqc20/RJD+lyUv313gYQD9SxBXXxcGtl1OJ0s2jZjBkMA4G
A1UdDwEB/wQEAwIBBjASBgNVHRMBAf8ECDAGAQH/AgEAMB0GA1UdDgQWBBR+2JY0
VhjrWsrUng+V8dgeZBOGJzAfBgNVHSMEGDAWgBSk7i5ewQE1GBg+mXlFd9n/FAFq
KjAKBggqhkjOPQQDAgNIADBFAiB9EQB+siuNboL7k78CUzhZJ+5lD0cXUpGYGWYT
rxcX6QIhALGptitzrZ4z/MDMBPkan48bqk6O08e1tQ9dJOIoEKq7
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIBeTCCAR6gAwIBAgIRAM3RAPH7k1Q+bICMC0mzKhkwCgYIKoZIzj0EAwIwGjEY
MBYGA1UEAxMPRXhhbXBsZSBSb290IENBMB4XDTIwMTIwMzAwMTAxNFoXDTMwMTIw
MTAwMTAxNFowGjEYMBYGA1UEAxMPRXhhbXBsZSBSb290IENBMFkwEwYHKoZIzj0C
AQYIKoZIzj0DAQcDQgAE1/AGExuSemtxPRzFECpefowtkcOQr7jaq355kfb2hUR2
LnMn+71fD4mZmMXT0kuxgeE2zC2CxOHdoJ/FmcQJxaNFMEMwDgYDVR0PAQH/BAQD
AgEGMBIGA1UdEwEB/wQIMAYBAf8CAQEwHQYDVR0OBBYEFKTuLl7BATUYGD6ZeUV3
2f8UAWoqMAoGCCqGSM49BAMCA0kAMEYCIQDEz2XKXPb0Q/Y40Gtxo8r6sa0Ra6U0
fpTPteqfpl8iGQIhAOo8tpUYiREVSYZu130fN0Gvy4WmJMFAi7JrVeSnZ7uP
-----END CERTIFICATE-----
`

	cases := []struct {
		desc        string
		cert        string
		expectedErr bool
	}{
		{
			desc:        "order: root, intermediate, server certs",
			cert:        rootCACert + intermediateCert + serverCert,
			expectedErr: false,
		},
		{
			desc:        "order: root, server, intermediate certs",
			cert:        rootCACert + serverCert + intermediateCert,
			expectedErr: false,
		},
		{
			desc:        "order: intermediate, root, server certs",
			cert:        intermediateCert + rootCACert + serverCert,
			expectedErr: false,
		},
		{
			desc:        "order: intermediate, server, root certs",
			cert:        intermediateCert + serverCert + rootCACert,
			expectedErr: false,
		},
		{
			desc:        "order: server, root, intermediate certs",
			cert:        serverCert + rootCACert + intermediateCert,
			expectedErr: false,
		},
		{
			desc:        "order: server, intermediate, root certs",
			cert:        serverCert + intermediateCert + rootCACert,
			expectedErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			certChain, err := fetchCertChains([]byte(tc.cert))
			if tc.expectedErr && err == nil || !tc.expectedErr && err != nil {
				t.Fatalf("expected error: %v, got error: %v", tc.expectedErr, err)
			}
			if string(certChain) != expectedCertChain {
				t.Fatalf(cmp.Diff(expectedCertChain, string(certChain)))
			}
		})
	}
}

func TestFetchCertChainWithLeafMissingSKI(t *testing.T) {
	rootCACert := `
-----BEGIN CERTIFICATE-----
MIIC8jCCAdqgAwIBAgIDEjRWMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNVBAMTB3Rl
c3QtY2EwHhcNMjQwNDAyMTYwODU0WhcNMzQwNDAyMTYwODU0WjASMRAwDgYDVQQD
Ewd0ZXN0LWNhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAt0R6suEJ
zlRDkMUKUEFIRRnqPmbDUM+h5k4tc5bgAJgX1EGf/lVBJ4gzUGzYayc9qyIcsKqI
/wyxsEm9SOnqR5lQkE/dJ3BsiSV/+wts6OX86KLWn4gHFm1xzl3xAj0/7w0qrEGj
5ASEF+RsfQq+oY/jglZCRWaVq23F77L6NeOFCicEKCRRLKClwXFFrGErwoUk3ef1
CJ7GD1C+7Pk4uHQC4BYttcSyVYfTn4fdYMEQtEY3hAWRsfZqJ/epRvxFFaDXnfGL
PWoj+IYRx0YWsV6FY8rqyat8PGtvY4JR5RdF9nNIKapV3n3W98tc6EiXBZybULsd
5z9PHU0hDabSxwIDAQABo1EwTzAdBgNVHSUEFjAUBggrBgEFBQcDAgYIKwYBBQUH
AwEwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUALAG+YJ+DGejdTyT9T6+nYg+
4ZIwDQYJKoZIhvcNAQELBQADggEBABgkDtHj51xeHwFfSYQmUXnTQl59VCXGdulU
Fx8yfI5aMJzWR0SGTnJ8/VpBUZi6VTTz45qvi8xJgnpF8SLtKjXQlbqIerO7KL+M
7EnK2O1IGMKPboGM3pgJJQ7jS6aPObtFvuLUwECYoFw6dEzQkauzZjNA5FjWPImM
9VonFvAOpA45r9/b5liZ/Lg7gfdOtlLYUpCU1bPtem4v60oFmKh5IMOdLDVCgcga
HXlyr1Q1xkPwnHMt1aOPJPuMs1DSfbhP40bUvYh3gU5B7XpUpaHxlltm/h9/CsPE
z9rzlA+Co/z78Wn/LtvjVrxJj4QHcfXhiIltAaAUnJP+kZ0+3I0=
-----END CERTIFICATE-----
`

	intermediateCert1 := `
-----BEGIN CERTIFICATE-----
MIIDFTCCAf2gAwIBAgIDEjRWMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNVBAMTB3Rl
c3QtY2EwHhcNMjQwNDAyMTYwODU1WhcNMzQwNDAyMTYwODU1WjAUMRIwEAYDVQQD
Ewl0ZXN0LWludDEwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCtCcWc
98ConFAuw6PpsuljvSDfoee9UqFCpfxeIHdTu1wcxTtdRROvbwbdDfc0UYsOIRlR
J+zVJdPJBS2n/lCiXVjfWgCmPQpSQhXyJAmD1VwgDrT4YFm31RCQmkQlQCaY7s4Y
KG/eawzaSA0CYBjMS4ss58zFJeZyPB5y0OMb6Cu7Q5MsJBkwpdjADuT5otfZPIDG
wB1N7vRtS37ajPR3cxrR7+jehcjEiZBeiW0tTdTgH1TFpoDqxCITkAyqu40AhG1i
xH4DZs285LaeqVZnLiW2CwlKAMXaHwxL8FmhoFN6+FwXq/fBE2oUFhI5V00wp99Y
WucRgoHZHjPLm4j9AgMBAAGjcjBwMB0GA1UdJQQWMBQGCCsGAQUFBwMCBggrBgEF
BQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBTOi/c99QBhXboU8Jo153ux
QY/QFjAfBgNVHSMEGDAWgBQAsAb5gn4MZ6N1PJP1Pr6diD7hkjANBgkqhkiG9w0B
AQsFAAOCAQEAhH5ygm9o2hxMJl4AuKfq3S2AKtDho+gW7D7XDTAyoCDjxskzTagz
DNi1sEkgOOt+pYPdzZLqAPb5qE+jvHpXSrIiG+wYEZ3I9bMjOfXFFh9wfgNNpiCO
KCP6c6XGJuAzDLLOqe726TjXFwh2rtCs6IXl6MZBfYEWpCYgbJytuGiVbEy/4zHu
REXmmUVKBTT7nq9zMGqK7rkyEJeq51uGrO4NfkgDEPJqN7RQySMi8drXFcjjyvEo
vfXoEug1sLeGyrEMD+8wfvhFJDtFIxCCu6gRIn6H8QMouNTIZBiZhtHPD9La5vh/
RAqYxnujJ1Gw4ZpQExiLxPq0MdP3NLOibg==
-----END CERTIFICATE-----
`

	intermediateCert2 := `
-----BEGIN CERTIFICATE-----
MIIDFzCCAf+gAwIBAgIDEjRWMA0GCSqGSIb3DQEBCwUAMBQxEjAQBgNVBAMTCXRl
c3QtaW50MTAeFw0yNDA0MDIxNjA4NTVaFw0zNDA0MDIxNjA4NTVaMBQxEjAQBgNV
BAMTCXRlc3QtaW50MjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBANYX
Dy2Uk67tIDsrVNnfTUJRFVCFNFgAEcFjTUZDLNPJiJy3i4QTDn+4oZpjlZjSC0l1
QabxqQGgAycwA0TBJykKCKWN1uCucVBvXW8s4OdF/2HQIw13HZWZJOTJvMK+kfIx
2/4cUr7lXl7QoC01VnhHQW1cLrzjfs2WZ+Nv1VGOxFx1S8yjNc4CvV4pKzeeANxE
cgjAdMqwaT9uLKJ2fa/6jnlf4xVrTaO96LlPaiI/qn5Q/xdcdizQ3SDREV5BXe4V
MOVoI0K9uoluy98RZvGG0G92PN7gM1BOfc8S/2Qaulq5eKzMShK4LmE2sWNniWjd
4IEysKsOnllIxHae9wkCAwEAAaNyMHAwHQYDVR0lBBYwFAYIKwYBBQUHAwIGCCsG
AQUFBwMBMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFC1hoo+AKqTs77o5YwSf
e11hmyX0MB8GA1UdIwQYMBaAFM6L9z31AGFduhTwmjXne7FBj9AWMA0GCSqGSIb3
DQEBCwUAA4IBAQCQhr+0m+UukGYqhwZSIlgL8YKulwMz0SNhyMkEcrxEl03Nx7SZ
KgwZH+bxgykYocL0RxAZcjBCmcoLyy3Ebl743s4eRppBXQc+kYzYyaWEUpUxNoq2
tFi+yqruQbkNSQ/rzKrRrwZAD8vP3mcUScjx9UNwXAbr4NbA10US1WlWYA17v4kf
HXGhV997/HNLNyUYizePeeU4DrdVNgI3hdy4mpyWRDWc6fyaKbeFJbWo+KHqT5zB
fTEBltyrpTqCdq0nXLwIzR+J4brtBH/LE2Wo9kp5bh4xUJQeybc46QeAyAvbV812
e4t6jdnEGzymy5pzbGjtEp+gW4zaSGLBGL3V
-----END CERTIFICATE-----
`

	serverCert := `
-----BEGIN CERTIFICATE-----
MIIDGjCCAgKgAwIBAgIDEjRWMA0GCSqGSIb3DQEBCwUAMBQxEjAQBgNVBAMTCXRl
c3QtaW50MjAeFw0yNDA0MDIxNjA4NTVaFw0yNTA0MDIxNjA4NTVaMBQxEjAQBgNV
BAMTCXRlc3QtbGVhZjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALg2
kdP54gsbku7vUfIYf59H5VQLoDXoCZpTO7lxsHEh9QveViINEgYJ5Ycq8YijRR9J
W2oAWSLdE+mEQ+kvzxjqIwcZgCq0cGwErXKaPZCNnQKiV0cGCgWBPyqZcE1noiSE
5FqWBK0FDXmdiA5p2TSNfu6okJqVB/u0Bxbxqq+/V+aQ5KuFjwmIBT+/2kxwBma2
Si/SDJts/H9izYcjTuxyJ1Cq9FPe87r/5t9riJ8QT1Czd2m+39yYJ7frnd1rjuh7
FAvqyULP9uSZN5FBR7+YPR3rWfUK+D5C1Nq2BJ+XUwdIZpZn+r848Vntgy631h+x
3O7tHO1aEhc0WXEIv0UCAwEAAaN1MHMwDgYDVR0PAQH/BAQDAgeAMB0GA1UdJQQW
MBQGCCsGAQUFBwMCBggrBgEFBQcDATAfBgNVHSMEGDAWgBQtYaKPgCqk7O+6OWME
n3tdYZsl9DAhBgNVHREEGjAYhwR/AAABhxAAAAAAAAAAAAAAAAAAAAABMA0GCSqG
SIb3DQEBCwUAA4IBAQAvFuuaf+xbL8pZkJ8g8yYHlqA1xFOebTUmzBPZ1c1tVIkf
KaKPTmgENmp4iiBgL/yptLohxQoJG2jr5BQqFialbs+A0lwLUe1PaEu0QE8x8ko4
BZl0xFVJ7Lm9/WcMDbXclIdnz2J/3Oqnv3ltEW/c0VKydHFMWds/P4DPNX50baI0
eDTPs4f3ZQDjkiV0o7gS86SbeGl92ByT/1nhz82gysayo/H2Ywg6j5hyjtb0gq7E
sw1Z28Ia6wLxhTBxqtOmSB4N7Y3E8C868lmiCoK4ETFFAiFHZU7gmlIXVul3X6IT
+i3TggXpd0XjNatkK2EfmLOv1bQNPz5i0oyYtJBS
-----END CERTIFICATE-----
`

	expectedCertChain := `-----BEGIN CERTIFICATE-----
MIIDGjCCAgKgAwIBAgIDEjRWMA0GCSqGSIb3DQEBCwUAMBQxEjAQBgNVBAMTCXRl
c3QtaW50MjAeFw0yNDA0MDIxNjA4NTVaFw0yNTA0MDIxNjA4NTVaMBQxEjAQBgNV
BAMTCXRlc3QtbGVhZjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALg2
kdP54gsbku7vUfIYf59H5VQLoDXoCZpTO7lxsHEh9QveViINEgYJ5Ycq8YijRR9J
W2oAWSLdE+mEQ+kvzxjqIwcZgCq0cGwErXKaPZCNnQKiV0cGCgWBPyqZcE1noiSE
5FqWBK0FDXmdiA5p2TSNfu6okJqVB/u0Bxbxqq+/V+aQ5KuFjwmIBT+/2kxwBma2
Si/SDJts/H9izYcjTuxyJ1Cq9FPe87r/5t9riJ8QT1Czd2m+39yYJ7frnd1rjuh7
FAvqyULP9uSZN5FBR7+YPR3rWfUK+D5C1Nq2BJ+XUwdIZpZn+r848Vntgy631h+x
3O7tHO1aEhc0WXEIv0UCAwEAAaN1MHMwDgYDVR0PAQH/BAQDAgeAMB0GA1UdJQQW
MBQGCCsGAQUFBwMCBggrBgEFBQcDATAfBgNVHSMEGDAWgBQtYaKPgCqk7O+6OWME
n3tdYZsl9DAhBgNVHREEGjAYhwR/AAABhxAAAAAAAAAAAAAAAAAAAAABMA0GCSqG
SIb3DQEBCwUAA4IBAQAvFuuaf+xbL8pZkJ8g8yYHlqA1xFOebTUmzBPZ1c1tVIkf
KaKPTmgENmp4iiBgL/yptLohxQoJG2jr5BQqFialbs+A0lwLUe1PaEu0QE8x8ko4
BZl0xFVJ7Lm9/WcMDbXclIdnz2J/3Oqnv3ltEW/c0VKydHFMWds/P4DPNX50baI0
eDTPs4f3ZQDjkiV0o7gS86SbeGl92ByT/1nhz82gysayo/H2Ywg6j5hyjtb0gq7E
sw1Z28Ia6wLxhTBxqtOmSB4N7Y3E8C868lmiCoK4ETFFAiFHZU7gmlIXVul3X6IT
+i3TggXpd0XjNatkK2EfmLOv1bQNPz5i0oyYtJBS
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIDFzCCAf+gAwIBAgIDEjRWMA0GCSqGSIb3DQEBCwUAMBQxEjAQBgNVBAMTCXRl
c3QtaW50MTAeFw0yNDA0MDIxNjA4NTVaFw0zNDA0MDIxNjA4NTVaMBQxEjAQBgNV
BAMTCXRlc3QtaW50MjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBANYX
Dy2Uk67tIDsrVNnfTUJRFVCFNFgAEcFjTUZDLNPJiJy3i4QTDn+4oZpjlZjSC0l1
QabxqQGgAycwA0TBJykKCKWN1uCucVBvXW8s4OdF/2HQIw13HZWZJOTJvMK+kfIx
2/4cUr7lXl7QoC01VnhHQW1cLrzjfs2WZ+Nv1VGOxFx1S8yjNc4CvV4pKzeeANxE
cgjAdMqwaT9uLKJ2fa/6jnlf4xVrTaO96LlPaiI/qn5Q/xdcdizQ3SDREV5BXe4V
MOVoI0K9uoluy98RZvGG0G92PN7gM1BOfc8S/2Qaulq5eKzMShK4LmE2sWNniWjd
4IEysKsOnllIxHae9wkCAwEAAaNyMHAwHQYDVR0lBBYwFAYIKwYBBQUHAwIGCCsG
AQUFBwMBMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFC1hoo+AKqTs77o5YwSf
e11hmyX0MB8GA1UdIwQYMBaAFM6L9z31AGFduhTwmjXne7FBj9AWMA0GCSqGSIb3
DQEBCwUAA4IBAQCQhr+0m+UukGYqhwZSIlgL8YKulwMz0SNhyMkEcrxEl03Nx7SZ
KgwZH+bxgykYocL0RxAZcjBCmcoLyy3Ebl743s4eRppBXQc+kYzYyaWEUpUxNoq2
tFi+yqruQbkNSQ/rzKrRrwZAD8vP3mcUScjx9UNwXAbr4NbA10US1WlWYA17v4kf
HXGhV997/HNLNyUYizePeeU4DrdVNgI3hdy4mpyWRDWc6fyaKbeFJbWo+KHqT5zB
fTEBltyrpTqCdq0nXLwIzR+J4brtBH/LE2Wo9kp5bh4xUJQeybc46QeAyAvbV812
e4t6jdnEGzymy5pzbGjtEp+gW4zaSGLBGL3V
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIDFTCCAf2gAwIBAgIDEjRWMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNVBAMTB3Rl
c3QtY2EwHhcNMjQwNDAyMTYwODU1WhcNMzQwNDAyMTYwODU1WjAUMRIwEAYDVQQD
Ewl0ZXN0LWludDEwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCtCcWc
98ConFAuw6PpsuljvSDfoee9UqFCpfxeIHdTu1wcxTtdRROvbwbdDfc0UYsOIRlR
J+zVJdPJBS2n/lCiXVjfWgCmPQpSQhXyJAmD1VwgDrT4YFm31RCQmkQlQCaY7s4Y
KG/eawzaSA0CYBjMS4ss58zFJeZyPB5y0OMb6Cu7Q5MsJBkwpdjADuT5otfZPIDG
wB1N7vRtS37ajPR3cxrR7+jehcjEiZBeiW0tTdTgH1TFpoDqxCITkAyqu40AhG1i
xH4DZs285LaeqVZnLiW2CwlKAMXaHwxL8FmhoFN6+FwXq/fBE2oUFhI5V00wp99Y
WucRgoHZHjPLm4j9AgMBAAGjcjBwMB0GA1UdJQQWMBQGCCsGAQUFBwMCBggrBgEF
BQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBTOi/c99QBhXboU8Jo153ux
QY/QFjAfBgNVHSMEGDAWgBQAsAb5gn4MZ6N1PJP1Pr6diD7hkjANBgkqhkiG9w0B
AQsFAAOCAQEAhH5ygm9o2hxMJl4AuKfq3S2AKtDho+gW7D7XDTAyoCDjxskzTagz
DNi1sEkgOOt+pYPdzZLqAPb5qE+jvHpXSrIiG+wYEZ3I9bMjOfXFFh9wfgNNpiCO
KCP6c6XGJuAzDLLOqe726TjXFwh2rtCs6IXl6MZBfYEWpCYgbJytuGiVbEy/4zHu
REXmmUVKBTT7nq9zMGqK7rkyEJeq51uGrO4NfkgDEPJqN7RQySMi8drXFcjjyvEo
vfXoEug1sLeGyrEMD+8wfvhFJDtFIxCCu6gRIn6H8QMouNTIZBiZhtHPD9La5vh/
RAqYxnujJ1Gw4ZpQExiLxPq0MdP3NLOibg==
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIC8jCCAdqgAwIBAgIDEjRWMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNVBAMTB3Rl
c3QtY2EwHhcNMjQwNDAyMTYwODU0WhcNMzQwNDAyMTYwODU0WjASMRAwDgYDVQQD
Ewd0ZXN0LWNhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAt0R6suEJ
zlRDkMUKUEFIRRnqPmbDUM+h5k4tc5bgAJgX1EGf/lVBJ4gzUGzYayc9qyIcsKqI
/wyxsEm9SOnqR5lQkE/dJ3BsiSV/+wts6OX86KLWn4gHFm1xzl3xAj0/7w0qrEGj
5ASEF+RsfQq+oY/jglZCRWaVq23F77L6NeOFCicEKCRRLKClwXFFrGErwoUk3ef1
CJ7GD1C+7Pk4uHQC4BYttcSyVYfTn4fdYMEQtEY3hAWRsfZqJ/epRvxFFaDXnfGL
PWoj+IYRx0YWsV6FY8rqyat8PGtvY4JR5RdF9nNIKapV3n3W98tc6EiXBZybULsd
5z9PHU0hDabSxwIDAQABo1EwTzAdBgNVHSUEFjAUBggrBgEFBQcDAgYIKwYBBQUH
AwEwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUALAG+YJ+DGejdTyT9T6+nYg+
4ZIwDQYJKoZIhvcNAQELBQADggEBABgkDtHj51xeHwFfSYQmUXnTQl59VCXGdulU
Fx8yfI5aMJzWR0SGTnJ8/VpBUZi6VTTz45qvi8xJgnpF8SLtKjXQlbqIerO7KL+M
7EnK2O1IGMKPboGM3pgJJQ7jS6aPObtFvuLUwECYoFw6dEzQkauzZjNA5FjWPImM
9VonFvAOpA45r9/b5liZ/Lg7gfdOtlLYUpCU1bPtem4v60oFmKh5IMOdLDVCgcga
HXlyr1Q1xkPwnHMt1aOPJPuMs1DSfbhP40bUvYh3gU5B7XpUpaHxlltm/h9/CsPE
z9rzlA+Co/z78Wn/LtvjVrxJj4QHcfXhiIltAaAUnJP+kZ0+3I0=
-----END CERTIFICATE-----
`

	cases := []struct {
		desc        string
		cert        string
		expectedErr bool
	}{
		{
			desc:        "order: root, intermediate1, intermediate2, server certs",
			cert:        rootCACert + intermediateCert1 + intermediateCert2 + serverCert,
			expectedErr: false,
		},
		{
			desc:        "order: root, server, intermediate1, intermediate2, certs",
			cert:        rootCACert + serverCert + intermediateCert1 + intermediateCert2,
			expectedErr: false,
		},
		{
			desc:        "order: intermediate1, intermediate2, root, server certs",
			cert:        intermediateCert1 + intermediateCert2 + rootCACert + serverCert,
			expectedErr: false,
		},
		{
			desc:        "order: intermediate1, intermediate2, server, root certs",
			cert:        intermediateCert1 + intermediateCert2 + serverCert + rootCACert,
			expectedErr: false,
		},
		{
			desc:        "order: server, root, intermediate1, intermediate2 certs",
			cert:        serverCert + rootCACert + intermediateCert1 + intermediateCert2,
			expectedErr: false,
		},
		{
			desc:        "order: server, intermediate1, intermediate2, root certs",
			cert:        serverCert + intermediateCert1 + intermediateCert2 + rootCACert,
			expectedErr: false,
		},

		{
			desc:        "order: root, intermediate2, intermediate1, server certs",
			cert:        rootCACert + intermediateCert1 + intermediateCert2 + serverCert,
			expectedErr: false,
		},
		{
			desc:        "order: root, server, intermediate2, intermediate1 certs",
			cert:        rootCACert + serverCert + intermediateCert1 + intermediateCert2,
			expectedErr: false,
		},
		{
			desc:        "order: intermediate2, intermediate1, root, server certs",
			cert:        intermediateCert1 + intermediateCert2 + rootCACert + serverCert,
			expectedErr: false,
		},
		{
			desc:        "order: intermediate2, intermediate1, server, root certs",
			cert:        intermediateCert1 + intermediateCert2 + serverCert + rootCACert,
			expectedErr: false,
		},
		{
			desc:        "order: server, root, intermediate2, intermediate1 certs",
			cert:        serverCert + rootCACert + intermediateCert1 + intermediateCert2,
			expectedErr: false,
		},
		{
			desc:        "order: server, intermediate2, intermediate1, root certs",
			cert:        serverCert + intermediateCert1 + intermediateCert2 + rootCACert,
			expectedErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			certChain, err := fetchCertChains([]byte(tc.cert))
			if tc.expectedErr && err == nil || !tc.expectedErr && err != nil {
				t.Fatalf("expected error: %v, got error: %v", tc.expectedErr, err)
			}
			if string(certChain) != expectedCertChain {
				t.Fatalf(cmp.Diff(expectedCertChain, string(certChain)))
			}
		})
	}
}

func TestFetchCertChainWarning(t *testing.T) {
	rootCACert := `
-----BEGIN CERTIFICATE-----
MIIBeTCCAR6gAwIBAgIRAM3RAPH7k1Q+bICMC0mzKhkwCgYIKoZIzj0EAwIwGjEY
MBYGA1UEAxMPRXhhbXBsZSBSb290IENBMB4XDTIwMTIwMzAwMTAxNFoXDTMwMTIw
MTAwMTAxNFowGjEYMBYGA1UEAxMPRXhhbXBsZSBSb290IENBMFkwEwYHKoZIzj0C
AQYIKoZIzj0DAQcDQgAE1/AGExuSemtxPRzFECpefowtkcOQr7jaq355kfb2hUR2
LnMn+71fD4mZmMXT0kuxgeE2zC2CxOHdoJ/FmcQJxaNFMEMwDgYDVR0PAQH/BAQD
AgEGMBIGA1UdEwEB/wQIMAYBAf8CAQEwHQYDVR0OBBYEFKTuLl7BATUYGD6ZeUV3
2f8UAWoqMAoGCCqGSM49BAMCA0kAMEYCIQDEz2XKXPb0Q/Y40Gtxo8r6sa0Ra6U0
fpTPteqfpl8iGQIhAOo8tpUYiREVSYZu130fN0Gvy4WmJMFAi7JrVeSnZ7uP
-----END CERTIFICATE-----
`

	serverCert := `
-----BEGIN CERTIFICATE-----
MIIBwjCCAWmgAwIBAgIQGIPRUsQ/sFI1fkxZHCSU6jAKBggqhkjOPQQDAjAkMSIw
IAYDVQQDExlFeGFtcGxlIEludGVybWVkaWF0ZSBDQSAxMB4XDTIwMTIwMzAwMTAz
NloXDTIwMTIwNDAwMTAzNlowFjEUMBIGA1UEAxMLZXhhbXBsZS5jb20wWTATBgcq
hkjOPQIBBggqhkjOPQMBBwNCAAS0FvMzMHAfc6mOIEgijRngeRcNaDdp63AbCVeJ
tuKNX7j4KLbkQcACj6g+hblJu4NCJChFmeEYf8b7xw+q0dPOo4GKMIGHMA4GA1Ud
DwEB/wQEAwIHgDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwHQYDVR0O
BBYEFIRRQ0915ExZz30TeVhCpwgP3SEYMB8GA1UdIwQYMBaAFH7YljRWGOtaytSe
D5Xx2B5kE4YnMBYGA1UdEQQPMA2CC2V4YW1wbGUuY29tMAoGCCqGSM49BAMCA0cA
MEQCIH9NxXnWaip9fZyv9VJcfFz7tcdxTq10SrTO7gKhyJkpAiAljZFFK687kc6J
kzqEt441cQasPp5ohL5U4cJN6lAuwA==
-----END CERTIFICATE-----
`

	expectedCert := `-----BEGIN CERTIFICATE-----
MIIBwjCCAWmgAwIBAgIQGIPRUsQ/sFI1fkxZHCSU6jAKBggqhkjOPQQDAjAkMSIw
IAYDVQQDExlFeGFtcGxlIEludGVybWVkaWF0ZSBDQSAxMB4XDTIwMTIwMzAwMTAz
NloXDTIwMTIwNDAwMTAzNlowFjEUMBIGA1UEAxMLZXhhbXBsZS5jb20wWTATBgcq
hkjOPQIBBggqhkjOPQMBBwNCAAS0FvMzMHAfc6mOIEgijRngeRcNaDdp63AbCVeJ
tuKNX7j4KLbkQcACj6g+hblJu4NCJChFmeEYf8b7xw+q0dPOo4GKMIGHMA4GA1Ud
DwEB/wQEAwIHgDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwHQYDVR0O
BBYEFIRRQ0915ExZz30TeVhCpwgP3SEYMB8GA1UdIwQYMBaAFH7YljRWGOtaytSe
D5Xx2B5kE4YnMBYGA1UdEQQPMA2CC2V4YW1wbGUuY29tMAoGCCqGSM49BAMCA0cA
MEQCIH9NxXnWaip9fZyv9VJcfFz7tcdxTq10SrTO7gKhyJkpAiAljZFFK687kc6J
kzqEt441cQasPp5ohL5U4cJN6lAuwA==
-----END CERTIFICATE-----
`

	var buf bytes.Buffer
	klog.SetOutput(&buf)
	klog.LogToStderr(false)
	defer klog.LogToStderr(true)
	// certificate chain missing intermediate certificate
	cert := serverCert + rootCACert
	certChain, err := fetchCertChains([]byte(cert))
	if err != nil {
		t.Fatalf("fetchCertChains() error = %v, expected nil", err)
	}
	if string(certChain) != expectedCert {
		t.Fatalf(cmp.Diff(expectedCert, string(certChain)))
	}

	klog.Flush()
	klog.SetOutput(&bytes.Buffer{}) // prevent further writes into buf
	capturedOutput := buf.String()
	if !strings.Contains(capturedOutput, `certificate chain is not complete due to missing intermediate/root certificates in the cert from key vault`) {
		t.Fatalf("expected warning, got: %s", capturedOutput)
	}
}

func TestGetSecretsStoreObjectContent(t *testing.T) {
	cases := []struct {
		desc        string
		parameters  map[string]string
		secrets     map[string]string
		expectedErr bool
	}{
		{
			desc:        "keyvault name not provided",
			parameters:  map[string]string{},
			expectedErr: true,
		},
		{
			desc: "tenantID not provided",
			parameters: map[string]string{
				"keyvaultName": "testKV",
			},
			expectedErr: true,
		},
		{
			desc: "usePodIdentity not a boolean as expected",
			parameters: map[string]string{
				"keyvaultName":   "testKV",
				"tenantId":       "tid",
				"usePodIdentity": "tru",
			},
			expectedErr: true,
		},
		{
			desc: "useVMManagedIdentity not a boolean as expected",
			parameters: map[string]string{
				"keyvaultName":         "testKV",
				"tenantId":             "tid",
				"usePodIdentity":       "false",
				"useVMManagedIdentity": "tru",
			},
			expectedErr: true,
		},
		{
			desc: "invalid cloud name",
			parameters: map[string]string{
				"keyvaultName": "testKV",
				"tenantId":     "tid",
				"cloudName":    "AzureCloud",
			},
			expectedErr: true,
		},
		{
			desc: "check azure cloud env file path is set",
			parameters: map[string]string{
				"keyvaultName":     "testKV",
				"tenantId":         "tid",
				"cloudName":        "AzureStackCloud",
				"cloudEnvFileName": "/etc/kubernetes/akscustom.json",
			},
			expectedErr: true,
		},
		{
			desc: "objects array not set",
			parameters: map[string]string{
				"keyvaultName":         "testKV",
				"tenantId":             "tid",
				"useVMManagedIdentity": "true",
			},
			expectedErr: true,
		},
		{
			desc: "objects not configured as an array",
			parameters: map[string]string{
				"keyvaultName":         "testKV",
				"tenantId":             "tid",
				"useVMManagedIdentity": "true",
				"objects": `
        - |
          objectName: secret1
          objectType: secret
          objectVersion: ""`,
			},
			expectedErr: true,
		},
		{
			desc: "objects array is empty",
			parameters: map[string]string{
				"keyvaultName":         "testKV",
				"tenantId":             "tid",
				"useVMManagedIdentity": "true",
				"objects": `
      array:`,
			},
			expectedErr: false,
		},
		{
			desc: "invalid object format",
			parameters: map[string]string{
				"keyvaultName":         "testKV",
				"tenantId":             "tid",
				"useVMManagedIdentity": "true",
				"objects": `
      array:
        - |
          objectName: secret1
          objectType: secret
          objectFormat: pkcs
          objectVersion: ""`,
			},
			expectedErr: true,
		},
		{
			desc: "invalid object encoding",
			parameters: map[string]string{
				"keyvaultName":         "testKV",
				"tenantId":             "tid",
				"useVMManagedIdentity": "true",
				"objects": `
      array:
        - |
          objectName: secret1
          objectType: secret
          objectEncoding: utf-16
          objectVersion: ""`,
			},
			expectedErr: true,
		},
		{
			desc: "error fetching from keyvault",
			parameters: map[string]string{
				"keyvaultName": "testKV",
				"tenantId":     "tid",
				"objects": `
      array:
        - |
          objectName: secret1
          objectType: secret
          objectVersion: ""`,
			},
			secrets: map[string]string{
				"clientid":     "AADClientID",
				"clientsecret": "AADClientSecret",
			},
			expectedErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			p := NewProvider(false, false)

			_, err := p.GetSecretsStoreObjectContent(testContext(t), tc.parameters, tc.secrets, 0420)
			if tc.expectedErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestGetCurve(t *testing.T) {
	cases := []struct {
		crv           azkeys.JSONWebKeyCurveName
		expectedCurve elliptic.Curve
		expectedErr   error
	}{
		{
			crv:           azkeys.JSONWebKeyCurveNameP256,
			expectedCurve: elliptic.P256(),
			expectedErr:   nil,
		},
		{
			crv:           azkeys.JSONWebKeyCurveNameP384,
			expectedCurve: elliptic.P384(),
			expectedErr:   nil,
		},
		{
			crv:           azkeys.JSONWebKeyCurveNameP521,
			expectedCurve: elliptic.P521(),
			expectedErr:   nil,
		},
		{
			crv:           azkeys.JSONWebKeyCurveNameP256K,
			expectedCurve: nil,
			expectedErr:   fmt.Errorf("curve P-256K is not supported"),
		},
	}

	for _, tc := range cases {
		actual, err := getCurve(tc.crv)
		assert.Equal(t, tc.expectedCurve, actual)
		assert.Equal(t, tc.expectedErr, err)
	}
}

func TestParsePrivateKey(t *testing.T) {
	cases := []struct {
		desc       string
		privateKey string
		checker    func(key interface{})
	}{
		{
			desc: "pkcs1 format rsa private key",
			privateKey: `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0AWQCdeukwkzIKKJNp3DaRe9azBZ8J/NFb2Nczq3Y8xcMDB/
eT7lfMMNYluLQPDzkRN9QHKiz8ei9ynxRiEC/Al2OsdZPdPqNxnBVDsFcD729nof
roBUXRch5dP5amXu5gP628Yu7l8pBoV+lOyyDGkRVHPecegxiVbxtjqhlrwlhRRF
zFGat1CiDq03Gtz1xH/pgaFQzKbTZ1rQE8JcTryZaTYfo5PrUDwhv8PfVHoHMEqp
N54onSoA2JLBeZz7xJvL6pBg0c6OhNCnUYEZBDnyHDBBJJ6FUijKQp6mZNbedi6I
h4QGJYeLP4HaJdPf9aXlChnbbwEaeBeedXzPjwIDAQABAoIBAQDMU7pwwIb8bDvp
IV2v5PTNZIEtKTgez4hNg3vOJG2APHqM5wY/HNWjX5/k7dBxgHtuE/uiczeS6iAb
sPoKDWD2GYElKSxyO5ZCeyzXxIWKBH7mCXzXFbxIF/G24yiJJwiqrFwaxabRg20z
t6pnM7uLzyQzlQB5WD5YDauseBjCidOb9Ri92rNnW+g/H6YZtI3beEAg/gTD/rP5
5ucRjp6rmbwZ90VA8O8frYpV7ofXxpekvD1Q8Vrk3XwBubq01tg7a8Ugal44ApaO
X7e/X6xw6bwISe1zCCm1YKPjNKrhqcE4ujHAghVbST+sb9XiNk0TvMb1qF/dh+zx
7iCalqxZAoGBAPjNNeay5hApmoQdiyyfPwR/RzAH9eSam8Wn5pJzQz2nLFGbozmn
fO5jvI06ACumgS8LZiIGmBlbPrKQtL91Z1ftwKgBGCgqI9BpskHDP04Z/QNDlRNA
gz3qtANTmKl69RvBv82QyLzsWwcLJhVxgMTsNPnd4Z7iB9soB2mG0iFNAoGBANYK
TzDvwM6oCmtRn38zgrX+6jc2ptCAuQYeL7pn51TbljcP0XkJ8LkFaBK3lzG1NUhL
DgOcEbFEtZpwpYgDYlbVwyt3m3QUQDqm93J86pf1J1jWF81PYgUJaS7/8lBzDUiK
+PZ4XV6zYBFxUCy2yh5rxsyhBoxLV0oRD+wbGkZLAoGBAM0izYVYDY5X7xltDnoN
FrVLh9NXTOteen7+j4JCXLdxpX3n2C3KJZycSTMcFlXnI+449M2rKC8H52rtGsod
L8b0tXsP4+4ByKOm8h18sS5hCRZu23QTJeKgKCnx/BYI1h07ozwHWytBqU/mZlEZ
03UJ2CgIRGVusdGFcI8WZRylAoGASMxE1u4Uc7UvpgSi7M6GPIQxAQpzfiLpyyzl
Ks9AHNp6osucgUBiQWuXVBZhNCTftHDimVOxqMsnwRljE3mjLsmRke0iUD67Abfc
HXJjD7/v3AUlH01Kl0/2GGgw8C/RasTpnFqf1x/HIueZTzv0Tph1iw+RfJH7ZFOd
SL6HFzUCgYBpod9mhdljh4VsysZqeFfbliESb+ue7PVZb/+X9lJ7DATIq4/farhi
9YkknRAqJmKEcsomn5b35Kj0QBwiDdEE7tISdkj36jgoaz6pyyuj9ys1BlCN0fBH
2QJGFpJ3TBKqIo2iGmPHLXZVFajPF/KNVDVNlc9EUIraVgmWwDuZ7g==
-----END RSA PRIVATE KEY-----
`,
			checker: func(key interface{}) {
				_, ok := key.(*rsa.PrivateKey)
				assert.True(t, ok)
			},
		},
		{
			desc: "pkcs8 format rsa private key",
			privateKey: `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDQBZAJ166TCTMg
ook2ncNpF71rMFnwn80VvY1zOrdjzFwwMH95PuV8ww1iW4tA8PORE31AcqLPx6L3
KfFGIQL8CXY6x1k90+o3GcFUOwVwPvb2eh+ugFRdFyHl0/lqZe7mA/rbxi7uXykG
hX6U7LIMaRFUc95x6DGJVvG2OqGWvCWFFEXMUZq3UKIOrTca3PXEf+mBoVDMptNn
WtATwlxOvJlpNh+jk+tQPCG/w99UegcwSqk3niidKgDYksF5nPvEm8vqkGDRzo6E
0KdRgRkEOfIcMEEknoVSKMpCnqZk1t52LoiHhAYlh4s/gdol09/1peUKGdtvARp4
F551fM+PAgMBAAECggEBAMxTunDAhvxsO+khXa/k9M1kgS0pOB7PiE2De84kbYA8
eoznBj8c1aNfn+Tt0HGAe24T+6JzN5LqIBuw+goNYPYZgSUpLHI7lkJ7LNfEhYoE
fuYJfNcVvEgX8bbjKIknCKqsXBrFptGDbTO3qmczu4vPJDOVAHlYPlgNq6x4GMKJ
05v1GL3as2db6D8fphm0jdt4QCD+BMP+s/nm5xGOnquZvBn3RUDw7x+tilXuh9fG
l6S8PVDxWuTdfAG5urTW2DtrxSBqXjgClo5ft79frHDpvAhJ7XMIKbVgo+M0quGp
wTi6McCCFVtJP6xv1eI2TRO8xvWoX92H7PHuIJqWrFkCgYEA+M015rLmECmahB2L
LJ8/BH9HMAf15JqbxafmknNDPacsUZujOad87mO8jToAK6aBLwtmIgaYGVs+spC0
v3VnV+3AqAEYKCoj0GmyQcM/Thn9A0OVE0CDPeq0A1OYqXr1G8G/zZDIvOxbBwsm
FXGAxOw0+d3hnuIH2ygHaYbSIU0CgYEA1gpPMO/AzqgKa1GffzOCtf7qNzam0IC5
Bh4vumfnVNuWNw/ReQnwuQVoEreXMbU1SEsOA5wRsUS1mnCliANiVtXDK3ebdBRA
Oqb3cnzql/UnWNYXzU9iBQlpLv/yUHMNSIr49nhdXrNgEXFQLLbKHmvGzKEGjEtX
ShEP7BsaRksCgYEAzSLNhVgNjlfvGW0Oeg0WtUuH01dM6156fv6PgkJct3GlfefY
LcolnJxJMxwWVecj7jj0zasoLwfnau0ayh0vxvS1ew/j7gHIo6byHXyxLmEJFm7b
dBMl4qAoKfH8FgjWHTujPAdbK0GpT+ZmURnTdQnYKAhEZW6x0YVwjxZlHKUCgYBI
zETW7hRztS+mBKLszoY8hDEBCnN+IunLLOUqz0Ac2nqiy5yBQGJBa5dUFmE0JN+0
cOKZU7GoyyfBGWMTeaMuyZGR7SJQPrsBt9wdcmMPv+/cBSUfTUqXT/YYaDDwL9Fq
xOmcWp/XH8ci55lPO/ROmHWLD5F8kftkU51IvocXNQKBgGmh32aF2WOHhWzKxmp4
V9uWIRJv657s9Vlv/5f2UnsMBMirj99quGL1iSSdEComYoRyyiaflvfkqPRAHCIN
0QTu0hJ2SPfqOChrPqnLK6P3KzUGUI3R8EfZAkYWkndMEqoijaIaY8ctdlUVqM8X
8o1UNU2Vz0RQitpWCZbAO5nu
-----END PRIVATE KEY-----
`,
			checker: func(key interface{}) {
				_, ok := key.(*rsa.PrivateKey)
				assert.True(t, ok)
			},
		},
		{
			desc: "ec private key",
			privateKey: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIB79Z1qMNIo69fgeElbOqLaqZpM79lUUo0j7h9swUakEoAoGCCqGSM49
AwEHoUQDQgAEO+YO1IMQkGJlsX59o3+qpamhHxOOVKUbF8m69XbYo7RpIxPr/COw
PxrUsXyXty7ERMp5QNyxjMWS+0w93FrAIw==
-----END EC PRIVATE KEY-----
`,
			checker: func(key interface{}) {
				_, ok := key.(*ecdsa.PrivateKey)
				assert.True(t, ok)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			pemBlock, _ := pem.Decode([]byte(tc.privateKey))

			actual, err := parsePrivateKey(pemBlock.Bytes)
			assert.NoError(t, err)
			tc.checker(actual)
		})
	}
}

func TestSplitCertAndKey(t *testing.T) {
	rootCACert := `-----BEGIN CERTIFICATE-----
MIIBeTCCAR6gAwIBAgIRAM3RAPH7k1Q+bICMC0mzKhkwCgYIKoZIzj0EAwIwGjEY
MBYGA1UEAxMPRXhhbXBsZSBSb290IENBMB4XDTIwMTIwMzAwMTAxNFoXDTMwMTIw
MTAwMTAxNFowGjEYMBYGA1UEAxMPRXhhbXBsZSBSb290IENBMFkwEwYHKoZIzj0C
AQYIKoZIzj0DAQcDQgAE1/AGExuSemtxPRzFECpefowtkcOQr7jaq355kfb2hUR2
LnMn+71fD4mZmMXT0kuxgeE2zC2CxOHdoJ/FmcQJxaNFMEMwDgYDVR0PAQH/BAQD
AgEGMBIGA1UdEwEB/wQIMAYBAf8CAQEwHQYDVR0OBBYEFKTuLl7BATUYGD6ZeUV3
2f8UAWoqMAoGCCqGSM49BAMCA0kAMEYCIQDEz2XKXPb0Q/Y40Gtxo8r6sa0Ra6U0
fpTPteqfpl8iGQIhAOo8tpUYiREVSYZu130fN0Gvy4WmJMFAi7JrVeSnZ7uP
-----END CERTIFICATE-----
`

	intermediateCert := `-----BEGIN CERTIFICATE-----
MIIBozCCAUmgAwIBAgIRANEldEfXaQ+L2M1ahC6w4vAwCgYIKoZIzj0EAwIwGjEY
MBYGA1UEAxMPRXhhbXBsZSBSb290IENBMB4XDTIwMTIwMzAwMTAyNFoXDTMwMTIw
MTAwMTAyNFowJDEiMCAGA1UEAxMZRXhhbXBsZSBJbnRlcm1lZGlhdGUgQ0EgMTBZ
MBMGByqGSM49AgEGCCqGSM49AwEHA0IABOhTE8r5NpDIDF/6VLgPT+//0IR59Uzn
78JfV54E0qFA21khrcqc20/RJD+lyUv313gYQD9SxBXXxcGtl1OJ0s2jZjBkMA4G
A1UdDwEB/wQEAwIBBjASBgNVHRMBAf8ECDAGAQH/AgEAMB0GA1UdDgQWBBR+2JY0
VhjrWsrUng+V8dgeZBOGJzAfBgNVHSMEGDAWgBSk7i5ewQE1GBg+mXlFd9n/FAFq
KjAKBggqhkjOPQQDAgNIADBFAiB9EQB+siuNboL7k78CUzhZJ+5lD0cXUpGYGWYT
rxcX6QIhALGptitzrZ4z/MDMBPkan48bqk6O08e1tQ9dJOIoEKq7
-----END CERTIFICATE-----
`

	serverCert := `-----BEGIN CERTIFICATE-----
MIIBwjCCAWmgAwIBAgIQGIPRUsQ/sFI1fkxZHCSU6jAKBggqhkjOPQQDAjAkMSIw
IAYDVQQDExlFeGFtcGxlIEludGVybWVkaWF0ZSBDQSAxMB4XDTIwMTIwMzAwMTAz
NloXDTIwMTIwNDAwMTAzNlowFjEUMBIGA1UEAxMLZXhhbXBsZS5jb20wWTATBgcq
hkjOPQIBBggqhkjOPQMBBwNCAAS0FvMzMHAfc6mOIEgijRngeRcNaDdp63AbCVeJ
tuKNX7j4KLbkQcACj6g+hblJu4NCJChFmeEYf8b7xw+q0dPOo4GKMIGHMA4GA1Ud
DwEB/wQEAwIHgDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwHQYDVR0O
BBYEFIRRQ0915ExZz30TeVhCpwgP3SEYMB8GA1UdIwQYMBaAFH7YljRWGOtaytSe
D5Xx2B5kE4YnMBYGA1UdEQQPMA2CC2V4YW1wbGUuY29tMAoGCCqGSM49BAMCA0cA
MEQCIH9NxXnWaip9fZyv9VJcfFz7tcdxTq10SrTO7gKhyJkpAiAljZFFK687kc6J
kzqEt441cQasPp5ohL5U4cJN6lAuwA==
-----END CERTIFICATE-----
`

	privateKey := `-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC17SizxfpkpsRR
agUl1J9xihKi9vE8GG2KwQl39T6g4rWMof8cwGWJO/72Xo68iGV223jy/nuaTg/Q
zvYpPy6VdwFqw1X3kM2c+wSkCqzvAquF+2V54Mwo+eTxW9TqOJ0/LpI7o21uny0l
yGtt8zBlbt7bL49XB4o3xDA0QJA9/KAG1fyM/3cIbqhi8r0UNLjlTVRwpR3FMt9e
ZK85E+jQcURcs29nGZ2NKys/ShJtqwl8yvTYNlvn/oxLEJsn8Wc/B2pdAf9wzqtA
MvDYizmJafRBc8Rhkyphu0wAmhjRQRUdUeAhRv8e5xD5ISS23qQbewgYqwouK7KZ
kajNEU2rAgMBAAECggEBAK9MJxUapkxH+RDt1KoAN+aigZSv2ADtFNhHa0VAdal2
6jLpgbWFmhDjU6i3slfuIb6meePC3PzxTQIJ+l4COHPi6OWj9PkIeWdS5MTgWIQx
kW8Xr08CEhdFu5npv7408SgJSvTWY8Lc9BbdCM84LqD+dRTEvhzA8ikMDNq8f4CJ
hLreFUUl/udHacpMdE8mpB6vgCUliZEjBlHHC9qD2mDKgWb0cm4jkO9PcHxz8CXL
szcRV2vqTwvsJcZWcJwTzjhFxq/lUZrgbwpn60iKlov3BCRoTJBppOXi01giom3v
Wz7Y7DoFbHfizh6jyBrf3ODhKJQ3CGvS65QCS0aJ/kECgYEA4JuGC9DpQYmlzWbV
0CqJYnTcZKqcPQx/A1QZDKot0VWqF61vZIku5XuoGKGfY3eLwVZJJZqxoTlVTbuT
nNzYJe+EHzftRoUxUqXZtIh9VdirJMwCu4RMdwk705FA8+8FcTKXarKWBbAzUmFi
iINR2rlRJHVyh2cOA9hWPbEXX0sCgYEAz1qAYUIMBGnccY9mALGArWTmtyNN3DcB
9tl3/5SzfL1NlcOWsBXdZL61oTl9LhOjqHGZbf164/uFuKNHTsT1E63180UKujmV
TbHL6N6MrMctaJfgru3+XprTMd5pwjzd8huX603OtS8Gvn5gKdBRkG1ZI8CrfTl6
sJI9YRvl7yECgYEAjUIiptHHsVkhdrIDLL1j1BEM/x6xzk9KnkxIyMdKs4n9xJBm
K0N/xAHmMT+Mn6DyuzBKJqVIq84EETQ0XQYjxpABdyTUTHK+F22JItpogRIYaLcJ
zOcitAaRoriKsh+UO6IGyqrwYTl0vY3Ty2lTlIzSNGzND81HajGn43q56UsCgYEA
pGqArZ3vZXiDgdBQ82/MNrFxd/oYfOtpNVFPI2vHvrtkT8KdM9bCjGXkI4kwR17v
QFuDa4G49hm0+KkPm9f09LvV8CXo0a1jRA4dP/Nn3IC68tqrIEo6js15dWuEtK4K
1zUmC0DRDT3SvS38FmvGoRzzt7PIxyzSqjvrS5sRgcECgYAQ6b0YsM4p+89s4ALK
BPfGIKpoIEMKUcwiT3ovRrwIu1vbu70WRcYAi5do6rwOakp3FyUcQznkeZEOAQmc
xrBy8R64vg83WMuRITAqY6vartSa3oehqUHW0YbhGDVEtSrolXEs5elArUHbpYnX
SIVZww73PTGisLmXfIvKvr8GBA==
-----END PRIVATE KEY-----
`

	cases := []struct {
		desc         string
		certAndKey   string
		expectedKey  string
		expectedCert string
	}{
		{
			desc:         "cert and key",
			certAndKey:   serverCert + privateKey,
			expectedCert: serverCert,
			expectedKey:  privateKey,
		},
		{
			desc:         "cert chain",
			certAndKey:   rootCACert + intermediateCert + serverCert + privateKey,
			expectedCert: rootCACert + intermediateCert + serverCert,
			expectedKey:  privateKey,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			cert, key := splitCertAndKey(tc.certAndKey)
			if cert != tc.expectedCert {
				t.Errorf("splitCertAndKey() = \n%q, want \n%q", cert, tc.expectedCert)
			}
			if key != tc.expectedKey {
				t.Errorf("splitCertAndKey() = \n%q, want \n%q", key, tc.expectedKey)
			}
		})
	}
}

func TestGetSecret(t *testing.T) {
	id := azsecrets.ID("https://test.vault.azure.net/secrets/secret1/v1")
	testPFX := "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"
	testCert := `-----BEGIN CERTIFICATE-----
MIIC2DCCAcACCQD9DZdcsr7kJDANBgkqhkiG9w0BAQsFADAuMRYwFAYDVQQDDA1k
ZW1vLnRlc3QuY29tMRQwEgYDVQQKDAtpbmdyZXNzLXRsczAeFw0yMDA1MjIxNjIz
MjZaFw0yMTA1MjIxNjIzMjZaMC4xFjAUBgNVBAMMDWRlbW8udGVzdC5jb20xFDAS
BgNVBAoMC2luZ3Jlc3MtdGxzMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKC
AQEAte0os8X6ZKbEUWoFJdSfcYoSovbxPBhtisEJd/U+oOK1jKH/HMBliTv+9l6O
vIhldtt48v57mk4P0M72KT8ulXcBasNV95DNnPsEpAqs7wKrhftleeDMKPnk8VvU
6jidPy6SO6Ntbp8tJchrbfMwZW7e2y+PVweKN8QwNECQPfygBtX8jP93CG6oYvK9
FDS45U1UcKUdxTLfXmSvORPo0HFEXLNvZxmdjSsrP0oSbasJfMr02DZb5/6MSxCb
J/FnPwdqXQH/cM6rQDLw2Is5iWn0QXPEYZMqYbtMAJoY0UEVHVHgIUb/HucQ+SEk
tt6kG3sIGKsKLiuymZGozRFNqwIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQCSZNbl
WFMjnZuGiFzIZoqfKOp/Dtw48poNJkrxMBJLkiciJD6drXj8vnTQrZUuR25TIiD/
Sq+cO+XVRcJKNP13FjFpRdyHYRtAze4TaQZSJlW2nyfeUtUQkwj2iMhv5l1UMnPG
7+Jxg56aA+IBvyE/tAQVvS0NPdq6Ht2MX6j40ERTXmS8qNdY6qi3ZCEAPazlNsUF
C6nLdViZ/vbQ+l6uEcNsEsPJ6SDTNKLkO9tU7pWCa6QBTncuFLbpDqr3Q+lvx4mv
MVw9RO3NiLuDiPQA0VfKSMrEJJUp4F88pbEax5nq525Rbp85RWkmVoc97UuFS+oc
ldGQrUHVb2/iI1fd
-----END CERTIFICATE-----
`
	testPrivateKey := `-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC17SizxfpkpsRR
agUl1J9xihKi9vE8GG2KwQl39T6g4rWMof8cwGWJO/72Xo68iGV223jy/nuaTg/Q
zvYpPy6VdwFqw1X3kM2c+wSkCqzvAquF+2V54Mwo+eTxW9TqOJ0/LpI7o21uny0l
yGtt8zBlbt7bL49XB4o3xDA0QJA9/KAG1fyM/3cIbqhi8r0UNLjlTVRwpR3FMt9e
ZK85E+jQcURcs29nGZ2NKys/ShJtqwl8yvTYNlvn/oxLEJsn8Wc/B2pdAf9wzqtA
MvDYizmJafRBc8Rhkyphu0wAmhjRQRUdUeAhRv8e5xD5ISS23qQbewgYqwouK7KZ
kajNEU2rAgMBAAECggEBAK9MJxUapkxH+RDt1KoAN+aigZSv2ADtFNhHa0VAdal2
6jLpgbWFmhDjU6i3slfuIb6meePC3PzxTQIJ+l4COHPi6OWj9PkIeWdS5MTgWIQx
kW8Xr08CEhdFu5npv7408SgJSvTWY8Lc9BbdCM84LqD+dRTEvhzA8ikMDNq8f4CJ
hLreFUUl/udHacpMdE8mpB6vgCUliZEjBlHHC9qD2mDKgWb0cm4jkO9PcHxz8CXL
szcRV2vqTwvsJcZWcJwTzjhFxq/lUZrgbwpn60iKlov3BCRoTJBppOXi01giom3v
Wz7Y7DoFbHfizh6jyBrf3ODhKJQ3CGvS65QCS0aJ/kECgYEA4JuGC9DpQYmlzWbV
0CqJYnTcZKqcPQx/A1QZDKot0VWqF61vZIku5XuoGKGfY3eLwVZJJZqxoTlVTbuT
nNzYJe+EHzftRoUxUqXZtIh9VdirJMwCu4RMdwk705FA8+8FcTKXarKWBbAzUmFi
iINR2rlRJHVyh2cOA9hWPbEXX0sCgYEAz1qAYUIMBGnccY9mALGArWTmtyNN3DcB
9tl3/5SzfL1NlcOWsBXdZL61oTl9LhOjqHGZbf164/uFuKNHTsT1E63180UKujmV
TbHL6N6MrMctaJfgru3+XprTMd5pwjzd8huX603OtS8Gvn5gKdBRkG1ZI8CrfTl6
sJI9YRvl7yECgYEAjUIiptHHsVkhdrIDLL1j1BEM/x6xzk9KnkxIyMdKs4n9xJBm
K0N/xAHmMT+Mn6DyuzBKJqVIq84EETQ0XQYjxpABdyTUTHK+F22JItpogRIYaLcJ
zOcitAaRoriKsh+UO6IGyqrwYTl0vY3Ty2lTlIzSNGzND81HajGn43q56UsCgYEA
pGqArZ3vZXiDgdBQ82/MNrFxd/oYfOtpNVFPI2vHvrtkT8KdM9bCjGXkI4kwR17v
QFuDa4G49hm0+KkPm9f09LvV8CXo0a1jRA4dP/Nn3IC68tqrIEo6js15dWuEtK4K
1zUmC0DRDT3SvS38FmvGoRzzt7PIxyzSqjvrS5sRgcECgYAQ6b0YsM4p+89s4ALK
BPfGIKpoIEMKUcwiT3ovRrwIu1vbu70WRcYAi5do6rwOakp3FyUcQznkeZEOAQmc
xrBy8R64vg83WMuRITAqY6vartSa3oehqUHW0YbhGDVEtSrolXEs5elArUHbpYnX
SIVZww73PTGisLmXfIvKvr8GBA==
-----END PRIVATE KEY-----
`

	cases := []struct {
		desc                           string
		initKeyVaultSecret             *azsecrets.SecretBundle
		inputKeyVaultObject            types.KeyVaultObject
		writeCertAndKeyInSeparateFiles bool
		expectedKeyVaultObject         []keyvaultObject
	}{
		{
			desc: "secret",
			initKeyVaultSecret: &azsecrets.SecretBundle{
				ID:    &id,
				Value: to.StringPtr("secret1value"),
			},
			inputKeyVaultObject: types.KeyVaultObject{
				ObjectName: "secret1",
			},
			expectedKeyVaultObject: []keyvaultObject{
				{
					content: "secret1value",
					version: "v1",
				},
			},
		},
		{
			desc: "secret with kid, pem cert and key",
			initKeyVaultSecret: &azsecrets.SecretBundle{
				ID:          &id,
				Value:       to.StringPtr(testCert + testPrivateKey),
				Kid:         to.StringPtr("https://testvault.vault.azure.net/keys/secrets/secret1/v1"),
				ContentType: to.StringPtr("application/x-pem-file"),
			},
			inputKeyVaultObject: types.KeyVaultObject{
				ObjectName: "secret1",
			},
			expectedKeyVaultObject: []keyvaultObject{
				{
					content: testCert + testPrivateKey,
					version: "v1",
				},
			},
		},
		{
			desc: "secret with kid, pfx, objectFormat=pfx",
			initKeyVaultSecret: &azsecrets.SecretBundle{
				ID:          &id,
				Value:       to.StringPtr(testPFX),
				Kid:         to.StringPtr("https://testvault.vault.azure.net/keys/secrets/secret1/v1"),
				ContentType: to.StringPtr("application/x-pkcs12"),
			},
			inputKeyVaultObject: types.KeyVaultObject{
				ObjectName:   "secret1",
				ObjectFormat: "pfx",
			},
			expectedKeyVaultObject: []keyvaultObject{
				{
					content: testPFX,
					version: "v1",
				},
			},
		},
		{
			desc: "secret with kid, pfx, objectFormat=pem",
			initKeyVaultSecret: &azsecrets.SecretBundle{
				ID:          &id,
				Value:       to.StringPtr(testPFX),
				Kid:         to.StringPtr("https://testvault.vault.azure.net/keys/secrets/secret1/v1"),
				ContentType: to.StringPtr("application/x-pkcs12"),
			},
			inputKeyVaultObject: types.KeyVaultObject{
				ObjectName:   "secret1",
				ObjectFormat: "pem",
			},
			expectedKeyVaultObject: []keyvaultObject{
				{
					content: testPrivateKey + testCert,
					version: "v1",
				},
			},
		},
		{
			desc: "secret with kid, pfx, default objectFormat",
			initKeyVaultSecret: &azsecrets.SecretBundle{
				ID:          &id,
				Value:       to.StringPtr(testPFX),
				Kid:         to.StringPtr("https://testvault.vault.azure.net/keys/secrets/secret1/v1"),
				ContentType: to.StringPtr("application/x-pkcs12"),
			},
			inputKeyVaultObject: types.KeyVaultObject{
				ObjectName: "secret1",
			},
			expectedKeyVaultObject: []keyvaultObject{
				{
					content: testPrivateKey + testCert,
					version: "v1",
				},
			},
		},
		{
			desc: "write cert and key in separate files",
			initKeyVaultSecret: &azsecrets.SecretBundle{
				ID:          &id,
				Value:       to.StringPtr(testPFX),
				Kid:         to.StringPtr("https://testvault.vault.azure.net/keys/secrets/secret1/v1"),
				ContentType: to.StringPtr("application/x-pkcs12"),
			},
			inputKeyVaultObject: types.KeyVaultObject{
				ObjectName: "secret1",
			},
			writeCertAndKeyInSeparateFiles: true,
			expectedKeyVaultObject: []keyvaultObject{
				{
					content:        testCert,
					version:        "v1",
					fileNameSuffix: ".crt",
				},
				{
					content:        testPrivateKey,
					version:        "v1",
					fileNameSuffix: ".key",
				},
				{
					content: testPrivateKey + testCert,
					version: "v1",
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := testContext(t)

			p := &provider{writeCertAndKeyInSeparateFiles: tc.writeCertAndKeyInSeparateFiles}
			kvClient := mock_keyvault.NewMockKeyVault(ctrl)
			kvClient.EXPECT().GetSecret(ctx, "secret1", "").Return(
				tc.initKeyVaultSecret, nil,
			)

			objs, err := p.getSecret(ctx, kvClient, tc.inputKeyVaultObject)
			if err != nil {
				t.Fatalf("getSecret() = %v, want nil", err)
			}
			if !reflect.DeepEqual(objs, tc.expectedKeyVaultObject) {
				t.Errorf("getSecret() = \n%v, want \n%v", objs, tc.expectedKeyVaultObject)
			}
		})
	}
}

func TestGetSecretError(t *testing.T) {
	id := azsecrets.ID("https://test.vault.azure.net/secrets/secret1/v1")

	cases := []struct {
		desc                   string
		initKeyVaultSecret     *azsecrets.SecretBundle
		kvError                error
		expectedKeyVaultObject []keyvaultObject
	}{
		{
			desc:               "keyvault get secret error",
			initKeyVaultSecret: &azsecrets.SecretBundle{},
			kvError:            errors.New("keyvault error"),
		},
		{
			desc: "secret value is nil",
			initKeyVaultSecret: &azsecrets.SecretBundle{
				ID: &id,
			},
		},
		{
			desc: "secret id is nil",
			initKeyVaultSecret: &azsecrets.SecretBundle{
				Value: to.StringPtr("test"),
			},
		},
		{
			desc: "secret has kid, not valid pfx",
			initKeyVaultSecret: &azsecrets.SecretBundle{
				ID:          &id,
				Value:       to.StringPtr("invalid"),
				Kid:         to.StringPtr("https://testvault.vault.azure.net/keys/secrets/secret1/v1"),
				ContentType: to.StringPtr("application/x-pkcs12"),
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := testContext(t)

			p := &provider{}
			kvClient := mock_keyvault.NewMockKeyVault(ctrl)
			kvClient.EXPECT().GetSecret(ctx, "secret1", "").Return(
				tc.initKeyVaultSecret, nil,
			)

			if _, err := p.getSecret(ctx, kvClient, types.KeyVaultObject{ObjectName: "secret1"}); err == nil {
				t.Fatalf("getSecret() = nil, want error")
			}
		})
	}
}

func TestGetCertificate(t *testing.T) {
	id := azcertificates.ID("https://test.vault.azure.net/certificates/cert1/v1")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := testContext(t)

	p := &provider{}
	kvClient := mock_keyvault.NewMockKeyVault(ctrl)
	kvClient.EXPECT().GetCertificate(ctx, "cert1", "").Return(
		&azcertificates.CertificateBundle{
			CER: []byte("test"),
			ID:  &id,
		}, nil,
	)

	objs, err := p.getCertificate(ctx, kvClient, types.KeyVaultObject{ObjectName: "cert1"})
	if err != nil {
		t.Fatalf("getCertificate() = %v, want nil", err)
	}

	expected := []keyvaultObject{
		{
			content: `-----BEGIN CERTIFICATE-----
dGVzdA==
-----END CERTIFICATE-----
`,
			version: "v1",
		},
	}

	if !reflect.DeepEqual(objs, expected) {
		t.Fatalf("getCertificate() = \n%v, want \n%v", objs, expected)
	}
}

func TestGetCertificateError(t *testing.T) {
	id := azcertificates.ID("https://test.vault.azure.net/certificates/cert1/v1")

	cases := []struct {
		desc                   string
		initKeyVaultCert       *azcertificates.CertificateBundle
		kvError                error
		expectedKeyVaultObject []keyvaultObject
	}{
		{
			desc:             "keyvault get certificate error",
			initKeyVaultCert: &azcertificates.CertificateBundle{},
			kvError:          errors.New("keyvault error"),
		},
		{
			desc: "certificate CER is nil",
			initKeyVaultCert: &azcertificates.CertificateBundle{
				ID: &id,
			},
		},
		{
			desc: "certificate id is nil",
			initKeyVaultCert: &azcertificates.CertificateBundle{
				CER: []byte("test"),
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := testContext(t)

			p := &provider{}
			kvClient := mock_keyvault.NewMockKeyVault(ctrl)
			kvClient.EXPECT().GetCertificate(ctx, "cert1", "").Return(
				tc.initKeyVaultCert, nil,
			)

			if _, err := p.getCertificate(ctx, kvClient, types.KeyVaultObject{ObjectName: "cert1"}); err == nil {
				t.Fatalf("getCertificate() = nil, want error")
			}
		})
	}
}

func testContext(t *testing.T) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return ctx
}
