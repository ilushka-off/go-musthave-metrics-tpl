package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResetStmt(t *testing.T) {
	tests := []struct {
		name     string
		typeSrc  string
		contains []string
		exact    string
	}{
		{name: "int", typeSrc: "int", exact: "r.f = 0"},
		{name: "string", typeSrc: "string", exact: `r.f = ""`},
		{name: "bool", typeSrc: "bool", exact: "r.f = false"},
		{
			name:    "pointer to primitive",
			typeSrc: "*string",
			contains: []string{
				"if r.f != nil {",
				`*r.f = ""`,
			},
		},
		{name: "slice", typeSrc: "[]int", exact: "r.f = r.f[:0]"},
		{name: "map", typeSrc: "map[string]string", exact: "clear(r.f)"},
		{
			name:    "pointer to unknown named type",
			typeSrc: "*Foo",
			contains: []string{
				"if r.f != nil {",
				"interface{}(r.f).(interface{ Reset() })",
			},
		},
		{
			name:    "value of unknown named type",
			typeSrc: "Foo",
			contains: []string{
				"interface{}(&r.f).(interface{ Reset() })",
			},
		},
		{name: "fixed-size array", typeSrc: "[5]int", exact: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.typeSrc)
			if err != nil {
				t.Fatalf("parser.ParseExpr(%q) error = %v", tt.typeSrc, err)
			}

			got := resetStmt("r.f", expr)

			if len(tt.contains) == 0 {
				if got != tt.exact {
					t.Fatalf("resetStmt() = %q, want %q", got, tt.exact)
				}
				return
			}

			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Fatalf("resetStmt() = %q, want it to contain %q", got, want)
				}
			}
		})
	}
}

func TestRun_GeneratesResetGenGo(t *testing.T) {
	dir := t.TempDir()

	const src = `package sample

// generate:reset
type Foo struct {
	Name    string
	Tags    []string
	Attrs   map[string]string
	Age     *int
	Child   *Foo
}

type NotAnnotated struct {
	X int
}
`
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := run(dir); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	genPath := filepath.Join(dir, "reset.gen.go")
	data, err := os.ReadFile(genPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", genPath, err)
	}

	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, genPath, data, parser.ParseComments); err != nil {
		t.Fatalf("generated file is not valid Go: %v\n%s", err, data)
	}

	got := string(data)
	for _, want := range []string{
		"package sample",
		"func (f *Foo) Reset() {",
		"f.Name = \"\"",
		"f.Tags = f.Tags[:0]",
		"clear(f.Attrs)",
		"if f.Age != nil {",
		"*f.Age = 0",
		"if f.Child != nil {",
		"interface{}(f.Child).(interface{ Reset() })",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated code missing %q, got:\n%s", want, got)
		}
	}

	if strings.Contains(got, "NotAnnotated") {
		t.Errorf("generated code must not mention the unannotated struct, got:\n%s", got)
	}
}

func TestRun_NoAnnotatedStructsSkipsGeneration(t *testing.T) {
	dir := t.TempDir()

	const src = `package sample

type Foo struct {
	Name string
}
`
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := run(dir); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "reset.gen.go")); !os.IsNotExist(err) {
		t.Fatalf("reset.gen.go should not have been created, stat err = %v", err)
	}
}
