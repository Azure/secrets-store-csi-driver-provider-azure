package provider

import (
	"fmt"
	"testing"
	"time"
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

func TestValidateObjectVersionAndNotAfter(t *testing.T) {
	cases := []struct {
		desc           string
		objectVersion  string
		objectNotAfter time.Time
		expectedErr    error
	}{
		{
			desc:           "no objectNotAfter and no objectVersion",
			objectVersion:  "",
			objectNotAfter: time.Time{},
			expectedErr:    nil,
		},
		{
			desc:           "objectNotAfter set with no objectVersion",
			objectVersion:  "",
			objectNotAfter: time.Date(2026, time.June, 5, 19, 41, 0, 0, time.UTC),
			expectedErr:    nil,
		},
		{
			desc:           "objectNotAfter set with seconds",
			objectVersion:  "",
			objectNotAfter: time.Date(2026, time.June, 5, 19, 41, 30, 0, time.UTC),
			expectedErr:    nil,
		},
		{
			desc:           "objectVersion and objectNotAfter are mutually exclusive",
			objectVersion:  "latest",
			objectNotAfter: time.Date(2026, time.June, 5, 19, 41, 0, 0, time.UTC),
			expectedErr:    fmt.Errorf("objectVersion and objectNotAfter are mutually exclusive"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := validateObjectVersionAndNotAfter(tc.objectVersion, tc.objectNotAfter)
			if tc.expectedErr != nil && err.Error() != tc.expectedErr.Error() || tc.expectedErr == nil && err != nil {
				t.Fatalf("expected err: %+v, got: %+v", tc.expectedErr, err)
			}
		})
	}
}
