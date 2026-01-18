package scenario

import (
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/build"
	"github.com/Josepavese/nido/internal/validator/report"
)

// PreFlight builds the scenario covering version and doctor checks.
func PreFlight() Scenario {
	return Scenario{
		Name: "preflight",
		Steps: []Step{
			versionStep,
			doctorStep,
		},
	}
}

func versionStep(ctx *Context) report.StepResult {
	args := []string{"version", "--json"}
	res := runNido(ctx, "version", args, 15*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	payload, err := parseJSON(res.Stdout)
	addAssertion(&res, "json_parse", err == nil, errDetails(err))
	if err == nil {
		cmdVal, _ := mustGet(payload, "command")
		addAssertion(&res, "command_match", cmdVal == "version", "")
		status, _ := mustGet(payload, "status")
		addAssertion(&res, "status_ok", status == "ok", "")
		data, _ := mustGet(payload, "data")
		if m, ok := data.(map[string]interface{}); ok {
			if v, ok := m["version"]; ok {
				addAssertion(&res, "version_match", v == build.Version, "")
			} else {
				addAssertion(&res, "version_present", false, "missing data.version")
			}
		} else {
			addAssertion(&res, "data_object", false, "data not object")
		}
	}
	finalize(&res)
	return res
}

func doctorStep(ctx *Context) report.StepResult {
	args := []string{"doctor", "--json"}
	res := runNido(ctx, "doctor", args, 45*time.Second)
	addAssertion(&res, "exit_zero", res.ExitCode == 0, res.Stderr)
	payload, err := parseJSON(res.Stdout)
	addAssertion(&res, "json_parse", err == nil, errDetails(err))
	if err == nil {
		status, _ := mustGet(payload, "status")
		addAssertion(&res, "status_ok", status == "ok", "")
		data, _ := mustGet(payload, "data")
		if m, ok := data.(map[string]interface{}); ok {
			if summary, ok := m["summary"].(map[string]interface{}); ok {
				failed := summary["failed"]
				addAssertion(&res, "doctor_failed_zero", failed == float64(0), "failed != 0")
			} else {
				addAssertion(&res, "summary_present", false, "missing summary")
			}
			if reports, ok := m["reports"].([]interface{}); ok {
				allPresent := true
				for _, r := range reports {
					if str, ok := r.(string); ok {
						if strings.TrimSpace(str) == "" {
							allPresent = false
							break
						}
					}
				}
				addAssertion(&res, "reports_nonempty", allPresent && len(reports) > 0, "")
			}
		} else {
			addAssertion(&res, "data_object", false, "data not object")
		}
	}
	finalize(&res)
	return res
}

func errDetails(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
