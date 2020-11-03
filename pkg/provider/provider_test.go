package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/azure"
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
		p, err := NewProvider()
		if err != nil {
			t.Fatalf("expected nil err, got: %v", err)
		}
		p.KeyvaultName = tc.vaultName

		for idx := range testEnvs {
			azCloudEnv, err := ParseAzureEnvironment(testEnvs[idx])
			if err != nil {
				t.Fatalf("Error parsing cloud environment %v", err)
			}
			p.AzureCloudEnvironment = azCloudEnv
			vaultURL, err := p.getVaultURL(context.Background())
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
		azureEnv, err := ParseAzureEnvironment(envName)
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
	_, err := ParseAzureEnvironment(wrongEnvName)
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
			value: `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`,
			expectedCert: `-----BEGIN CERTIFICATE-----
MIIEBDCCAewCCQDT03s5KAJfljANBgkqhkiG9w0BAQsFADBEMQswCQYDVQQGEwJV
UzELMAkGA1UECAwCV0ExEDAOBgNVBAcMB3JlZG1vbmQxFjAUBgNVBAMMDWRlbW8u
dGVzdC5jb20wHhcNMjAwNjE1MDIzNzUzWhcNMjExMDI4MDIzNzUzWjBEMQswCQYD
VQQGEwJVUzELMAkGA1UECAwCV0ExEDAOBgNVBAcMB3JlZG1vbmQxFjAUBgNVBAMM
DWRlbW8udGVzdC5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCe
Lkx+0iAqaiF/Ts9H+RJh20+vjfZ5f2CkeKCWlaQjKgIMa2y35MvqKRimqV70UjY2
cTiuUpKl8fWommBGpSoSsQzC647qZFCc1KuzLII86OvuwYAPMU+vpIQfJ5/b5m0d
GMclMF12KH4RaONdE7v8waHsoAWWnRY9ogmM9J5DW9Y0fevDCcnXC57PQdV+xcDc
JBGQS4DHfekPxkiKXSiUlLtodV6hRbLFnQb+4Nmdbqaqxw4hSx+3bTduduruhPSY
uONCOAImuiFHcB65L55sC+irKYImK/grEItgOl0PNU0fsJCW68g6fWw+OX2gdVpv
9gazLc3dFGgQWbMRBoPJAgMBAAEwDQYJKoZIhvcNAQELBQADggIBAIoMamQ72DnQ
qWOfVfQsx4HUo0GUeiXUuVOBozfVFlwuNp0c5pxhf3gpjs3Jd3bgpI5dKMGt4lYh
gQvHtd1nbc81LCfrQxWZOs3PJ7ybDwfrDep/Y22s0MJHWAOsqLTPRwfwcpG9gj5m
CsxA1lYbvc7mmkbLhMwvRrGMbiFf53tZJOP1Et5vYmdePDcW7P2D9uNH5d4bQF/S
HNDIcNuFSBG5ZfMOkq2VrgYGLVtLXumPE3ZAML658tTqbcdhI7iE2VJSgoPFWuoL
23uCSX85eV4jp2A7H679BPkJKHZGwkWunZacJueXND/F5c9I3FsAc21Ek5eXG7Ii
lJue9XXJcY07Z6wYjYlngAxaWzNEdo7udj+HOb3unIa+FzTP3r+h33ldeUF+FARn
0B1s/jkYTUmEa6q6A0IRWyN3lwm91h5J9QGK/xcXKyiOr67oXFgzsq942umtf2Es
oRgZgcpxBnJvv2Zb3tCIBhT3hVAcviYhNGsGf1jyOrYTXYfGpvPn6kEsNcNTjtA5
Z/DN0KANwcAyY9m9CXt3ssMBV//NX9slxxT2foaDW7WKW+Wq1kRLUTqGsMzmMkxe
dcIVkTwoU5+ZbLEXBIqKhGiWGXGnpgQUf1vWeEx+/eWS5iIQ3hQfWHeCJNyN0TSi
HeQ9KRXRZgdNTDGc5KniC8jxlJPgwSMZ
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIFBDCCAuwCCQC1m3QEbgnlPDANBgkqhkiG9w0BAQsFADBEMQswCQYDVQQGEwJV
UzELMAkGA1UECAwCV0ExEDAOBgNVBAcMB3JlZG1vbmQxFjAUBgNVBAMMDWRlbW8u
dGVzdC5jb20wHhcNMjAwNjE1MDIzNjQ3WhcNMjMwNDA1MDIzNjQ3WjBEMQswCQYD
VQQGEwJVUzELMAkGA1UECAwCV0ExEDAOBgNVBAcMB3JlZG1vbmQxFjAUBgNVBAMM
DWRlbW8udGVzdC5jb20wggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQDo
hBzgRrwLT/B1XIwJWxj36hmQ+wRgEc+Q1JWjJiaVYeBc1utQevTCEAZbn9pA9D4f
sMTvoooioF2TILfLaCE8LKszaSC/6TgnndFvnvx1BkdUNQdYPsaUhuqaor1HSgRx
ZECl9017ftIppMJaffjF3hTf1V8nlcjU6XjlynGEYub/CZOhJCyo6d8IU+gQs4AB
9yfmJZzhiUPxuVQ1ukXxLh/BMtWs4+KJZvHDJmdgDOovjnQp+9jB6ykPjKrvo+Mg
gczyt2l+d1RQyatCfeqHvqU8KADoUWdedpHTPV7L4MevIuU45C1mhsHOWCykPWQL
bi0gZ1wHlPmOnRinW2TY8gY+LBvILxXLQUu4cMzv8o8uWdtwSpISsXD6q2E5nxZq
f0PkJlWsqNj0I7Qv8S3QzXVjyjuiueKfkX7leJayYdsy9EVnNBZKWGTg254F9QG9
jeezN+BRcEb3thoa41rqYbc+IY19HOc4tq9OX+zvm11NVZyTujLpv+x1SFyVKM8Z
ivd7cGjC80MKk/95t20E+mZtLGpXoYb4ndwJYNsNARi7TBIe03kP8lUlyxTMZijb
mfXYyyX6eReaS/S26iEaLi5L82XqF7kYl8ss9G/zBKmPnGaQCF11pYM0q09x2o2p
iF8RvltUO2b0/SBhHpdD6/kzLiGnDtU26FjI8BhhmQIDAQABMA0GCSqGSIb3DQEB
CwUAA4ICAQDC1dMw07Ooec28Td1H+oVLel/P5uvtctRz54JL4HHMxrlSxLqIvrye
cnediGSCxFSrSvg9sUfq6UILcF8KUYmmTjsXQC6QMVa7Kt5BggLI+BS5Vlj6BEQp
2V5EtXXnU8YIHB7gt5iZU07Yk3aBQioGgSUKOEMjlarnHV50XB6NkS9x3MO+pw/Y
jE1DxmRNKHXOyOgv4PQbqPyXFqryL2kq6QpDqqj6ucH/49IjqNVf7dVJPl3sC13v
wVDGVlznJWRbU1WyQpkFGFaPVwqf7e7OWk0pxW+0RptzcAjZQa5qGlxXPF4H0v16
sSDcI7qPcRj8ztL/Uw2r9q/n5DY+ag5IEB4Acw5nJl6LJWcFdYb3YAfZWPwVzOqy
iUU/Lli9D/Jsmb7erXT3AFD0L4JgRnrLQiPqjNfIsAWsVHEuCzPwQEd7Tvi127Di
wSIW87flu9GeZbheLPeXzQmGQjgGT8rvX7nXtSEsSKhaefHwDPfxFH33pRFc7YUm
GsmWcS0+ewWlzF1d/m3ist0C4w8GyQ5jYuamyErVNa/pQ1iHsi7A9wfq2XrUabi6
a5H8yLyBQ0iqu9hEcBHTmlD6rupWVU3MRee7H+956J5rbzJkngils4TP78jbgQIO
fij5z2FZL+lrHgkj7cE29O8plJNbHTi+AsguKCy4OSBSha53tFUB6A==
-----END CERTIFICATE-----
`,
			expectedKey: `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCeLkx+0iAqaiF/
Ts9H+RJh20+vjfZ5f2CkeKCWlaQjKgIMa2y35MvqKRimqV70UjY2cTiuUpKl8fWo
mmBGpSoSsQzC647qZFCc1KuzLII86OvuwYAPMU+vpIQfJ5/b5m0dGMclMF12KH4R
aONdE7v8waHsoAWWnRY9ogmM9J5DW9Y0fevDCcnXC57PQdV+xcDcJBGQS4DHfekP
xkiKXSiUlLtodV6hRbLFnQb+4Nmdbqaqxw4hSx+3bTduduruhPSYuONCOAImuiFH
cB65L55sC+irKYImK/grEItgOl0PNU0fsJCW68g6fWw+OX2gdVpv9gazLc3dFGgQ
WbMRBoPJAgMBAAECggEAdICrLJj8weCLLWXROzRSeqp5pVZM262ac2i49k/epVuS
SK1yVHr6SUvdcRq76f0HGtAHLDV69ygfg/+3uzB4rW2jdYjjKPWbffcRQxUcE1qG
MsJn3Ei4ZSgpe3zGu8KaSNzwCA74N0eZmp8DHpGjWoUPCuXNV+H4+In4uM2OJxM7
VcqlTXPdp94XCOYM9K138fSoegwpu+ef37fm+tSR5rIyh+pqi5SCiSKn5R4UVRCu
dXEYclxL603tt1eYl/C8MqmtrVCcReSlpfLUX9xuT145egz2rPfMV6Bl1+6GxfJn
4PZOnbQZXXgo8UDgwlqhZHKll2t26awXwyTja0VDkQKBgQDNFMKazHlOSexqnuBL
HOVE/sdA3GMtlGlwGjhYxSRqoz8KBag7RVDWkrfgXbHX+G4aJc3PErz+aKxSV0XA
DG9WjJDJOojsFT9/XWpCu1aA26lT1QJL6yDcLdokjQC586JwhkV7JR2YNkRduF9H
NnqcA/iD9y2oocn6UnRyo+VQFQKBgQDFdH5+c6Iceup7LuVDBhBXH9RbOVqEqSA3
QvTe5U46cul7bSjYgQn+ksnn9UML39sytnVrnC3LOLZmIEJzy1n22rY0b82KrnCY
XU9sjGeniIBG8j0fXKDvSGBsy5NL1u7zAU3v9sCnArbACFPVu7S/hO7V24ZiJJHt
/hDVcOWd5QKBgCDkNn37U21R/9/t0U1aug7ByhVGA4YY6nw3SFg8biXIPuENnTi8
WkW/zEvo2xAnYQlCjOqsN7GZ+iFOq/osRGMeMk6D29f5ZHC5+8PuJeaO1G6EmFSy
xldp5zW7g6VPRPtFHbmtbzytX3OkkWtremixXldT+ne0Ux+Zv+FvFeUtAoGAFPg6
NtOw87VaEZr5XhTWx2np84YzxsLvWO8TcliH5k0t3p3JKLULiq2sI6Y4aJptfQVD
kxoTAvIS7OWgKQv/kefIUelNutyruIKwXKbMm04z0VUIiLwdm0vkcaltCzDYT5Zj
4IgkDZiML/iybpBwsaY8dxnJO8MGfG/u+bvzpsECgYEAirskIr2+Gn2IwFnLf14g
I6+joeBo7bu/31yhLm4XcDPD2rl3PMIPCvxRe5FfPQOIAF9Vxibsomdhi30Z2Vmo
6GxVnQB9Zvy+V0LkfHvF7qUyxcY76o7SzwrZ0e4or3snPLMyOV8ZIH0wfolqHm9y
Uz7sJMWoq7mOrINHQ0ZmaiE=
-----END PRIVATE KEY-----
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			content, err := decodePKCS12(tc.value)
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
	file, err := ioutil.TempFile("", "ut")
	defer os.Remove(file.Name())
	if err != nil {
		t.Fatalf("expected error to be nil, got: %+v", err)
	}
	_, err = io.WriteString(file, fmt.Sprintf(`{"name": "%s"}`, azureStackCloudEnvName))
	if err != nil {
		t.Fatalf("expected error to be nil, got: %+v", err)
	}
	_, err = ParseAzureEnvironment(azureStackCloudEnvName)
	if err == nil {
		t.Fatalf("expected error to be not nil as AZURE_ENVIRONMENT_FILEPATH is not set")
	}

	err = setAzureEnvironmentFilePath(file.Name())
	defer os.Unsetenv(azure.EnvironmentFilepathName)
	if err != nil {
		t.Fatalf("expected error to be nil, got: %+v", err)
	}
	env, err := ParseAzureEnvironment(azureStackCloudEnvName)
	if err != nil {
		t.Fatalf("expected error to be nil, got: %+v", err)
	}
	if env.Name != azureStackCloudEnvName {
		t.Fatalf("expected environment name to be '%s', got: '%s'", azureStackCloudEnvName, env.Name)
	}
}

func TestValidateObjectFormat(t *testing.T) {
	cases := []struct {
		desc         string
		objectFormat string
		objectType   string
		expectedErr  error
	}{
		{
			desc:         "no object format specified",
			objectFormat: "",
			objectType:   "cert",
			expectedErr:  nil,
		},
		{
			desc:         "object format not valid",
			objectFormat: "pkcs",
			objectType:   "secret",
			expectedErr:  fmt.Errorf("invalid objectFormat: pkcs, should be PEM or PFX"),
		},
		{
			desc:         "object format PFX, but object type not secret",
			objectFormat: "pfx",
			objectType:   "cert",
			expectedErr:  fmt.Errorf("PFX format only supported for objectType: secret"),
		},
		{
			desc:         "object format PFX case insensitive check",
			objectFormat: "PFX",
			objectType:   "secret",
			expectedErr:  nil,
		},
		{
			desc:         "valid object format and type",
			objectFormat: "pfx",
			objectType:   "secret",
			expectedErr:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := validateObjectFormat(tc.objectFormat, tc.objectType)
			if tc.expectedErr != nil && err.Error() != tc.expectedErr.Error() || tc.expectedErr == nil && err != nil {
				t.Fatalf("expected err: %+v, got: %+v", tc.expectedErr, err)
			}
		})
	}
}

func TestValidateObjectEncoding(t *testing.T) {
	cases := []struct {
		desc           string
		objectEncoding string
		objectType     string
		expectedErr    error
	}{
		{
			desc:           "No encoding specified",
			objectEncoding: "",
			objectType:     "cert",
			expectedErr:    nil,
		},
		{
			desc:           "Invalid encoding specified",
			objectEncoding: "utf-16",
			objectType:     "secret",
			expectedErr:    fmt.Errorf("invalid objectEncoding: utf-16, should be hex, base64 or utf-8"),
		},
		{
			desc:           "Object Encoding Base64, but objectType is not secret",
			objectEncoding: "base64",
			objectType:     "cert",
			expectedErr:    fmt.Errorf("objectEncoding only supported for objectType: secret"),
		},
		{
			desc:           "Object Encoding case-insensitive check",
			objectEncoding: "BasE64",
			objectType:     "secret",
			expectedErr:    nil,
		},
		{
			desc:           "Valid ObjectEncoding and Type",
			objectEncoding: "base64",
			objectType:     "secret",
			expectedErr:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := validateObjectEncoding(tc.objectEncoding, tc.objectType)
			if tc.expectedErr != nil && err.Error() != tc.expectedErr.Error() || tc.expectedErr == nil && err != nil {
				t.Fatalf("expected err: %+v, got: %+v", tc.expectedErr, err)
			}
		})
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

func TestFormatKeyVaultObject(t *testing.T) {
	cases := []struct {
		desc                   string
		keyVaultObject         KeyVaultObject
		expectedKeyVaultObject KeyVaultObject
	}{
		{
			desc: "leading and trailing whitespace trimmed from all fields",
			keyVaultObject: KeyVaultObject{
				ObjectName:     "secret1     ",
				ObjectVersion:  "",
				ObjectEncoding: "base64   ",
				ObjectType:     "  secret",
				ObjectAlias:    "",
			},
			expectedKeyVaultObject: KeyVaultObject{
				ObjectName:     "secret1",
				ObjectVersion:  "",
				ObjectEncoding: "base64",
				ObjectType:     "secret",
				ObjectAlias:    "",
			},
		},
		{
			desc: "no data loss for already sanitized object",
			keyVaultObject: KeyVaultObject{
				ObjectName:     "secret1",
				ObjectVersion:  "version1",
				ObjectEncoding: "base64",
				ObjectType:     "secret",
				ObjectAlias:    "alias",
			},
			expectedKeyVaultObject: KeyVaultObject{
				ObjectName:     "secret1",
				ObjectVersion:  "version1",
				ObjectEncoding: "base64",
				ObjectType:     "secret",
				ObjectAlias:    "alias",
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

func TestValidateFilePath(t *testing.T) {
	cases := []struct {
		desc        string
		fileName    string
		expectedErr error
	}{
		{
			desc:        "file name is absolute path",
			fileName:    "/secret1",
			expectedErr: fmt.Errorf("file name must be a relative path"),
		},
		{
			desc:        "file name contains '..'",
			fileName:    "secret1/..",
			expectedErr: fmt.Errorf("file name must not contain '..'"),
		},
		{
			desc:        "file name starts with '..'",
			fileName:    "../secret1",
			expectedErr: fmt.Errorf("file name must not contain '..'"),
		},
		{
			desc:        "file name is empty",
			fileName:    "",
			expectedErr: fmt.Errorf("file name must not be empty"),
		},
		{
			desc:        "valid file name",
			fileName:    "secret1",
			expectedErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := validateFileName(tc.fileName)
			if tc.expectedErr != nil && err.Error() != tc.expectedErr.Error() || tc.expectedErr == nil && err != nil {
				t.Fatalf("expected err: %+v, got: %+v", tc.expectedErr, err)
			}
		})
	}
}
