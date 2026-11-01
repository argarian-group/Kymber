package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	if err := generate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate() error {
	root := "internal/templates"
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "catalog" {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, entry.Name(), "template.go")); err == nil {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return fmt.Errorf("no templates found under %s", root)
	}
	sort.Strings(names)
	var source bytes.Buffer
	source.WriteString("package catalog\n\nimport (\n\t\"kymber/backend/internal/templates\"\n")
	for i, name := range names {
		fmt.Fprintf(&source, "\ttemplate%d \"kymber/backend/internal/templates/%s\"\n", i, name)
	}
	source.WriteString(")\n\nfunc All() []templates.Template { return []templates.Template{")
	for i := range names {
		fmt.Fprintf(&source, "template%d.Definition(),", i)
	}
	source.WriteString("} }\n\nfunc Get(engine string) (templates.Template, bool) { switch engine {\n")
	for i := range names {
		fmt.Fprintf(&source, "case template%d.Definition().Engine: return template%d.Definition(), true\n", i, i)
	}
	source.WriteString("default: return templates.Template{}, false\n} }\n\nfunc DockerArgs(engine string, config templates.Config) ([]string, bool) { switch engine {\n")
	for i := range names {
		fmt.Fprintf(&source, "case template%d.Definition().Engine: return template%d.DockerArgs(config), true\n", i, i)
	}
	source.WriteString("default: return nil, false\n} }\n")
	formatted, err := format.Source(source.Bytes())
	if err != nil {
		return fmt.Errorf("format generated template registry: %w", err)
	}
	return os.WriteFile(filepath.Join(root, "catalog", "catalog_gen.go"), formatted, 0644)
}
