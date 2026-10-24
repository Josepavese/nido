// Package identifier owns Nido's portable single-component storage identities.
package identifier

import "fmt"

// Validate accepts the same VM/template/cache identity on every supported host.
// An identity never selects a path outside the owning storage directory.
func Validate(name string) error {
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("invalid identifier %q", name)
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
			return fmt.Errorf("invalid identifier %q: use letters, digits, dots, hyphens or underscores", name)
		}
	}
	return nil
}
