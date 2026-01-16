//go:build windows

package utils

// Umask is a no-op on Windows platforms
func Umask(mask int) (old int, err error) {
	return 0, nil
}
