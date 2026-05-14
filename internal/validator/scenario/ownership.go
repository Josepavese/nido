package scenario

import (
	"strings"
	"unicode"

	"github.com/Josepavese/nido/internal/validator/util"
)

const validatorGeneratedPrefix = "nido-val-"

func validatorRandomName(kind string) string {
	kind = sanitizeResourceKind(kind)
	if kind == "" {
		kind = "resource"
	}
	return util.RandomName(validatorGeneratedPrefix + kind)
}

func isValidatorGeneratedVMName(name string) bool {
	return isValidatorGeneratedName(name)
}

func isValidatorGeneratedTemplateName(name string) bool {
	return isValidatorGeneratedName(name)
}

func isValidatorGeneratedName(name string) bool {
	if !strings.HasPrefix(name, validatorGeneratedPrefix) {
		return false
	}
	idx := strings.LastIndex(name, "-")
	if idx < len(validatorGeneratedPrefix) {
		return false
	}
	return isFixedHex(name[idx+1:], 6)
}

func sanitizeResourceKind(kind string) string {
	var out strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(kind) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func isFixedHex(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}
