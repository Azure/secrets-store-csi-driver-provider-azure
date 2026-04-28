package auth

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// mockTransporter is a simple mock that satisfies policy.Transporter for tests.
type mockTransporter struct{}

func (m *mockTransporter) Do(_ *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK}, nil
}

func TestNewConfig(t *testing.T) {
	cases := []struct {
		desc                     string
		mode                     IdentityMode
		userAssignedIdentityID   string
		workloadIdentityClientID string
		serviceAccountToken      string
		secrets                  map[string]string
		expectedConfig           Config
		expectedErr              bool
	}{
		{
			desc:           "secrets nil for service principal mode",
			mode:           IdentityModeNone,
			expectedErr:    true,
			expectedConfig: Config{},
		},
		{
			desc:        "returns the correct auth config",
			mode:        IdentityModeNone,
			expectedErr: false,
			secrets:     map[string]string{"clientid": "testclientid", "clientsecret": "testclientsecret"},
			expectedConfig: Config{
				IdentityMode:           IdentityModeNone,
				UserAssignedIdentityID: "",
				AADClientID:            "testclientid",
				AADClientSecret:        "testclientsecret",
			},
		},
		{
			desc:                     "returns the correct auth config with workload identity",
			mode:                     IdentityModeNone,
			workloadIdentityClientID: "testworkloadclientid",
			serviceAccountToken:      "testworkloadtoken",
			expectedErr:              false,
			secrets:                  map[string]string{},
			expectedConfig: Config{
				IdentityMode:             IdentityModeNone,
				UserAssignedIdentityID:   "",
				AADClientID:              "",
				AADClientSecret:          "",
				WorkloadIdentityClientID: "testworkloadclientid",
				ServiceAccountToken:      "testworkloadtoken",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			config, err := NewConfig(
				tc.mode,
				tc.userAssignedIdentityID,
				tc.workloadIdentityClientID,
				tc.serviceAccountToken,
				tc.secrets,
			)
			if tc.expectedErr && err == nil || !tc.expectedErr && err != nil {
				t.Fatalf("expected error: %v, got error: %v", tc.expectedErr, err)
			}
			if !reflect.DeepEqual(config, tc.expectedConfig) {
				t.Fatalf("expected config: %+v, got: %+v", tc.expectedConfig, config)
			}
		})
	}
}

func TestGetCredential(t *testing.T) {
	cases := []struct {
		desc                 string
		secrets              map[string]string
		expectedClientID     string
		expectedClientSecret string
		expectedErr          bool
	}{
		{
			desc:        "client secret missing for service principal mode",
			secrets:     map[string]string{"clientid": "testclientid"},
			expectedErr: true,
		},
		{
			desc:        "client ID missing for service principal mode",
			secrets:     map[string]string{"clientsecret": "testclientsecret"},
			expectedErr: true,
		},
		{
			desc:                 "returns correct client id and client secret",
			secrets:              map[string]string{"clientid": "testclientid", "clientsecret": "testclientsecret"},
			expectedClientID:     "testclientid",
			expectedClientSecret: "testclientsecret",
			expectedErr:          false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			clientID, clientSecret, err := getCredential(tc.secrets)
			if tc.expectedErr && err == nil || !tc.expectedErr && err != nil {
				t.Fatalf("expected error: %v, got error: %v", tc.expectedErr, err)
			}
			if clientID != tc.expectedClientID {
				t.Fatalf("expected clientID: %v, got: %v", tc.expectedClientID, clientID)
			}
			if clientSecret != tc.expectedClientSecret {
				t.Fatalf("expected client secret: %v, got: %v", tc.expectedClientSecret, clientSecret)
			}
		})
	}
}

