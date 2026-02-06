package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Josepavese/nido/internal/validator/config"
	"github.com/Josepavese/nido/internal/validator/report"
	"github.com/Josepavese/nido/internal/validator/runner"
	"github.com/Josepavese/nido/internal/validator/scenario"
	"github.com/Josepavese/nido/internal/validator/state"
	"github.com/Josepavese/nido/internal/validator/util"
)

func main() {
	runID := util.NewRunID()
	cfg := config.Parse(runID)
	cfg.NidoBin = ensureNidoBin(cfg)

	rep, err := report.New(cfg.LogFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create reporter: %v\n", err)
		os.Exit(1)
	}
	defer rep.Close()

	st := &state.State{}
	r := runner.New()

	ctx := &scenario.Context{
		RunID:    runID,
		Config:   cfg,
		Runner:   r,
		State:    st,
		Reporter: rep,
		Vars:     map[string]string{},
		Start:    time.Now(),
	}

	// SIGNAL HANDLING /////////////////////////////////////////////////////////
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println()
		fmt.Fprintf(os.Stderr, "%s\n", colorWarn(">> [INTERRUPT] Caught signal! Initiating emergency sweep..."))

		// Run global sweep to cleanup any left-over VMs
		scenario.Sweep(ctx)

		fmt.Fprintf(os.Stderr, "%s\n", colorWarn(">> [INTERRUPT] Sweep complete. Exiting."))
		time.Sleep(500 * time.Millisecond)
		os.Exit(130) // Standard SIGINT exit code
	}()
	///////////////////////////////////////////////////////////////////////////

	scenarios := []scenario.Scenario{
		scenario.PreClean(),
		scenario.PreFlight(),
		scenario.ImageCacheTemplate(),
		scenario.VMLifecycle(),
		scenario.VMMutableConfig(),
		scenario.VMSpawnResources(),
		scenario.WorkflowExec(),
		scenario.MCPProtocol(),
		scenario.Auxiliary(),
		scenario.Cleanup(),
	}

	if cfg.Scenario != "" {
		filtered := []scenario.Scenario{}
		for _, sc := range scenarios {
			if strings.EqualFold(sc.Name, cfg.Scenario) || sc.Name == "cleanup" {
				filtered = append(filtered, sc)
			}
		}
		if len(filtered) > 0 { // Allow single scenario + cleanup
			scenarios = filtered
		} else {
			fmt.Printf(">> [WARN] Scenario '%s' not found. Running all.\n", cfg.Scenario)
		}
	}

	ctx.Total = countSteps(scenarios)
	scenario.Run(ctx, scenarios)

	_ = rep.WriteSummary(cfg.SummaryFile, time.Since(ctx.Start))

	printRetroSummary(runID, rep, cfg, ctx)
}

func countSteps(scenarios []scenario.Scenario) int {
	total := 0
	for _, sc := range scenarios {
		total += len(sc.Steps)
	}
	return total
}

func printRetroSummary(runID string, rep *report.Reporter, cfg config.Config, ctx *scenario.Context) {
	duration := time.Since(ctx.Start).Round(time.Millisecond)
	fmt.Println()
	fmt.Println(colorBanner("==[ NIDO VALIDATOR // SYNTH RUN ]================================"))
	fmt.Printf("RUN ID  %s\n", runID)
	fmt.Printf("SCORE   PASS:%d  FAIL:%d  SKIP:%d  DURATION:%s\n", rep.Summary.Pass, rep.Summary.Fail, rep.Summary.Skip, duration)
	if len(ctx.Failures) == 0 {
		fmt.Println(colorBanner("STATUS  All systems green. High score maintained."))
	} else {
		fmt.Println(colorWarn("GLITCH  Detected anomalies:"))
		for _, f := range ctx.Failures {
			detail := failureDetail(f)
			fmt.Printf("  - [%s/%s] %s\n", strings.ToUpper(f.Scenario), f.Command, detail)
		}
	}
	fmt.Printf("LOGS    %s\n", cfg.LogFile)
	fmt.Printf("RECAP   %s\n", cfg.SummaryFile)
	fmt.Println("=================================================================")
}

func failureDetail(res report.StepResult) string {
	if res.Stderr != "" {
		return res.Stderr
	}
	for _, ar := range res.Assertions {
		if ar.Result == "FAIL" {
			if ar.Details != "" {
				return fmt.Sprintf("%s: %s", ar.Name, ar.Details)
			}
			return ar.Name
		}
	}
	if res.Error != "" {
		return res.Error
	}
	return "unknown glitch"
}

func colorBanner(val string) string {
	if os.Getenv("NO_COLOR") != "" {
		return val
	}
	return "\033[95m" + val + "\033[0m"
}

func colorWarn(val string) string {
	if os.Getenv("NO_COLOR") != "" {
		return val
	}
	return "\033[93m" + val + "\033[0m"
}

func ensureNidoBin(cfg config.Config) string {
	if cfg.NidoBin != "" {
		if _, err := exec.LookPath(cfg.NidoBin); err == nil {
			return cfg.NidoBin
		}
	}
	tmp := filepath.Join(os.TempDir(), "nido-validator-bin")
	fmt.Printf(">> [BUILD] nido not found, compiling into %s\n", tmp)
	cmd := exec.Command("go", "build", "-o", tmp, "./cmd/nido")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = cfg.WorkingDir
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "auto-build failed, falling back to %s: %v\n", cfg.NidoBin, err)
		return cfg.NidoBin
	}
	return tmp
}
