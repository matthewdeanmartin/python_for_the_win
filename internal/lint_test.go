package internal_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestNoShellExecutionLiterals(t *testing.T) {
	e := filepath.WalkDir("..", func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, e := parser.ParseFile(token.NewFileSet(), p, nil, 0)
		if e != nil {
			return e
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if l, ok := n.(*ast.BasicLit); ok && l.Kind == token.STRING {
				v, _ := strconv.Unquote(l.Value)
				v = strings.ToLower(v)
				for _, bad := range []string{"cmd.exe", "cmd /c", "powershell", "pwsh"} {
					if strings.Contains(v, bad) {
						t.Errorf("forbidden shell literal in %s", p)
					}
				}
			}
			return true
		})
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}
