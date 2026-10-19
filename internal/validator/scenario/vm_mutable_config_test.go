package scenario

import (
	"encoding/json"
	"testing"
)

func TestRawConfigReadback(t *testing.T) {
	for _, tt := range []struct {
		body string
		want []string
		ok   bool
	}{
		{`{"raw_qemu_args":["-name","marker"]}`, []string{"-name", "marker"}, true},
		{`{"raw_qemu_args":["marker","-name"]}`, []string{"-name", "marker"}, false},
		{`{"raw_qemu_args":null}`, nil, true},
		{`{"raw_qemu_args":[]}`, nil, true},
		{`{}`, nil, false},
		{`{"raw_qemu_args":""}`, nil, false},
		{`{"raw_qemu_args":[null]}`, nil, false},
		{`{"raw_qemu_args":["-name","marker"]}`, nil, false},
	} {
		var vm map[string]interface{}
		if err := json.Unmarshal([]byte(tt.body), &vm); err != nil {
			t.Fatal(err)
		}
		if got := matchesRawConfigArgs(vm, tt.want); got != tt.ok {
			t.Errorf("readback %s matches %#v = %v, want %v", tt.body, tt.want, got, tt.ok)
		}
	}
}
