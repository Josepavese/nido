package main

import "testing"

func TestShouldRemoveSpawnTemplate(t *testing.T) {
	cases := []struct {
		name         string
		linkedClones bool
		tpl          string
		downloaded   bool
		want         bool
	}{
		{"linked clones keep the template", true, "/images/ubuntu.qcow2", true, false},
		{"downloaded template is cleaned for standalone VMs", false, "/images/ubuntu.qcow2", true, true},
		{"cached image survives a standalone spawn", false, "/images/ubuntu.qcow2", false, false},
		{"user-provided disk path survives", false, "/tmp/disk.qcow2", false, false},
		{"no template", false, "", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRemoveSpawnTemplate(tc.linkedClones, tc.tpl, tc.downloaded); got != tc.want {
				t.Fatalf("shouldRemoveSpawnTemplate(%v, %q, %v) = %v, want %v", tc.linkedClones, tc.tpl, tc.downloaded, got, tc.want)
			}
		})
	}
}
