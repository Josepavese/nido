package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const SchemaVersion = "1.0"

const jsonEnvVar = "NIDO_JSON"

type Response struct {
	SchemaVersion string      `json:"schema_version"`
	Command       string      `json:"command"`
	Status        string      `json:"status"`
	Timestamp     string      `json:"timestamp"`
	Data          interface{} `json:"data,omitempty"`
	Error         *Problem    `json:"error,omitempty"`
}

type Problem struct {
	Type     string      `json:"type"`
	Title    string      `json:"title"`
	Detail   string      `json:"detail"`
	Instance string      `json:"instance,omitempty"`
	Code     string      `json:"code,omitempty"`
	Hint     string      `json:"hint,omitempty"`
	Details  interface{} `json:"details,omitempty"`
}

func NewResponseOK(command string, data interface{}) Response {
	return Response{
		SchemaVersion: SchemaVersion,
		Command:       command,
		Status:        "ok",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Data:          data,
	}
}

func NewResponseError(command, code, title, detail, hint string, details interface{}) Response {
	return Response{
		SchemaVersion: SchemaVersion,
		Command:       command,
		Status:        "error",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Error: &Problem{
			Type:    "about:blank",
			Title:   title,
			Detail:  detail,
			Code:    code,
			Hint:    hint,
			Details: details,
		},
	}
}

func PrintJSON(resp Response) error {
	payload, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode JSON response: %v\n", err)
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(payload))
	return err
}

func SetJSONMode(enabled bool) {
	if enabled {
		_ = os.Setenv(jsonEnvVar, "1")
		return
	}
	_ = os.Unsetenv(jsonEnvVar)
}

func IsJSONMode() bool {
	return os.Getenv(jsonEnvVar) == "1"
}
