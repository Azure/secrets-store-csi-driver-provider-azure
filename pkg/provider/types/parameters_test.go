package types

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGetKeyVaultName(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		expected   string
	}{
		{
			name: "empty",
			parameters: map[string]string{
				KeyVaultNameParameter: "",
			},
			expected: "",
		},
		{
			name: "not empty",
			parameters: map[string]string{
				KeyVaultNameParameter: "test",
			},
			expected: "test",
		},
		{
			name: "trim spaces",
			parameters: map[string]string{
				KeyVaultNameParameter: " test ",
			},
			expected: "test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := GetKeyVaultName(test.parameters)
			if actual != test.expected {
				t.Errorf("GetKeyVaultName() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetCloudName(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		expected   string
	}{
		{
			name: "empty",
			parameters: map[string]string{
				CloudNameParameter: "",
			},
			expected: "",
		},
		{
			name: "not empty",
			parameters: map[string]string{
				CloudNameParameter: "test",
			},
			expected: "test",
		},
		{
			name: "trim spaces",
			parameters: map[string]string{
				CloudNameParameter: " test ",
			},
			expected: "test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := GetCloudName(test.parameters)
			if actual != test.expected {
				t.Errorf("GetCloudName() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetUsePodIdentity(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		expected   bool
	}{
		{
			name: "empty",
			parameters: map[string]string{
				UsePodIdentityParameter: "",
			},
			expected: false,
		},
		{
			name: "set to true",
			parameters: map[string]string{
				UsePodIdentityParameter: "true",
			},
			expected: true,
		},
		{
			name: "set to false",
			parameters: map[string]string{
				UsePodIdentityParameter: "false",
			},
			expected: false,
		},
		{
			name: "set to True",
			parameters: map[string]string{
				UsePodIdentityParameter: "True",
			},
			expected: true,
		},
		{
			name: "set to False",
			parameters: map[string]string{
				UsePodIdentityParameter: "False",
			},
			expected: false,
		},
		{
			name: "trim spaces",
			parameters: map[string]string{
				UsePodIdentityParameter: " true ",
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := GetUsePodIdentity(test.parameters)
			if err != nil {
				t.Errorf("GetUsePodIdentity() error = %v, expected nil", err)
			}
			if actual != test.expected {
				t.Errorf("GetUsePodIdentity() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetUsePodServiceAccountAnnotation(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		expected   bool
	}{
		{
			name: "empty",
			parameters: map[string]string{
				UsePodServiceAccountAnnotationParameter: "",
			},
			expected: false,
		},
		{
			name: "set to true",
			parameters: map[string]string{
				UsePodServiceAccountAnnotationParameter: "true",
			},
			expected: true,
		},
		{
			name: "set to false",
			parameters: map[string]string{
				UsePodServiceAccountAnnotationParameter: "false",
			},
			expected: false,
		},
		{
			name: "set to True",
			parameters: map[string]string{
				UsePodServiceAccountAnnotationParameter: "True",
			},
			expected: true,
		},
		{
			name: "set to False",
			parameters: map[string]string{
				UsePodServiceAccountAnnotationParameter: "False",
			},
			expected: false,
		},
		{
			name: "trim spaces",
			parameters: map[string]string{
				UsePodServiceAccountAnnotationParameter: " true ",
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := GetUsePodServiceAccountAnnotation(test.parameters)
			if err != nil {
				t.Errorf("GetUsePodServiceAccountAnnotation() error = %v, expected nil", err)
			}
			if actual != test.expected {
				t.Errorf("GetUsePodServiceAccountAnnotation() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetUsePodIdentityError(t *testing.T) {
	parameters := map[string]string{
		UsePodIdentityParameter: "test",
	}
	if _, err := GetUsePodIdentity(parameters); err == nil {
		t.Errorf("GetUsePodIdentity() error = nil, expected error")
	}
}

func TestGetUseVMManagedIdentity(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		expected   bool
	}{
		{
			name: "empty",
			parameters: map[string]string{
				UseVMManagedIdentityParameter: "",
			},
			expected: false,
		},
		{
			name: "set to true",
			parameters: map[string]string{
				UseVMManagedIdentityParameter: "true",
			},
			expected: true,
		},
		{
			name: "set to false",
			parameters: map[string]string{
				UseVMManagedIdentityParameter: "false",
			},
			expected: false,
		},
		{
			name: "set to True",
			parameters: map[string]string{
				UseVMManagedIdentityParameter: "True",
			},
			expected: true,
		},
		{
			name: "set to False",
			parameters: map[string]string{
				UseVMManagedIdentityParameter: "False",
			},
			expected: false,
		},
		{
			name: "trim spaces",
			parameters: map[string]string{
				UseVMManagedIdentityParameter: " true ",
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := GetUseVMManagedIdentity(test.parameters)
			if err != nil {
				t.Errorf("GetUseVMManagedIdentity() error = %v, expected nil", err)
			}
			if actual != test.expected {
				t.Errorf("GetUseVMManagedIdentity() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetUseVMManagedIdentityError(t *testing.T) {
	parameters := map[string]string{
		UseVMManagedIdentityParameter: "test",
	}
	if _, err := GetUseVMManagedIdentity(parameters); err == nil {
		t.Errorf("GetUseVMManagedIdentity() error = nil, expected error")
	}
}

func TestGetUserAssignedIdentityID(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		expected   string
	}{
		{
			name: "empty",
			parameters: map[string]string{
				UserAssignedIdentityIDParameter: "",
			},
			expected: "",
		},
		{
			name: "not empty",
			parameters: map[string]string{
				UserAssignedIdentityIDParameter: "test",
			},
			expected: "test",
		},
		{
			name: "trim spaces",
			parameters: map[string]string{
				UserAssignedIdentityIDParameter: " test ",
			},
			expected: "test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := GetUserAssignedIdentityID(test.parameters)
			if actual != test.expected {
				t.Errorf("GetUserAssignedIdentityID() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetTenantID(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		expected   string
	}{
		{
			name: "empty",
			parameters: map[string]string{
				TenantIDParameter: "",
			},
			expected: "",
		},
		{
			name: "not empty",
			parameters: map[string]string{
				TenantIDParameter: "test",
			},
			expected: "test",
		},
		{
			name: "trim spaces",
			parameters: map[string]string{
				TenantIDParameter: " test ",
			},
			expected: "test",
		},
		{
			name: "new tenantID parameter",
			parameters: map[string]string{
				"tenantID": "test",
			},
			expected: "test",
		},
		{
			name: "new tenantID parameter with spaces",
			parameters: map[string]string{
				"tenantID": " test ",
			},
			expected: "test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := GetTenantID(test.parameters)
			if actual != test.expected {
				t.Errorf("GetTenantID() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetCloudEnvFileName(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		expected   string
	}{
		{
			name: "empty",
			parameters: map[string]string{
				CloudEnvFileNameParameter: "",
			},
			expected: "",
		},
		{
			name: "not empty",
			parameters: map[string]string{
				CloudEnvFileNameParameter: "test",
			},
			expected: "test",
		},
		{
			name: "trim spaces",
			parameters: map[string]string{
				CloudEnvFileNameParameter: " test ",
			},
			expected: "test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := GetCloudEnvFileName(test.parameters)
			if actual != test.expected {
				t.Errorf("GetCloudEnvFileName() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetPodName(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		expected   string
	}{
		{
			name: "empty",
			parameters: map[string]string{
				CSIAttributePodName: "",
			},
			expected: "",
		},
		{
			name: "not empty",
			parameters: map[string]string{
				CSIAttributePodName: "test",
			},
			expected: "test",
		},
		{
			name: "trim spaces",
			parameters: map[string]string{
				CSIAttributePodName: " test ",
			},
			expected: "test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := GetPodName(test.parameters)
			if actual != test.expected {
				t.Errorf("GetPodName() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetPodNamespace(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		expected   string
	}{
		{
			name: "empty",
			parameters: map[string]string{
				CSIAttributePodNamespace: "",
			},
			expected: "",
		},
		{
			name: "not empty",
			parameters: map[string]string{
				CSIAttributePodNamespace: "test",
			},
			expected: "test",
		},
		{
			name: "trim spaces",
			parameters: map[string]string{
				CSIAttributePodNamespace: " test ",
			},
			expected: "test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := GetPodNamespace(test.parameters)
			if actual != test.expected {
				t.Errorf("GetPodNamespace() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetClientID(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		expected   string
	}{
		{
			name: "empty",
			parameters: map[string]string{
				"clientID": "",
			},
			expected: "",
		},
		{
			name: "not empty",
			parameters: map[string]string{
				"clientID": "test",
			},
			expected: "test",
		},
		{
			name: "trim spaces",
			parameters: map[string]string{
				"clientID": " test ",
			},
			expected: "test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := GetClientID(test.parameters)
			if actual != test.expected {
				t.Errorf("GetClientID() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetServiceAccountTokens(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		expected   string
	}{
		{
			name: "empty",
			parameters: map[string]string{
				CSIAttributeServiceAccountTokens: "",
			},
			expected: "",
		},
		{
			name: "not empty",
			parameters: map[string]string{
				CSIAttributeServiceAccountTokens: "test",
			},
			expected: "test",
		},
		{
			name: "trim spaces",
			parameters: map[string]string{
				CSIAttributeServiceAccountTokens: " test ",
			},
			expected: "test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := GetServiceAccountTokens(test.parameters)
			if actual != test.expected {
				t.Errorf("GetServiceAccountTokens() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetServiceAccountName(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		expected   string
	}{
		{
			name: "empty",
			parameters: map[string]string{
				CSIAttributeServiceAccountName: "",
			},
			expected: "",
		},
		{
			name: "not empty",
			parameters: map[string]string{
				CSIAttributeServiceAccountName: "test",
			},
			expected: "test",
		},
		{
			name: "trim spaces",
			parameters: map[string]string{
				CSIAttributeServiceAccountName: " test ",
			},
			expected: "test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := GetServiceAccountName(test.parameters)
			if actual != test.expected {
				t.Errorf("GetServiceAccountName() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetObjects(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		expected   string
	}{
		{
			name: "empty",
			parameters: map[string]string{
				ObjectsParameter: "",
			},
			expected: "",
		},
		{
			name: "not empty",
			parameters: map[string]string{
				ObjectsParameter: "test",
			},
			expected: "test",
		},
		{
			name: "trim spaces",
			parameters: map[string]string{
				ObjectsParameter: " test ",
			},
			expected: "test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := GetObjects(test.parameters)
			if actual != test.expected {
				t.Errorf("GetObjects() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetObjectsArray(t *testing.T) {
	tests := []struct {
		name     string
		objects  string
		expected StringArray
	}{
		{
			name:     "empty",
			objects:  "",
			expected: StringArray{},
		},
		{
			name:    "valid yaml",
			objects: "array:\n- |\n  filePermission: \"\"\n  objectAlias: \"\"\n  objectEncoding: \"\"\n  objectFormat: \"\"\n  objectName: secret1\n  objectType: cert\n  objectVersion: \"\"\n- |\n  filePermission: \"\"\n  objectAlias: \"\"\n  objectEncoding: \"\"\n  objectFormat: \"\"\n  objectName: secret2\n  objectType: cert\n  objectVersion: \"\"\n",
			expected: StringArray{
				Array: []string{
					"filePermission: \"\"\nobjectAlias: \"\"\nobjectEncoding: \"\"\nobjectFormat: \"\"\nobjectName: secret1\nobjectType: cert\nobjectVersion: \"\"\n",
					"filePermission: \"\"\nobjectAlias: \"\"\nobjectEncoding: \"\"\nobjectFormat: \"\"\nobjectName: secret2\nobjectType: cert\nobjectVersion: \"\"\n",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := GetObjectsArray(test.objects)
			if err != nil {
				t.Errorf("GetObjectsArray() error = %v", err)
			}
			if !reflect.DeepEqual(actual, test.expected) {
				t.Errorf("GetObjectsArray() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetObjectsArrayError(t *testing.T) {
	objects := "invalid"
	if _, err := GetObjectsArray(objects); err == nil {
		t.Errorf("GetObjectsArray() error is nil, expected error")
	}
}

func TestIsSyncingSingleVersion(t *testing.T) {
	tests := []struct {
		name     string
		object   KeyVaultObject
		expected bool
	}{
		{
			name:     "object version history uninitialized",
			object:   KeyVaultObject{},
			expected: true,
		},
		{
			name: "object version history set to 0",
			object: KeyVaultObject{
				ObjectVersionHistory: 0,
			},
			expected: true,
		},
		{
			name: "object version history set to 1",
			object: KeyVaultObject{
				ObjectVersionHistory: 1,
			},
			expected: true,
		},
		{
			name: "object version history set higher than 1",
			object: KeyVaultObject{
				ObjectVersionHistory: 4,
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.object.IsSyncingSingleVersion()
			if actual != test.expected {
				t.Errorf("IsSyncingSingleVersion() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetObjectUID(t *testing.T) {
	tests := []struct {
		name     string
		object   KeyVaultObject
		expected string
	}{
		{
			name: "syncing a single version (with alias)",
			object: KeyVaultObject{
				ObjectType:    "secret",
				ObjectName:    "single-version",
				ObjectAlias:   "alias",
				ObjectVersion: "version-id",
			},
			expected: "secret/single-version",
		},
		{
			name: "syncing a single version (without alias)",
			object: KeyVaultObject{
				ObjectType:    "secret",
				ObjectName:    "single-version",
				ObjectVersion: "version-id",
			},
			expected: "secret/single-version",
		},
		{
			name: "syncing multiple versions (with alias)",
			object: KeyVaultObject{
				ObjectType:           "secret",
				ObjectName:           "multiple-versions",
				ObjectAlias:          filepath.Join("alias", "0"),
				ObjectVersion:        "version-id",
				ObjectVersionHistory: 10,
			},
			expected: "secret/multiple-versions/0",
		},
		{
			name: "syncing multiple versions (without alias)",
			object: KeyVaultObject{
				ObjectType:           "secret",
				ObjectName:           "multiple-versions",
				ObjectAlias:          filepath.Join("multiple-versions", "0"),
				ObjectVersion:        "version-id",
				ObjectVersionHistory: 10,
			},
			expected: "secret/multiple-versions/0",
		},
		{
			name: "syncing multiple versions with multiple levels in path",
			object: KeyVaultObject{
				ObjectType:           "secret",
				ObjectName:           "multiple-versions",
				ObjectAlias:          filepath.Join("folder", "multiple-versions", "8"),
				ObjectVersion:        "version-id",
				ObjectVersionHistory: 10,
			},
			expected: "secret/multiple-versions/8",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.object.GetObjectUID()
			if actual != test.expected {
				t.Errorf("GetObjectUID() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetFileName(t *testing.T) {
	tests := []struct {
		name     string
		object   KeyVaultObject
		expected string
	}{
		{
			name: "empty",
			object: KeyVaultObject{
				ObjectName: "",
			},
			expected: "",
		},
		{
			name: "object alias and object name",
			object: KeyVaultObject{
				ObjectName:  "test",
				ObjectAlias: "alias",
			},
			expected: "alias",
		},
		{
			name: "object name only",
			object: KeyVaultObject{
				ObjectName: "test",
			},
			expected: "test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.object.GetFileName()
			if actual != test.expected {
				t.Errorf("GetFileName() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetFilePermission(t *testing.T) {
	cases := []struct {
		name     string
		object   KeyVaultObject
		expected int32
	}{
		{
			name: "valid file permission",
			object: KeyVaultObject{
				FilePermission: "0600",
			},
			expected: 0600,
		},
		{
			name:     "empty file permission",
			object:   KeyVaultObject{},
			expected: 0644,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			actual, err := test.object.GetFilePermission(os.FileMode(0644))
			if err != nil {
				t.Errorf("GetFilePermission() error = %v", err)
			}
			if actual != test.expected {
				t.Errorf("GetFilePermission() = %v, expected %v", actual, test.expected)
			}
		})
	}
}

func TestGetFilePermissionError(t *testing.T) {
	cases := []struct {
		name   string
		object KeyVaultObject
	}{
		{
			name: "invalid file permission",
			object: KeyVaultObject{
				FilePermission: "0900",
			},
		},
		{
			name: "invalid octal number",
			object: KeyVaultObject{
				FilePermission: "900",
			},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.object.GetFilePermission(os.FileMode(0644)); err == nil {
				t.Errorf("GetFilePermission() error = nil, expected error")
			}
		})
	}
}
