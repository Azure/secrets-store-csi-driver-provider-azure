package types

import (
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
