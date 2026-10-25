package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Reusable graphical behavior belongs in the kit; pages only supply state and
// dimensions. This catches a return to local form rendering/width workarounds.
func TestTUIFormRenderingHasOneOwner(t *testing.T) {
	root := locateRepoRoot(t)
	err := filepath.WalkDir(filepath.Join(root, "internal/tui/app/pages"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(source, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			method, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || method.Sel.Name != "View" {
				return true
			}
			field, ok := method.X.(*ast.SelectorExpr)
			if ok && field.Sel.Name == "Form" {
				t.Errorf("form rendering bypasses kit viewport: %s", path)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