func TestParseServiceAccountTokenError(t *testing.T) {
	cases := []struct {
		desc     string
		saTokens string
	}{
		{
			desc:     "empty serviceaccount tokens",
			saTokens: "",
		},
		{
			desc:     "invalid serviceaccount tokens",
			saTokens: "invalid",
		},
		{
			desc:     "token for audience not found",
			saTokens: `{"aud1":{"token":"eyJhbGciOiJSUzI1NiIsImtpZCI6InRhVDBxbzhQVEZ1ajB1S3BYUUxIclRsR01XakxjemJNOTlzWVMxSlNwbWcifQ.eyJhdWQiOlsiYXBpOi8vQXp1cmVBRGlUb2tlbkV4Y2hhbmdlIl0sImV4cCI6MTY0MzIzNDY0NywiaWF0IjoxNjQzMjMxMDQ3LCJpc3MiOiJodHRwczovL2t1YmVybmV0ZXMuZGVmYXVsdC5zdmMuY2x1c3Rlci5sb2NhbCIsImt1YmVybmV0ZXMuaW8iOnsibmFtZXNwYWNlIjoidGVzdC12MWFscGhhMSIsInBvZCI6eyJuYW1lIjoic2VjcmV0cy1zdG9yZS1pbmxpbmUtY3JkIiwidWlkIjoiYjBlYmZjMzUtZjEyNC00ZTEyLWI3N2UtYjM0MjM2N2IyMDNmIn0sInNlcnZpY2VhY2NvdW50Ijp7Im5hbWUiOiJkZWZhdWx0IiwidWlkIjoiMjViNGY1NzgtM2U4MC00NTczLWJlOGQtZTdmNDA5ZDI0MmI2In19LCJuYmYiOjE2NDMyMzEwNDcsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDp0ZXN0LXYxYWxwaGExOmRlZmF1bHQifQ.ALE46aKmtTV7dsuFOwDZqvEjdHFUTNP-JVjMxexTemmPA78fmPTUZF0P6zANumA03fjX3L-MZNR3PxmEZgKA9qEGIDsljLsUWsVBEquowuBh8yoBYkGkMJmRfmbfS3y7_4Q7AU3D9Drw4iAHcn1GwedjOQC0i589y3dkNNqf8saqHfXkbSSLtSE0f2uzI-PjuTKvR1kuojEVNKlEcA4wsKfoiRpkua17sHkHU0q9zxCMDCr_1f8xbigRnRx0wscU3vy-8KhF3zQtpcWkk3r4C5YSXut9F3xjz5J9DUQn2vNMfZg4tOdcR-9Xv9fbY5iujiSlS58GEktSEa3SE9wrCw\",\"expirationTimestamp\":\"2022-01-26T22:04:07Z\"},\"gcp\":{\"token\":\"eyJhbGciOiJSUzI1NiIsImtpZCI6InRhVDBxbzhQVEZ1ajB1S3BYUUxIclRsR01XakxjemJNOTlzWVMxSlNwbWcifQ.eyJhdWQiOlsiZ2NwIl0sImV4cCI6MTY0MzIzNDY0NywiaWF0IjoxNjQzMjMxMDQ3LCJpc3MiOiJodHRwczovL2t1YmVybmV0ZXMuZGVmYXVsdC5zdmMuY2x1c3Rlci5sb2NhbCIsImt1YmVybmV0ZXMuaW8iOnsibmFtZXNwYWNlIjoidGVzdC12MWFscGhhMSIsInBvZCI6eyJuYW1lIjoic2VjcmV0cy1zdG9yZS1pbmxpbmUtY3JkIiwidWlkIjoiYjBlYmZjMzUtZjEyNC00ZTEyLWI3N2UtYjM0MjM2N2IyMDNmIn0sInNlcnZpY2VhY2NvdW50Ijp7Im5hbWUiOiJkZWZhdWx0IiwidWlkIjoiMjViNGY1NzgtM2U4MC00NTczLWJlOGQtZTdmNDA5ZDI0MmI2In19LCJuYmYiOjE2NDMyMzEwNDcsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDp0ZXN0LXYxYWxwaGExOmRlZmF1bHQifQ.BT0YGI7bGdSNaIBqIEnVL0Ky5t-fynaemSGxjGdKOPl0E22UIVGDpAMUhaS19i20c-Dqs-Kn0N-R5QyDNpZg8vOL5KIFqu2kSYNbKxtQW7TPYIsV0d9wUZjLSr54DKrmyXNMGRoT2bwcF4yyfmO46eMmZSaXN8Y4lgapeabg6CBVVQYHD-GrgXf9jVLeJfCQkTuojK1iXOphyD6NqlGtVCaY1jWxbBMibN0q214vKvQboub8YMuvclGdzn_l_ZQSTjvhBj9I-W1t-JArVjqHoIb8_FlR9BSgzgL7V3Jki55vmiOdEYqMErJWrIZPP3s8qkU5hhO9rSVEd3LJHponvQ","expirationTimestamp":"2022-01-26T22:04:07Z"}}`, //nolint
		},
		{
			desc:     "token value has wrong JSON type",
			saTokens: `{"api://AzureADTokenExchange":{"token":12345,"expirationTimestamp":"2022-01-26T22:04:07Z"}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if _, err := ParseServiceAccountToken(tc.saTokens); err == nil {
				t.Errorf("ParseServiceAccountToken(%s) = nil, want error", tc.saTokens)
			}
		})
	}
}

func TestParseServiceAccountToken(t *testing.T) {
	saTokens := `{"api://AzureADTokenExchange":{"token":"eyJhbGciOiJSUzI1NiIsImtpZCI6InRhVDBxbzhQVEZ1ajB1S3BYUUxIclRsR01XakxjemJNOTlzWVMxSlNwbWcifQ.eyJhdWQiOlsiYXBpOi8vQXp1cmVBRGlUb2tlbkV4Y2hhbmdlIl0sImV4cCI6MTY0MzIzNDY0NywiaWF0IjoxNjQzMjMxMDQ3LCJpc3MiOiJodHRwczovL2t1YmVybmV0ZXMuZGVmYXVsdC5zdmMuY2x1c3Rlci5sb2NhbCIsImt1YmVybmV0ZXMuaW8iOnsibmFtZXNwYWNlIjoidGVzdC12MWFscGhhMSIsInBvZCI6eyJuYW1lIjoic2VjcmV0cy1zdG9yZS1pbmxpbmUtY3JkIiwidWlkIjoiYjBlYmZjMzUtZjEyNC00ZTEyLWI3N2UtYjM0MjM2N2IyMDNmIn0sInNlcnZpY2VhY2NvdW50Ijp7Im5hbWUiOiJkZWZhdWx0IiwidWlkIjoiMjViNGY1NzgtM2U4MC00NTczLWJlOGQtZTdmNDA5ZDI0MmI2In19LCJuYmYiOjE2NDMyMzEwNDcsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDp0ZXN0LXYxYWxwaGExOmRlZmF1bHQifQ.ALE46aKmtTV7dsuFOwDZqvEjdHFUTNP-JVjMxexTemmPA78fmPTUZF0P6zANumA03fjX3L-MZNR3PxmEZgKA9qEGIDsljLsUWsVBEquowuBh8yoBYkGkMJmRfmbfS3y7_4Q7AU3D9Drw4iAHcn1GwedjOQC0i589y3dkNNqf8saqHfXkbSSLtSE0f2uzI-PjuTKvR1kuojEVNKlEcA4wsKfoiRpkua17sHkHU0q9zxCMDCr_1f8xbigRnRx0wscU3vy-8KhF3zQtpcWkk3r4C5YSXut9F3xjz5J9DUQn2vNMfZg4tOdcR-9Xv9fbY5iujiSlS58GEktSEa3SE9wrCw","expirationTimestamp":"2022-01-26T22:04:07Z"},"aud2":{"token":"eyJhbGciOiJSUzI1NiIsImtpZCI6InRhVDBxbzhQVEZ1ajB1S3BYUUxIclRsR01XakxjemJNOTlzWVMxSlNwbWcifQ.eyJhdWQiOlsiZ2NwIl0sImV4cCI6MTY0MzIzNDY0NywiaWF0IjoxNjQzMjMxMDQ3LCJpc3MiOiJodHRwczovL2t1YmVybmV0ZXMuZGVmYXVsdC5zdmMuY2x1c3Rlci5sb2NhbCIsImt1YmVybmV0ZXMuaW8iOnsibmFtZXNwYWNlIjoidGVzdC12MWFscGhhMSIsInBvZCI6eyJuYW1lIjoic2VjcmV0cy1zdG9yZS1pbmxpbmUtY3JkIiwidWlkIjoiYjBlYmZjMzUtZjEyNC00ZTEyLWI3N2UtYjM0MjM2N2IyMDNmIn0sInNlcnZpY2VhY2NvdW50Ijp7Im5hbWUiOiJkZWZhdWx0IiwidWlkIjoiMjViNGY1NzgtM2U4MC00NTczLWJlOGQtZTdmNDA5ZDI0MmI2In19LCJuYmYiOjE2NDMyMzEwNDcsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDp0ZXN0LXYxYWxwaGExOmRlZmF1bHQifQ.BT0YGI7bGdSNaIBqIEnVL0Ky5t-fynaemSGxjGdKOPl0E22UIVGDpAMUhaS19i20c-Dqs-Kn0N-R5QyDNpZg8vOL5KIFqu2kSYNbKxtQW7TPYIsV0d9wUZjLSr54DKrmyXNMGRoT2bwcF4yyfmO46eMmZSaXN8Y4lgapeabg6CBVVQYHD-GrgXf9jVLeJfCQkTuojK1iXOphyD6NqlGtVCaY1jWxbBMibN0q214vKvQboub8YMuvclGdzn_l_ZQSTjvhBj9I-W1t-JArVjqHoIb8_FlR9BSgzgL7V3Jki55vmiOdEYqMErJWrIZPP3s8qkU5hhO9rSVEd3LJHponvQ","expirationTimestamp":"2022-01-26T22:04:07Z"}}` //nolint
	expectedToken := `eyJhbGciOiJSUzI1NiIsImtpZCI6InRhVDBxbzhQVEZ1ajB1S3BYUUxIclRsR01XakxjemJNOTlzWVMxSlNwbWcifQ.eyJhdWQiOlsiYXBpOi8vQXp1cmVBRGlUb2tlbkV4Y2hhbmdlIl0sImV4cCI6MTY0MzIzNDY0NywiaWF0IjoxNjQzMjMxMDQ3LCJpc3MiOiJodHRwczovL2t1YmVybmV0ZXMuZGVmYXVsdC5zdmMuY2x1c3Rlci5sb2NhbCIsImt1YmVybmV0ZXMuaW8iOnsibmFtZXNwYWNlIjoidGVzdC12MWFscGhhMSIsInBvZCI6eyJuYW1lIjoic2VjcmV0cy1zdG9yZS1pbmxpbmUtY3JkIiwidWlkIjoiYjBlYmZjMzUtZjEyNC00ZTEyLWI3N2UtYjM0MjM2N2IyMDNmIn0sInNlcnZpY2VhY2NvdW50Ijp7Im5hbWUiOiJkZWZhdWx0IiwidWlkIjoiMjViNGY1NzgtM2U4MC00NTczLWJlOGQtZTdmNDA5ZDI0MmI2In19LCJuYmYiOjE2NDMyMzEwNDcsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDp0ZXN0LXYxYWxwaGExOmRlZmF1bHQifQ.ALE46aKmtTV7dsuFOwDZqvEjdHFUTNP-JVjMxexTemmPA78fmPTUZF0P6zANumA03fjX3L-MZNR3PxmEZgKA9qEGIDsljLsUWsVBEquowuBh8yoBYkGkMJmRfmbfS3y7_4Q7AU3D9Drw4iAHcn1GwedjOQC0i589y3dkNNqf8saqHfXkbSSLtSE0f2uzI-PjuTKvR1kuojEVNKlEcA4wsKfoiRpkua17sHkHU0q9zxCMDCr_1f8xbigRnRx0wscU3vy-8KhF3zQtpcWkk3r4C5YSXut9F3xjz5J9DUQn2vNMfZg4tOdcR-9Xv9fbY5iujiSlS58GEktSEa3SE9wrCw`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         //nolint

	token, err := ParseServiceAccountToken(saTokens)
	if err != nil {
		t.Fatalf("ParseServiceAccountToken(%s) = %v, want nil", saTokens, err)
	}
	if token != expectedToken {
		t.Errorf("ParseServiceAccountToken(%s) = %s, want %s", saTokens, token, expectedToken)
	}
}

func TestGetScope(t *testing.T) {
	tests := []struct {
		name     string
		scope    string
		expected string
	}{
		{
			name:     "resource doesn't have /.default suffix",
			scope:    "https://vault.azure.net",
			expected: "https://vault.azure.net/.default",
		},
		{
			name:     "resource has /.default suffix",
			scope:    "https://vault.azure.net/.default",
			expected: "https://vault.azure.net/.default",
		},
		{
			name:     "resource doesn't  have /.default suffix and has trailing slash",
			scope:    "https://vault.azure.net/",
			expected: "https://vault.azure.net//.default",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scope := getScope(test.scope)
			if scope != test.expected {
				t.Errorf("expected scope %s, got %s", test.expected, scope)
			}
		})
	}
}

func TestParseIdentityBindingTokenError(t *testing.T) {
	cases := []struct {
		desc     string
		saTokens string
	}{
		{
			desc:     "empty serviceaccount tokens",
			saTokens: "",
		},
		{
			desc:     "invalid serviceaccount tokens",
			saTokens: "invalid",
		},
		{
			desc:     "identity binding audience not found",
			saTokens: `{"api://AzureADTokenExchange":{"token":"some-token","expirationTimestamp":"2099-01-01T00:00:00Z"}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if _, err := ParseIdentityBindingToken(tc.saTokens); err == nil {
				t.Error("ParseIdentityBindingToken() = nil, want error")
			}
		})
	}
}

