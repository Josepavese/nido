package scenario

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	cli "github.com/Josepavese/nido/v4/internal/cli"
	"github.com/Josepavese/nido/v4/internal/events"
	"github.com/Josepavese/nido/v4/internal/validator/report"
)

const eventScenarioTimeout = 10 * time.Second

func EventStream() Scenario { return Scenario{Name: "event-stream", Steps: []Step{eventStreamStep}} }

// The new CLI command is exercised in a private home, including a mutation from
// another process and invalid usage. No actual user state or VM is a fixture.
func eventStreamStep(ctx *Context) report.StepResult {
	result := report.StepResult{Command: "events --json", Result: "PASS", StartedAt: time.Now()}
	root, err := os.MkdirTemp("", "nv-")
	if err != nil {
		addAssertion(&result, "private_home", false, err.Error())
		return result
	}
	defer os.RemoveAll(root)
	nidoRoot := filepath.Join(root, ".nido")
	if err := os.MkdirAll(nidoRoot, 0700); err != nil {
		addAssertion(&result, "private_root", false, err.Error())
		return result
	}
	configPath := filepath.Join(nidoRoot, "config.env")
	if err := os.WriteFile(configPath, []byte("SSH_USER=fixture\n"), 0600); err != nil {
		addAssertion(&result, "private_config", false, err.Error())
		return result
	}
	environment := []string{}
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if !strings.EqualFold(key, "HOME") && !strings.EqualFold(key, "USERPROFILE") {
			environment = append(environment, value)
		}
	}
	environment = append(environment, "HOME="+root, "USERPROFILE="+root)
	deadline, cancel := context.WithTimeout(context.Background(), eventScenarioTimeout)
	defer cancel()
	stream := exec.CommandContext(deadline, ctx.Config.NidoBin, "events", "--json")
	stream.Env = environment
	stream.Dir = root
	stdout, err := stream.StdoutPipe()
	if err != nil {
		addAssertion(&result, "pipe", false, err.Error())
		return result
	}
	if err := stream.Start(); err != nil {
		addAssertion(&result, "start", false, err.Error())
		return result
	}
	defer func() { cancel(); stream.Wait() }()
	decoder := json.NewDecoder(stdout)
	read := func() (events.Event, error) {
		var envelope struct {
			Schema  string       `json:"schema_version"`
			Command string       `json:"command"`
			Data    events.Event `json:"data"`
		}
		err := decoder.Decode(&envelope)
		if err != nil {
			return events.Event{}, err
		}
		if envelope.Schema != cli.SchemaVersion || envelope.Command != "events" {
			return events.Event{}, os.ErrInvalid
		}
		return envelope.Data, nil
	}
	first, err := read()
	addAssertion(&result, "initial_resync", err == nil && first.Kind == events.Resync, errDetails(err))
	if err != nil {
		return result
	}
	mutation := exec.CommandContext(deadline, ctx.Config.NidoBin, "config", "set", "SSH_USER", "event-validator", "--json")
	mutation.Env = environment
	mutation.Dir = root
	output, err := mutation.CombinedOutput()
	addAssertion(&result, "independent_cli_mutation", err == nil, string(output))
	if err != nil {
		return result
	}
	seen := false
	for !seen {
		event, readErr := read()
		if readErr != nil {
			addAssertion(&result, "mutation_event", false, readErr.Error())
			return result
		}
		seen = event.Source == events.Operation && event.Kind == events.Completed && event.Action == events.ConfigSet
	}
	addAssertion(&result, "mutation_event", seen, "")
	invalid := exec.CommandContext(deadline, ctx.Config.NidoBin, "events", "unexpected", "--json")
	invalid.Env = environment
	invalid.Dir = root
	addAssertion(&result, "invalid_arguments", invalid.Run() != nil, "")
	return result
}
