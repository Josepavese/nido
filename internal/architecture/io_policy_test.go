package architecture

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var forbiddenPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{name: "fmt.Print", pattern: regexp.MustCompile(`\bfmt\.(Print|Printf|Println)\s*\(`)},
	{name: "fmt.Fprint(os.Stdout|os.Stderr)", pattern: regexp.MustCompile(`\bfmt\.(Fprint|Fprintf|Fprintln)\s*\(\s*os\.Std(out|err)\b`)},
	{name: "println", pattern: regexp.MustCompile(`\bprintln\s*\(`)},
	{name: "os.Stdout", pattern: regexp.MustCompile(`\bos\.Stdout\b`)},
	{name: "os.Stderr", pattern: regexp.MustCompile(`\bos\.Stderr\b`)},
	{name: "os.Stdin", pattern: regexp.MustCompile(`\bos\.Stdin\b`)},
}

func TestNoDirectUserIOInCoreLayers(t *testing.T) {
	repoRoot := locateRepoRoot(t)
	protectedDirs := []string{
		filepath.Join(repoRoot, "internal", "provider"),
		filepath.Join(repoRoot, "internal", "pkg", "sysutil"),
		filepath.Join(repoRoot, "internal", "image"),
		filepath.Join(repoRoot, "internal", "builder"),
	}

	var violations []string
	for _, dir := range protectedDirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(data)
			for _, rule := range forbiddenPatterns {
				if rule.pattern.FindStringIndex(text) != nil {
					rel, _ := filepath.Rel(repoRoot, path)
					violations = append(violations, rel+": "+rule.name)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}

	if len(violations) > 0 {
		t.Fatalf("direct user I/O found in protected layers:\n%s", strings.Join(violations, "\n"))
	}
}

func locateRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