func TestParseIdentityBindingToken(t *testing.T) {
	saTokens := `{"api://AKSIdentityBinding":{"token":"identity-binding-token","expirationTimestamp":"2099-01-01T00:00:00Z"},"api://AzureADTokenExchange":{"token":"wi-token","expirationTimestamp":"2099-01-01T00:00:00Z"}}` // nolint:gosec // test data, not credentials
	expectedToken := "identity-binding-token"

	token, err := ParseIdentityBindingToken(saTokens)
	if err != nil {
		t.Fatalf("ParseIdentityBindingToken() = %v, want nil", err)
	}
	if token != expectedToken {
		t.Errorf("ParseIdentityBindingToken() = %s, want %s", token, expectedToken)
	}
}

func TestNewConfig_IdentityBinding(t *testing.T) {
	clientID := "test-client-id"
	token := "test-token"

	config, err := NewConfig(
		IdentityModeAzureTokenProxy,
		"", // userAssignedIdentityID
		clientID,
		token,
		nil, // secrets
	)

	if err != nil {
		t.Fatalf("NewConfig() unexpected error: %v", err)
	}
	if config.IdentityMode != IdentityModeAzureTokenProxy {
		t.Errorf("IdentityMode = %d, want %d", config.IdentityMode, IdentityModeAzureTokenProxy)
	}
	if config.WorkloadIdentityClientID != clientID {
		t.Errorf("WorkloadIdentityClientID = %s, want %s", config.WorkloadIdentityClientID, clientID)
	}
	if config.ServiceAccountToken != token {
		t.Errorf("ServiceAccountToken = %s, want %s", config.ServiceAccountToken, token)
	}
}

