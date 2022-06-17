package provider

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/types"
)

// validate is a helper function to validate the given object
func validate(kv types.KeyVaultObject) error {
	if err := validateObjectFormat(kv.ObjectFormat, kv.ObjectType); err != nil {
		return err
	}
	if err := validateObjectEncoding(kv.ObjectEncoding, kv.ObjectType); err != nil {
		return err
	}
	if err := validateFileName(kv.GetFileName()); err != nil {
		return err
	}
	return nil
}

// validateObjectFormat checks if the object format is valid and is supported
// for the given object type
func validateObjectFormat(objectFormat, objectType string) error {
	if len(objectFormat) == 0 {
		return nil
	}
	if !strings.EqualFold(objectFormat, types.ObjectFormatPEM) && !strings.EqualFold(objectFormat, types.ObjectFormatPFX) {
		return fmt.Errorf("invalid objectFormat: %v, should be PEM or PFX", objectFormat)
	}
	// Azure Key Vault returns the base64 encoded binary content only for type secret
	// for types cert/key, the content is always in pem format
	if objectFormat == types.ObjectFormatPFX && objectType != types.VaultObjectTypeSecret {
		return fmt.Errorf("PFX format only supported for objectType: secret")
	}
	return nil
}

// validateObjectEncoding checks if the object encoding is valid and is supported
// for the given object type
func validateObjectEncoding(objectEncoding, objectType string) error {
	if len(objectEncoding) == 0 {
		return nil
	}

	// ObjectEncoding is supported only for secret types
	if objectType != types.VaultObjectTypeSecret {
		return fmt.Errorf("objectEncoding only supported for objectType: secret")
	}

	if !strings.EqualFold(objectEncoding, types.ObjectEncodingHex) && !strings.EqualFold(objectEncoding, types.ObjectEncodingBase64) && !strings.EqualFold(objectEncoding, types.ObjectEncodingUtf8) {
		return fmt.Errorf("invalid objectEncoding: %v, should be hex, base64 or utf-8", objectEncoding)
	}

	return nil
}

// This validate will make sure fileName:
// 1. is not abs path
// 2. does not contain any '..' elements
// 3. does not start with '..'
// These checks have been implemented based on -
// [validateLocalDescendingPath] https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/validation/validation.go#L1158-L1170
// [validatePathNoBacksteps] https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/validation/validation.go#L1172-L1186
func validateFileName(fileName string) error {
	if len(fileName) == 0 {
		return fmt.Errorf("file name must not be empty")
	}
	// is not abs path
	if filepath.IsAbs(fileName) {
		return fmt.Errorf("file name must be a relative path")
	}
	// does not have any element which is ".."
	parts := strings.Split(filepath.ToSlash(fileName), "/")
	for _, item := range parts {
		if item == ".." {
			return fmt.Errorf("file name must not contain '..'")
		}
	}
	// fallback logic if .. is missed in the previous check
	if strings.Contains(fileName, "..") {
		return fmt.Errorf("file name must not contain '..'")
	}
	return nil
}
