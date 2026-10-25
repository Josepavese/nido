package machine

import "testing"

func TestArchitectureResolution(t *testing.T) {
	for _, tc := range []struct{ request, host, want string }{
		{"", ARM64, ARM64}, {Auto, AMD64, AMD64}, {AMD64, ARM64, AMD64},
		{"aarch64", AMD64, ARM64}, {"x86_64", ARM64, AMD64},
	} {
		got, err := Resolve(tc.request, tc.host)
		if err != nil || got != tc.want {
			t.Fatalf("Resolve(%q,%q) = %q,%v", tc.request, tc.host, got, err)
		}
	}
	for _, invalid := range []string{"armv7", "386", "../../amd64"} {
		if _, err := Resolve(invalid, AMD64); err == nil {
			t.Fatalf("accepted unsupported ISA %q", invalid)
		}
	}
}
