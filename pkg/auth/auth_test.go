package auth

import (
	"reflect"
	"testing"
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
