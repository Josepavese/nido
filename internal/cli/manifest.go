package cli

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

//go:embed commands.yaml
var embeddedManifest []byte

type Manifest struct {
	App      AppSpec             `yaml:"app"`
	Groups   []CommandGroup      `yaml:"groups"`
	Flags    map[string]FlagSpec `yaml:"flags"`
	Commands []CommandSpec       `yaml:"commands"`
}

type AppSpec struct {
	Use   string `yaml:"use"`
	Short string `yaml:"short"`
	Long  string `yaml:"long"`
}

type CommandGroup struct {
	ID    string `yaml:"id"`
	Title string `yaml:"title"`
}

type FlagSpec struct {
	Type       string `yaml:"type"`
	Long       string `yaml:"long"`
	Short      string `yaml:"short"`
	Usage      string `yaml:"usage"`
	Completion string `yaml:"completion"`
	Default    any    `yaml:"default"`
	Hidden     bool   `yaml:"hidden"`
}

type CommandFlagRef struct {
	Name     string `yaml:"name"`
	Required bool   `yaml:"required"`
}

type ArgsSpec struct {
	Min int `yaml:"min"`
	Max int `yaml:"max"`
}

type CommandSpec struct {
	ID                    string           `yaml:"id"`
	Use                   string           `yaml:"use"`
	Aliases               []string         `yaml:"aliases"`
	Group                 string           `yaml:"group"`
	Short                 string           `yaml:"short"`
	Long                  string           `yaml:"long"`
	Examples              []string         `yaml:"examples"`
	Hidden                bool             `yaml:"hidden"`
	Deprecated            string           `yaml:"deprecated"`
	Flags                 []CommandFlagRef `yaml:"flags"`
	Args                  ArgsSpec         `yaml:"args"`
	PositionalCompletions []string         `yaml:"positional_completions"`
	CustomCompletion      string           `yaml:"custom_completion"`
	Action                string           `yaml:"action"`
	Commands              []CommandSpec    `yaml:"commands"`
}

type ActionFunc func(cmd *cobra.Command, args []string)
type CompletionFunc func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

func LoadManifest() (*Manifest, error) {
	var manifest Manifest
	if err := yaml.Unmarshal(embeddedManifest, &manifest); err != nil {
		return nil, fmt.Errorf("failed to decode CLI manifest: %w", err)
	}
	return &manifest, manifest.Validate()
}

func (m *Manifest) Validate() error {
	if strings.TrimSpace(m.App.Use) == "" {
		return fmt.Errorf("manifest app.use is required")
	}
	if len(m.Commands) == 0 {
		return fmt.Errorf("manifest must define at least one command")
	}
	if len(m.Flags) == 0 {
		return fmt.Errorf("manifest must define flags")
	}
	seen := map[string]bool{}
	for _, cmd := range m.Commands {
		if err := validateCommand(cmd, m.Flags, seen); err != nil {
			return err
		}
	}
	return nil
}

func validateCommand(cmd CommandSpec, flags map[string]FlagSpec, seen map[string]bool) error {
	if strings.TrimSpace(cmd.ID) == "" {
		return fmt.Errorf("command without id")
	}
	if seen[cmd.ID] {
		return fmt.Errorf("duplicate command id: %s", cmd.ID)
	}
	seen[cmd.ID] = true
	if strings.TrimSpace(cmd.Use) == "" {
		return fmt.Errorf("command %s missing use", cmd.ID)
	}
	for _, ref := range cmd.Flags {
		if _, ok := flags[ref.Name]; !ok {
			return fmt.Errorf("command %s references unknown flag %s", cmd.ID, ref.Name)
		}
	}
	if cmd.Args.Max >= 0 && cmd.Args.Max < cmd.Args.Min {
		return fmt.Errorf("command %s has invalid args range", cmd.ID)
	}
	for _, child := range cmd.Commands {
		if err := validateCommand(child, flags, seen); err != nil {
			return err
		}
	}
	return nil
}
