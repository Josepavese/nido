package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/image"
)

// Config
const (
	TestVMParams = "test-validator"
	NidoCmd      = "go"
	NidoArgs     = "run ./cmd/nido"
)

func main() {
	registryFile := flag.String("registry", "registry/images.json", "Path to registry file")
	filter := flag.String("filter", "", "Filter images by name (substring match)")
	flag.Parse()

	fmt.Println("🚀 Starting Nido Registry Validation Protocol...")

	// 1. Load Registry
	data, err := os.ReadFile(*registryFile)
	if err != nil {
		fatal("Failed to read registry: %v", err)
	}

	var catalog image.Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		fatal("Failed to parse registry: %v", err)
	}

	fmt.Printf("📋 Found %d images in catalog.\n", len(catalog.Images))

	successCount := 0
	failCount := 0
	failures := []string{}

	// 2. Validate Loop
	for _, img := range catalog.Images {
		// Apply filter if provided
		if *filter != "" && !strings.Contains(img.Name, *filter) {
			continue
		}

		for _, ver := range img.Versions {
			imageTag := fmt.Sprintf("%s:%s", img.Name, ver.Version)
			vmName := fmt.Sprintf("%s-%s-%s", TestVMParams, img.Name, strings.ReplaceAll(ver.Version, ".", "-"))

			fmt.Printf("\n------------------------------------------------\n")
			fmt.Printf("🧪 Testing Image: %s\n", imageTag)
			fmt.Printf("------------------------------------------------\n")

			start := time.Now()
			if err := runTest(vmName, imageTag); err != nil {
				fmt.Printf("\n❌ FAILED: %s (%v)\n", imageTag, err)
				failCount++
				failures = append(failures, fmt.Sprintf("%s: %v", imageTag, err))

				// Cleanup leftovers (silently)
				nidoSilent("delete", vmName)
			} else {
				fmt.Printf("\n✅ PASSED: %s (Time: %s)\n", imageTag, time.Since(start).Round(time.Second))
				successCount++
			}
		}
	}

	// 3. Summary
	fmt.Printf("\n================================================\n")
	fmt.Printf("SUMMARY\n")
	fmt.Printf("Passed: %d\n", successCount)
	fmt.Printf("Failed: %d\n", failCount)
	if len(failures) > 0 {
		fmt.Printf("\nFailures:\n")
		for _, f := range failures {
			fmt.Printf("- %s\n", f)
		}
		os.Exit(1)
	}
	fmt.Printf("✨ All systems nominal.\n")
}

func runTest(vmName, imageTag string) error {
	// A. Spawn (Handles Download + Start)
	// We force --image to ensure it pulls/verifies
	fmt.Printf("🐣 Spawning %s...\n", vmName)
	if err := nido("spawn", vmName, "--image", imageTag); err != nil {
		return fmt.Errorf("spawn failed: %w", err)
	}

	// B. Wait for SSH
	// Poll for connectivity
	fmt.Printf("⏳ Waiting for connectivity...\n")
	connected := false
	for i := 0; i < 60; i++ { // Try for ~3 minutes (cloud-init is slow)
		// Try running a simple echo command
		// "nido ssh <vm> echo ok"
		time.Sleep(3 * time.Second)

		out, err := nidoOutput("ssh", vmName, "echo", "READY")
		if err == nil && strings.Contains(out, "READY") {
			connected = true
			fmt.Printf("   Connected!\n")
			break
		}

		// Debug output every 10 attempts
		if i > 0 && i%10 == 0 {
			msg := strings.TrimSpace(out)
			if msg == "" {
				msg = "no response"
			}
			fmt.Printf(" [%s] ", msg)
		}
		fmt.Printf(".")
	}
	fmt.Printf("\n")

	if !connected {
		// Dump info for debugging
		nido("info", vmName)
		return fmt.Errorf("ssh connectivity timeout")
	}

	// C. Delete
	fmt.Printf("🗑️  Cleaning up...\n")
	if err := nido("delete", vmName); err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	return nil
}

func nido(args ...string) error {
	fullArgs := append(strings.Split(NidoArgs, " "), args...)
	cmd := exec.Command(NidoCmd, fullArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func nidoSilent(args ...string) error {
	fullArgs := append(strings.Split(NidoArgs, " "), args...)
	cmd := exec.Command(NidoCmd, fullArgs...)
	// Explicitly discard output
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func nidoOutput(args ...string) (string, error) {
	fullArgs := append(strings.Split(NidoArgs, " "), args...)
	cmd := exec.Command(NidoCmd, fullArgs...)
	// Capture output, don't pipe to stdout
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func fatal(format string, args ...interface{}) {
	fmt.Printf("❌ "+format+"\n", args...)
	os.Exit(1)
}
