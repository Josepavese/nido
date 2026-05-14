package scenario

import (
	"reflect"
	"testing"

	"github.com/Josepavese/nido/internal/validator/config"
)

func TestStoppedNonValidatorVMsBlocksOnlyStoppedNonTestVMs(t *testing.T) {
	raw := `{
		"data": {
			"vms": [
				{"name": "nido-val-vm-a1b2c3", "state": "stopped"},
				{"name": "nido-win-pal", "state": "stopped"},
				{"name": "val-cfg-vm-abcdef", "state": "stopped"},
				{"name": "work-vm", "state": "running"}
			]
		}
	}`

	blocked, err := stoppedNonValidatorVMs(raw)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"nido-win-pal", "val-cfg-vm-abcdef"}; !reflect.DeepEqual(blocked, want) {
		t.Fatalf("blocked VMs = %#v, want %#v", blocked, want)
	}
}

func TestValidatorGeneratedNameRecognitionIsStrict(t *testing.T) {
	valid := validatorRandomName("vm-test")
	if !isValidatorGeneratedVMName(valid) {
		t.Fatalf("expected %q to be recognized as validator-generated", valid)
	}

	for _, name := range []string{
		"nido-win-pal",
		"cli-val-vm-abcdef",
		"val-cfg-vm-abcdef",
		"vm_template_src-abcdef",
		"vm_from_template-abcdef",
		"tpl_primary-abcdef",
		"nido-val-vm-test",
		"nido-val-vm-test-zzzzzz",
	} {
		if isValidatorGeneratedVMName(name) {
			t.Fatalf("did not expect %q to be recognized as validator-generated", name)
		}
	}
}

func TestDestructiveHelpersRefuseNonValidatorNames(t *testing.T) {
	ctx := &Context{Config: config.Config{NidoBin: "nido"}}

	vmRes := runDeleteValidatorVM(ctx, "nido-win-pal", 0)
	if vmRes.Result != "SKIP" {
		t.Fatalf("non-validator VM delete result = %s, want SKIP", vmRes.Result)
	}

	tplRes := runDeleteValidatorTemplate(ctx, "production-template", 0)
	if tplRes.Result != "SKIP" {
		t.Fatalf("non-validator template delete result = %s, want SKIP", tplRes.Result)
	}
}