func TestGetCredential_IdentityBinding(t *testing.T) {
	// Save and restore the package-level proxy transport state
	savedTransport := proxyTransport
	savedErr := proxyTransportErr
	defer func() {
		proxyTransport = savedTransport
		proxyTransportErr = savedErr
	}()

	// Set up a mock proxy transport (no network required)
	SetProxyTransport(&mockTransporter{}, nil)

	config := Config{
		IdentityMode:             IdentityModeAzureTokenProxy,
		WorkloadIdentityClientID: "test-client-id",
		ServiceAccountToken:      "test-token",
	}

	cred, err := config.GetCredential(
		"test-pod",
		"default",
		"https://vault.azure.net",
		"https://login.microsoftonline.com/",
		"test-tenant-id",
		"2579",
	)

	if err != nil {
		t.Fatalf("GetCredential() unexpected error: %v", err)
	}
	if cred == nil {
		t.Fatal("GetCredential() returned nil credential")
	}
}

func TestGetCredential_IdentityBinding_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		clientID string
		token    string
	}{
		{"missing client ID", "", "token"},
		{"missing token", "client-id", ""},
		{"both missing", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				IdentityMode:             IdentityModeAzureTokenProxy,
				WorkloadIdentityClientID: tt.clientID,
				ServiceAccountToken:      tt.token,
			}

			_, err := config.GetCredential(
				"test-pod", "default", "https://vault.azure.net",
				"https://login.microsoftonline.com/", "test-tenant-id", "2579",
			)

			if err == nil {
				t.Fatal("expected error but got nil")
			}
			if !strings.Contains(err.Error(), "required") {
				t.Errorf("expected error containing 'required', got: %v", err)
			}
		})
	}
}

func TestGetManagedIdentityTokenCredential(t *testing.T) {
	tests := []struct {
		name             string
		identityClientID string
	}{
		{
			name:             "system-assigned identity with empty client ID",
			identityClientID: "",
		},
		{
			name:             "user-assigned identity with client ID",
			identityClientID: "user-assigned-client-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred, err := getManagedIdentityTokenCredential(tt.identityClientID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cred == nil {
				t.Fatal("expected non-nil credential")
			}
		})
	}
}
