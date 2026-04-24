package utils

import "regexp"

var rfcPattern = regexp.MustCompile(`^[A-ZÑ&]{3,4}\d{6}(?:[A-Z\d]{3})?$`)

func ValidarRFC(rfc string) bool {
	return rfcPattern.MatchString(rfc)
}
