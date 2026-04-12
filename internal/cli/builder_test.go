package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestLoadManifestAndBuildRoot(t *testing.T) {
	manifest, err := LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	actions := map[string]ActionFunc{}
	completions := map[string]CompletionFunc{}
	collectManifestHandlers(manifest.Commands, manifest.Flags, actions, completions)

	builder := &Builder{
		Manifest:              manifest,
		Actions:               actions,
		FlagCompletions:       completions,
		PositionalCompletions: completions,
	}

	root, err := builder.BuildRoot()
	if err != nil {
		t.Fatalf("BuildRoot failed: %v", err)
	}

	cmd, _, err := root.Find([]string{"completion", "bash"})
	if err != nil {
		t.Fatalf("root.Find(completion bash) failed: %v", err)
	}
	if cmd == nil || cmd.Name() != "bash" {
		t.Fatalf("unexpected command lookup result: %#v", cmd)
	}
}

func TestRootHelpIncludesGroupsAndCommands(t *testing.T) {
	manifest, err := LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	actions := map[string]ActionFunc{}
	completions := map[string]CompletionFunc{}
	collectManifestHandlers(manifest.Commands, manifest.Flags, actions, completions)

	builder := &Builder{
		Manifest:              manifest,
		Actions:               actions,
		FlagCompletions:       completions,
		PositionalCompletions: completions,
	}
	root, err := builder.BuildRoot()
	if err != nil {
		t.Fatalf("BuildRoot failed: %v", err)
	}

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	help := out.String()
	for _, want := range []string{"VM Management", "Storage & Genetics", "System Ops", "completion", "spawn"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help output missing %q\n%s", want, help)
		}
	}
}

func TestBashCompletionGeneratesScript(t *testing.T) {
	manifest, err := LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	actions := map[string]ActionFunc{}
	completions := map[string]CompletionFunc{}
	collectManifestHandlers(manifest.Commands, manifest.Flags, actions, completions)

	builder := &Builder{
		Manifest:              manifest,
		Actions:               actions,
		FlagCompletions:       completions,
		PositionalCompletions: completions,
	}
	root, err := builder.BuildRoot()
	if err != nil {
		t.Fatalf("BuildRoot failed: %v", err)
	}

	var script bytes.Buffer
	if err := root.GenBashCompletionV2(&script, true); err != nil {
		t.Fatalf("GenBashCompletionV2 failed: %v", err)
	}

	text := script.String()
	if !strings.Contains(text, "_nido") {
		t.Fatalf("expected bash completion script to contain command name, got:\n%s", text)
	}
}

func collectManifestHandlers(commands []CommandSpec, flags map[string]FlagSpec, actions map[string]ActionFunc, completions map[string]CompletionFunc) {
	for _, spec := range commands {
		if spec.Action != "" {
			actions[spec.Action] = func(cmd *cobra.Command, args []string) {}
		}

		if spec.CustomCompletion != "" {
			completions[spec.CustomCompletion] = noOpCompletion
		}
		for _, source := range spec.PositionalCompletions {
			if strings.TrimSpace(source) == "" {
				continue
			}
			completions[source] = noOpCompletion
		}
		for _, ref := range spec.Flags {
			flagSpec := flags[ref.Name]
			if strings.TrimSpace(flagSpec.Completion) != "" {
				completions[flagSpec.Completion] = noOpCompletion
			}
		}

		collectManifestHandlers(spec.Commands, flags, actions, completions)
	}
}

func noOpCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}
