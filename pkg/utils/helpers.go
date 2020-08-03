package utils

import "regexp"

// RedactClientID Apply regex to a sensitive string and return the redacted value
func RedactClientID(sensitiveString string) string {
	r, _ := regexp.Compile(`^(\S{4})(\S|\s)*(\S{4})$`)
	return r.ReplaceAllString(sensitiveString, "$1##### REDACTED #####$3")
}
