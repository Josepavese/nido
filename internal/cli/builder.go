package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Builder struct {
	Manifest              *Manifest
	Actions               map[string]ActionFunc
	FlagCompletions       map[string]CompletionFunc
	PositionalCompletions map[string]CompletionFunc
}

func (b *Builder) BuildRoot() (*cobra.Command, error) {
	if b.Manifest == nil {
		return nil, fmt.Errorf("builder manifest is nil")
	}

	root := &cobra.Command{
		Use:           b.Manifest.App.Use,
		Short:         b.Manifest.App.Short,
		Long:          b.Manifest.App.Long,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.Run = func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	}
	root.CompletionOptions.DisableDefaultCmd = true

	for _, group := range b.Manifest.Groups {
		root.AddGroup(&cobra.Group{ID: group.ID, Title: group.Title})
	}

	for _, spec := range b.Manifest.Commands {
		cmd, err := b.buildCommand(spec)
		if err != nil {
			return nil, err
		}
		root.AddCommand(cmd)
	}

	return root, nil
}

func (b *Builder) buildCommand(spec CommandSpec) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:        spec.Use,
		Aliases:    spec.Aliases,
		GroupID:    spec.Group,
		Short:      spec.Short,
		Long:       spec.Long,
		Hidden:     spec.Hidden,
		Deprecated: spec.Deprecated,
		Example:    strings.Join(spec.Examples, "\n"),
		Args:       makeArgsValidator(spec.Args),
	}

	if spec.Action != "" {
		action := b.Actions[spec.Action]
		if action == nil {
			return nil, fmt.Errorf("command %s references unknown action %s", spec.ID, spec.Action)
		}
		cmd.Run = action
	}

	for _, ref := range spec.Flags {
		flagSpec := b.Manifest.Flags[ref.Name]
		if err := bindFlag(cmd.Flags(), flagSpec); err != nil {
			return nil, fmt.Errorf("command %s flag %s: %w", spec.ID, ref.Name, err)
		}
		if ref.Required {
			if err := cmd.MarkFlagRequired(flagSpec.Long); err != nil {
				return nil, err
			}
		}
		if flagSpec.Hidden {
			if err := cmd.Flags().MarkHidden(flagSpec.Long); err != nil {
				return nil, err
			}
		}
		if flagSpec.Completion != "" {
			if err := b.attachFlagCompletion(cmd, flagSpec); err != nil {
				return nil, err
			}
		}
	}

	if spec.CustomCompletion != "" {
		if fn := b.PositionalCompletions[spec.CustomCompletion]; fn != nil {
			cmd.ValidArgsFunction = fn
		} else {
			return nil, fmt.Errorf("command %s references unknown custom completion %s", spec.ID, spec.CustomCompletion)
		}
	} else if len(spec.PositionalCompletions) > 0 {
		cmd.ValidArgsFunction = b.makePositionalCompletion(spec.PositionalCompletions)
	}

	for _, childSpec := range spec.Commands {
		child, err := b.buildCommand(childSpec)
		if err != nil {
			return nil, err
		}
		cmd.AddCommand(child)
	}

	if spec.Action == "" && len(spec.Commands) == 0 {
		return nil, fmt.Errorf("command %s has neither action nor children", spec.ID)
	}
	if spec.Action == "" && len(spec.Commands) > 0 {
		cmd.Run = func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		}
	}

	return cmd, nil
}

func bindFlag(flags *pflag.FlagSet, spec FlagSpec) error {
	switch spec.Type {
	case "bool":
		defaultVal, _ := spec.Default.(bool)
		flags.BoolP(spec.Long, spec.Short, defaultVal, spec.Usage)
	case "int":
		defaultVal, _ := spec.Default.(int)
		flags.IntP(spec.Long, spec.Short, defaultVal, spec.Usage)
	case "string":
		defaultVal, _ := spec.Default.(string)
		flags.StringP(spec.Long, spec.Short, defaultVal, spec.Usage)
	case "stringArray":
		defaultVal, _ := toStringSlice(spec.Default)
		flags.StringArrayP(spec.Long, spec.Short, defaultVal, spec.Usage)
	default:
		return fmt.Errorf("unsupported flag type %s", spec.Type)
	}
	return nil
}

func toStringSlice(v any) ([]string, bool) {
	switch typed := v.(type) {
	case []string:
		return typed, true
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	default:
		return nil, false
	}
}

func makeArgsValidator(spec ArgsSpec) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < spec.Min {
			return fmt.Errorf("requires at least %d arg(s)", spec.Min)
		}
		if spec.Max >= 0 && len(args) > spec.Max {
			return fmt.Errorf("accepts at most %d arg(s)", spec.Max)
		}
		return nil
	}
}

func (b *Builder) attachFlagCompletion(cmd *cobra.Command, spec FlagSpec) error {
	fn := b.FlagCompletions[spec.Completion]
	if fn == nil {
		return fmt.Errorf("unknown flag completion source %s", spec.Completion)
	}
	return cmd.RegisterFlagCompletionFunc(spec.Long, fn)
}

func (b *Builder) makePositionalCompletion(sources []string) CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		idx := len(args)
		if idx >= len(sources) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		source := sources[idx]
		if strings.TrimSpace(source) == "" {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		fn := b.PositionalCompletions[source]
		if fn == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return fn(cmd, args, toComplete)
	}
}
