package widget

import "unicode"

// FilterFunc defines which characters are allowed in an input.
// Returns true if the rune is allowed.
type FilterFunc func(rune) bool

// Common Filters

// FilterNumber allows only digits.
func FilterNumber(r rune) bool {
	return unicode.IsDigit(r)
}

// FilterPort allows digits.
func FilterPort(r rune) bool {
	return unicode.IsDigit(r)
}

// FilterNoSpace allows anything except spaces.
func FilterNoSpace(r rune) bool {
	return !unicode.IsSpace(r)
}

// FilterHostName allows alphanumerics, hyphens, and dots.
func FilterHostName(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.'
}

// FilterLabel allows specific char set suitable for labels (no spaces, safe chars).
func FilterLabel(r rune) bool {
	return FilterHostName(r)
}

// FilterAlphaNumeric allows letters and numbers.
func FilterAlphaNumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
