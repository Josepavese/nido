package report

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// AssertionResult captures a single assertion outcome.
type AssertionResult struct {
	Name    string `json:"name"`
	Result  string `json:"result"` // PASS | FAIL | SKIP
	Details string `json:"details,omitempty"`
}

// StepResult records the outcome of an executed step.
type StepResult struct {
	RunID      string            `json:"run_id"`
	StepID     string            `json:"step_id"`
	Scenario   string            `json:"scenario"`
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	Env        map[string]string `json:"env,omitempty"`
	Cwd        string            `json:"cwd,omitempty"`
	ExitCode   int               `json:"exit_code"`
	DurationMs int64             `json:"duration_ms"`
	TimedOut   bool              `json:"timed_out,omitempty"`
	Stdout     string            `json:"stdout,omitempty"`
	Stderr     string            `json:"stderr,omitempty"`
	Assertions []AssertionResult `json:"assertions,omitempty"`
	Result     string            `json:"result"` // PASS | FAIL | SKIP
	Error      string            `json:"error,omitempty"`
	StartedAt  time.Time         `json:"started_at"`
}

// Summary aggregates counts across the run.
type Summary struct {
	Pass int
	Fail int
	Skip int
}

// Reporter writes NDJSON step results and can emit a summary.
type Reporter struct {
	mu      sync.Mutex
	file    *os.File
	writer  *bufio.Writer
	Summary Summary
}

// New creates a Reporter writing to the given NDJSON path.
func New(ndjsonPath string) (*Reporter, error) {
	if err := os.MkdirAll(filepath.Dir(ndjsonPath), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(ndjsonPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	return &Reporter{
		file:   f,
		writer: bufio.NewWriter(f),
	}, nil
}

// Close flushes and closes the underlying writer.
func (r *Reporter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.writer != nil {
		r.writer.Flush()
	}
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// WriteStep appends a step result as NDJSON.
func (r *Reporter) WriteStep(res StepResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	payload, err := json.Marshal(res)
	if err != nil {
		return err
	}
	if _, err := r.writer.Write(payload); err != nil {
		return err
	}
	if err := r.writer.WriteByte('\n'); err != nil {
		return err
	}
	switch res.Result {
	case "PASS":
		r.Summary.Pass++
	case "FAIL":
		r.Summary.Fail++
	case "SKIP":
		r.Summary.Skip++
	}
	return nil
}

// WriteSummary writes a human-readable summary to the given path.
func (r *Reporter) WriteSummary(path string, duration time.Duration) error {
	content := []byte(
		"Run summary\n" +
			"Duration: " + duration.String() + "\n" +
			"PASS: " + strconv.Itoa(r.Summary.Pass) + "\n" +
			"FAIL: " + strconv.Itoa(r.Summary.Fail) + "\n" +
			"SKIP: " + strconv.Itoa(r.Summary.Skip) + "\n")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}
