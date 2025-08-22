package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/types"
)

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

func TestValidateNoInvisibleCharacters(t *testing.T) {
	cases := []struct {
		desc        string
		input       string
		fieldName   string
		expectedErr error
	}{
		{
			desc:        "empty string",
			input:       "",
			fieldName:   "testField",
			expectedErr: nil,
		},
		{
			desc:        "normal string",
			input:       "secret1",
			fieldName:   "objectName",
			expectedErr: nil,
		},
		{
			desc:        "string with zero width space (U+200B)",
			input:       "secret1\u200B",
			fieldName:   "objectName",
			expectedErr: fmt.Errorf("field objectName contains invisible character Zero Width Space (U+200B) at position 7"),
		},
		{
			desc:        "string with zero width non-joiner (U+200C)",
			input:       "secret\u200C1",
			fieldName:   "objectName",
			expectedErr: fmt.Errorf("field objectName contains invisible character Zero Width Non-Joiner (U+200C) at position 6"),
		},
		{
			desc:        "string with zero width joiner (U+200D)",
			input:       "\u200Dsecret1",
			fieldName:   "objectName",
			expectedErr: fmt.Errorf("field objectName contains invisible character Zero Width Joiner (U+200D) at position 0"),
		},
		{
			desc:        "string with BOM (U+FEFF)",
			input:       "\uFEFFsecret1",
			fieldName:   "objectName",
			expectedErr: fmt.Errorf("field objectName contains invisible character Zero Width No-Break Space/BOM (U+FEFF) at position 0"),
		},
		{
			desc:        "string with word joiner (U+2060)",
			input:       "sec\u2060ret1",
			fieldName:   "objectName",
			expectedErr: fmt.Errorf("field objectName contains invisible character Word Joiner (U+2060) at position 3"),
		},
		{
			desc:        "string with other format character",
			input:       "secret\u200E1", // Left-to-Right Mark
			fieldName:   "objectName",
			expectedErr: fmt.Errorf("field objectName contains invisible format character (U+200E) at position 6"),
		},
		{
			desc:        "string with allowed soft hyphen",
			input:       "secret\u00AD1", // Soft hyphen (should be allowed)
			fieldName:   "objectName",
			expectedErr: nil,
		},
		{
			desc:        "string with visible unicode characters",
			input:       "secret-测试-1",
			fieldName:   "objectName",
			expectedErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := validateNoInvisibleCharacters(tc.input, tc.fieldName)
			if tc.expectedErr != nil && err.Error() != tc.expectedErr.Error() || tc.expectedErr == nil && err != nil {
				t.Fatalf("expected err: %+v, got: %+v", tc.expectedErr, err)
			}
		})
	}
}

func TestValidateIntegration(t *testing.T) {
	cases := []struct {
		desc        string
		kv          types.KeyVaultObject
		expectedErr string
	}{
		{
			desc: "valid key vault object",
			kv: types.KeyVaultObject{
				ObjectName:    "secret1",
				ObjectType:    "secret",
				ObjectFormat:  "pem",
				ObjectVersion: "v1",
			},
			expectedErr: "",
		},
		{
			desc: "object name with zero width space",
			kv: types.KeyVaultObject{
				ObjectName:    "secret\u200B1",
				ObjectType:    "secret",
				ObjectFormat:  "pem",
				ObjectVersion: "v1",
			},
			expectedErr: "field objectName contains invisible character Zero Width Space (U+200B) at position 6",
		},
		{
			desc: "object alias with invisible character",
			kv: types.KeyVaultObject{
				ObjectName:    "secret1",
				ObjectAlias:   "alias\u200C1",
				ObjectType:    "secret",
				ObjectFormat:  "pem",
				ObjectVersion: "v1",
			},
			expectedErr: "field objectAlias contains invisible character Zero Width Non-Joiner (U+200C) at position 5",
		},
		{
			desc: "object type with invisible character",
			kv: types.KeyVaultObject{
				ObjectName:    "secret1",
				ObjectType:    "sec\u200Dret",
				ObjectFormat:  "pem",
				ObjectVersion: "v1",
			},
			expectedErr: "field objectType contains invisible character Zero Width Joiner (U+200D) at position 3",
		},
		{
			desc: "file permission with invisible character",
			kv: types.KeyVaultObject{
				ObjectName:     "secret1",
				ObjectType:     "secret",
				ObjectFormat:   "pem",
				ObjectVersion:  "v1",
				FilePermission: "0644\u2060",
			},
			expectedErr: "field filePermission contains invisible character Word Joiner (U+2060) at position 4",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := validate(tc.kv)
			if tc.expectedErr == "" && err != nil {
				t.Fatalf("expected no error, got: %+v", err)
			}
			if tc.expectedErr != "" && (err == nil || !strings.Contains(err.Error(), tc.expectedErr)) {
				t.Fatalf("expected err containing: %s, got: %+v", tc.expectedErr, err)
			}
		})
	}
}
