package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The device owner and portable consumers are explicit package boundaries.
// Harnesses under validator qualify native backends; they are not application consumers.
func TestDeviceAndAdapterBoundaries(t *testing.T) {
	root := locateRepoRoot(t)
	for _, dir := range []string{"internal", "cmd/nido"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			source, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			aliases := map[string]string{}
			for _, imp := range source.Imports {
				value, _ := strconv.Unquote(imp.Path.Value)
				name := filepath.Base(value)
				if imp.Name != nil {
					name = imp.Name.Name
				}
				aliases[name] = value
				if value == "github.com/fsnotify/fsnotify" && !strings.HasPrefix(rel, "internal/pkg/sysutil/") {
					t.Errorf("native watcher outside PAL: %s", rel)
				}
				if strings.HasPrefix(rel, "internal/events/") && strings.HasPrefix(value, "github.com/Josepavese/nido/v4/internal/") && value != "github.com/Josepavese/nido/v4/internal/pkg/sysutil" && value != "github.com/Josepavese/nido/v4/internal/pkg/identifier" {
					t.Errorf("event contract depends on domain adapter: %s", rel)
				}
				if value == "os/exec" && (strings.HasPrefix(rel, "cmd/nido/") || strings.HasPrefix(rel, "internal/mcp/") || strings.HasPrefix(rel, "internal/tui/") || strings.HasPrefix(rel, "internal/builder/") || strings.HasPrefix(rel, "internal/image/")) {
					t.Errorf("native process API outside PAL: %s", rel)
				}
			}
			ast.Inspect(source, func(node ast.Node) bool {
				if literal, ok := node.(*ast.BasicLit); ok && literal.Kind == token.STRING {
					value, _ := strconv.Unquote(literal.Value)
					if (strings.HasPrefix(value, "/sys/") || value == "/dev/kvm") && !strings.HasPrefix(rel, "internal/hardware/") && !strings.HasPrefix(rel, "internal/validator/") {
						t.Errorf("device access outside HAL: %s: %s", rel, value)
					}
				}
				if selector, ok := node.(*ast.SelectorExpr); ok {
					if selector.Sel.Name == "DeviceValue" && !strings.HasPrefix(rel, "internal/hardware/") && !strings.HasPrefix(rel, "internal/pkg/sysutil/") {
						t.Errorf("raw device access outside HAL/PAL: %s", rel)
					}
					if pkg, ok := selector.X.(*ast.Ident); ok {
						if aliases[pkg.Name] == "net" && (selector.Sel.Name == "Dial" || selector.Sel.Name == "DialTimeout" || selector.Sel.Name == "Listen") && (strings.HasPrefix(rel, "internal/tui/") || strings.HasPrefix(rel, "internal/mcp/") || strings.HasPrefix(rel, "cmd/nido/")) {
							t.Errorf("native event/network connection outside PAL: %s", rel)
						}
						if aliases[pkg.Name] == "github.com/Josepavese/nido/v4/internal/provider" && selector.Sel.Name == "QemuProvider" {
							t.Errorf("concrete VM adapter type in consumer: %s", rel)
						}
						if aliases[pkg.Name] == "github.com/Josepavese/nido/v4/internal/image" && selector.Sel.Name == "Downloader" {
							t.Errorf("concrete transfer adapter in consumer: %s", rel)
						}
						if aliases[pkg.Name] == "runtime" && (selector.Sel.Name == "GOOS" || selector.Sel.Name == "GOARCH") && (strings.HasPrefix(rel, "internal/provider/") || strings.HasPrefix(rel, "internal/builder/") || strings.HasPrefix(rel, "cmd/nido/") || strings.HasPrefix(rel, "internal/mcp/") || strings.HasPrefix(rel, "internal/tui/")) {
							t.Errorf("runtime platform selection outside native backend: %s", rel)
						}
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestHALDependsOnPALRatherThanNativeFilesystemOrProcess(t *testing.T) {
	root := locateRepoRoot(t)
	files, err := filepath.Glob(filepath.Join(root, "internal/hardware/*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		source, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range source.Imports {
			value, _ := strconv.Unquote(imp.Path.Value)
			if value == "os" || value == "os/exec" {
				t.Errorf("HAL bypasses PAL in %s: %s", path, value)
			}
		}
	}
}
