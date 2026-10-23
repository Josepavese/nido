package build

import "testing"

func TestNormalizeVersion(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"(devel)", ""},
		{"4.8.0", "v4.8.0"},
		{"v4.8.0", "v4.8.0"},
		{" v4.8.0 ", "v4.8.0"},
	}
	for _, tc := range cases {
		if got := normalizeVersion(tc.in); got != tc.want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestVersionIsResolved(t *testing.T) {
	if Version == "" {
		t.Fatal("Version must never be empty")
	}
}
