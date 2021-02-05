package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	cases := []struct {
		desc                   string
		usePodIdentity         bool
		useVMManagedIdentity   bool
		userAssignedIdentityID string
		secrets                map[string]string
		expectedConfig         Config
		expectedErr            bool
	}{
		{
			desc:                 "pod identity and vm managed identity enabled",
			usePodIdentity:       true,
			useVMManagedIdentity: true,
			expectedErr:          true,
			expectedConfig:       Config{},
		},
		{
			desc:                 "secrets nil for service principal mode",
			usePodIdentity:       false,
			useVMManagedIdentity: false,
			expectedErr:          true,
			expectedConfig:       Config{},
		},
		{
			desc:                 "returns the correct auth config",
			usePodIdentity:       false,
			useVMManagedIdentity: false,
			expectedErr:          false,
			secrets:              map[string]string{"clientid": "testclientid", "clientsecret": "testclientsecret"},
			expectedConfig: Config{
				UsePodIdentity:         false,
				UseVMManagedIdentity:   false,
				UserAssignedIdentityID: "",
				AADClientID:            "testclientid",
				AADClientSecret:        "testclientsecret",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			config, err := NewConfig(tc.usePodIdentity, tc.useVMManagedIdentity, tc.userAssignedIdentityID, tc.secrets)
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

func TestGetServicePrincipalToken(t *testing.T) {
	config := Config{
		AADClientID:     "AADClientID",
		AADClientSecret: "AADClientSecret",
	}
	env := &azure.PublicCloud
	token, err := config.GetServicePrincipalToken("pod", "default", env.KeyVaultEndpoint, env.ActiveDirectoryEndpoint, "tenantID", "2579")
	assert.NoError(t, err)

	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, "tenantID")
	assert.NoError(t, err)

	spt, err := adal.NewServicePrincipalToken(*oauthConfig, config.AADClientID, config.AADClientSecret, env.KeyVaultEndpoint)
	assert.NoError(t, err)

	assert.Equal(t, token, spt)
}

func TestGetServicePrincipalTokenFromMSIWithUserAssignedID(t *testing.T) {
	configs := []Config{
		{
			UseVMManagedIdentity:   true,
			UserAssignedIdentityID: "UserAssignedIdentityID",
		},
		// uses managed identity when sp credentials are provided
		{
			UseVMManagedIdentity:   true,
			UserAssignedIdentityID: "UserAssignedIdentityID",
			AADClientID:            "AADClientID",
			AADClientSecret:        "AADClientSecret",
		},
	}
	env := &azure.PublicCloud

	for _, config := range configs {
		token, err := config.GetServicePrincipalToken("pod", "default", env.KeyVaultEndpoint, env.ActiveDirectoryEndpoint, "tenantID", "2579")
		assert.NoError(t, err)

		msiEndpoint, err := adal.GetMSIVMEndpoint()
		assert.NoError(t, err)

		spt, err := adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint,
			env.KeyVaultEndpoint, config.UserAssignedIdentityID)
		assert.NoError(t, err)
		assert.Equal(t, token, spt)
	}
}

func TestGetServicePrincipalTokenFromMSI(t *testing.T) {
	configs := []Config{
		{
			UseVMManagedIdentity: true,
		},
		// uses managed identity when sp credentials are provided
		{
			UseVMManagedIdentity: true,
			AADClientID:          "AADClientID",
			AADClientSecret:      "AADClientSecret",
		},
	}
	env := &azure.PublicCloud

	for _, config := range configs {
		token, err := config.GetServicePrincipalToken("pod", "default", env.KeyVaultEndpoint, env.ActiveDirectoryEndpoint, "tenantID", "2579")
		assert.NoError(t, err)

		msiEndpoint, err := adal.GetMSIVMEndpoint()
		assert.NoError(t, err)

		spt, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, env.KeyVaultEndpoint)
		assert.NoError(t, err)
		assert.Equal(t, token, spt)
	}
}

func TestGetServicePrincipalTokenPodIdentity(t *testing.T) {
	config := Config{
		UsePodIdentity: true,
	}
	env := &azure.PublicCloud

	cases := []struct {
		desc        string
		tokenResp   NMIResponse
		podName     string
		expectedErr error
	}{
		{
			desc:        "pod name is empty",
			tokenResp:   NMIResponse{},
			podName:     "",
			expectedErr: fmt.Errorf("pod information is not available. deploy a CSIDriver object to set podInfoOnMount: true"),
		},
		{
			desc:        "token response is empty",
			tokenResp:   NMIResponse{},
			podName:     "pod",
			expectedErr: fmt.Errorf("nmi did not return expected values in response: token and clientid"),
		},
		{
			desc: "valid token response",
			tokenResp: NMIResponse{
				Token: adal.Token{
					AccessToken: "accessToken",
					ExpiresIn:   "0",
					ExpiresOn:   "0",
					NotBefore:   "0",
				},
				ClientID: "clientID",
			},
			podName:     "pod",
			expectedErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			// mock NMI server
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.String(), "/host/token/")
				tr, err := json.Marshal(tc.tokenResp)
				assert.NoError(t, err)

				w.Write(tr)
			}))
			defer ts.Close()

			splitURL := strings.Split(ts.URL, ":")
			mockNMIPort := splitURL[len(splitURL)-1]

			token, err := config.GetServicePrincipalToken(tc.podName, "default", env.KeyVaultEndpoint, env.ActiveDirectoryEndpoint, "tenantID", mockNMIPort)
			assert.Equal(t, tc.expectedErr, err)

			if tc.expectedErr == nil {
				oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, "tenantID")
				assert.NoError(t, err)

				spt, err := adal.NewServicePrincipalTokenFromManualToken(*oauthConfig, "clientID", env.KeyVaultEndpoint, tc.tokenResp.Token, nil)
				assert.NoError(t, err)
				assert.Equal(t, token, spt)
			}
		})
	}
}
