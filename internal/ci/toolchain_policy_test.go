package ci

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type workflowConfig struct {
	Jobs map[string]struct {
		Steps []struct {
			Uses string `yaml:"uses"`
			With struct {
				GoVersion     string `yaml:"go-version"`
				GoVersionFile string `yaml:"go-version-file"`
				CheckLatest   bool   `yaml:"check-latest"`
			} `yaml:"with"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func TestGoToolchainPolicy(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test location")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))

	policyData, err := os.ReadFile(filepath.Join(root, ".go-version"))
	if err != nil {
		t.Fatalf("read .go-version: %v", err)
	}
	policy := strings.TrimSpace(string(policyData))
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+$`).MatchString(policy) {
		t.Fatalf(".go-version must select a Go minor release, got %q", policy)
	}

	goModData, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	match := regexp.MustCompile(`(?m)^go[ \t]+([0-9]+\.[0-9]+\.[0-9]+)[ \t]*$`).FindSubmatch(goModData)
	if match == nil {
		t.Fatal("go.mod must declare an exact minimum Go patch version")
	}
	if minimum := string(match[1]); !strings.HasPrefix(minimum, policy+".") {
		t.Fatalf("go.mod minimum %s is outside the .go-version %s release line", minimum, policy)
	}

	workflowDir := filepath.Join(root, ".github", "workflows")
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		t.Fatalf("read workflows: %v", err)
	}

	setupCount := 0
	for _, entry := range entries {
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".yml" && filepath.Ext(entry.Name()) != ".yaml") {
			continue
		}
		path := filepath.Join(workflowDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		var workflow workflowConfig
		var document yaml.Node
		if err := yaml.Unmarshal(data, &document); err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		if hasMappingKey(&document, "GO_VERSION") {
			t.Errorf("%s duplicates the canonical .go-version policy", entry.Name())
		}
		if err := document.Decode(&workflow); err != nil {
			t.Fatalf("decode %s: %v", entry.Name(), err)
		}
		for jobName, job := range workflow.Jobs {
			for stepIndex, step := range job.Steps {
				if !strings.HasPrefix(step.Uses, "actions/setup-go@") {
					continue
				}
				setupCount++
				location := entry.Name() + ":" + jobName
				if step.Uses != "actions/setup-go@v6" {
					t.Errorf("%s step %d uses %q", location, stepIndex+1, step.Uses)
				}
				if step.With.GoVersion != "" {
					t.Errorf("%s step %d sets go-version instead of using .go-version", location, stepIndex+1)
				}
				if step.With.GoVersionFile != ".go-version" {
					t.Errorf("%s step %d go-version-file = %q", location, stepIndex+1, step.With.GoVersionFile)
				}
				if !step.With.CheckLatest {
					t.Errorf("%s step %d must set check-latest: true", location, stepIndex+1)
				}
			}
		}
	}
	if setupCount == 0 {
		t.Fatal("no actions/setup-go steps found")
	}
}

func hasMappingKey(node *yaml.Node, key string) bool {
	if node.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				return true
			}
		}
	}
	for _, child := range node.Content {
		if hasMappingKey(child, key) {
			return true
		}
	}
	return false
}
