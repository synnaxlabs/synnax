// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package migrate provides an Oracle plugin that generates gorp migration registration
// files. It generates migrate.gen.go in the parent service package containing migration
// chains: codec transitions and schema resolution migrations.
package migrate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

type Plugin struct{}

func New() *Plugin                            { return &Plugin{} }
func (p *Plugin) Name() string                { return "go/migrate" }
func (p *Plugin) Domains() []string           { return []string{"go"} }
func (p *Plugin) Requires() []string          { return []string{"go/types", "go/marshal"} }
func (p *Plugin) Check(*plugin.Request) error { return nil }

var goPostWriter = &exec.PostWriter{
	Extensions: []string{".go"},
	Commands:   [][]string{{"gofmt", "-w"}},
}

func (p *Plugin) PostWrite(files []string) error {
	return goPostWriter.PostWrite(files)
}

// migrationEntry describes a gorp entry type that needs migration support.
type migrationEntry struct {
	GoName  string
	KeyName string
	GoPath  string
	// SchemaChange is non-nil when the schema changed between snapshots.
	SchemaChange *schemaChange
}

type schemaChange struct {
	Version      int
	OldLayoutGo  string // Go source literal for []gorp.FieldLayout
	NewLayoutGo  string
}

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	// Group entries by output path.
	outputEntries := make(map[string][]migrationEntry)
	var outputOrder []string

	for _, entry := range req.Resolutions.StructTypes() {
		if !hasMigrateAnnotation(entry) {
			continue
		}
		form, ok := entry.Form.(resolution.StructForm)
		if !ok || !form.HasKeyDomain {
			continue
		}
		goPath := output.GetPath(entry, "go")
		if goPath == "" {
			continue
		}

		me := migrationEntry{
			GoName:  getGoName(entry),
			KeyName: findKeyFieldGoName(entry, req.Resolutions),
			GoPath:  goPath,
		}

		// Check for schema change if we have a previous snapshot.
		if req.OldResolutions != nil {
			sc, err := detectSchemaChange(entry, req)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to detect schema change for %s", me.GoName)
			}
			me.SchemaChange = sc
		}

		if _, exists := outputEntries[goPath]; !exists {
			outputOrder = append(outputOrder, goPath)
		}
		outputEntries[goPath] = append(outputEntries[goPath], me)
	}

	// Generate migrate.gen.go for each output path.
	for _, goPath := range outputOrder {
		entries := outputEntries[goPath]
		pkg := naming.DerivePackageName(goPath)

		needsGorp := true
		needsBinary := true
		for _, e := range entries {
			if e.SchemaChange != nil {
				needsGorp = true
			}
			_ = needsBinary
		}

		content, err := renderMigrateFile(pkg, entries, needsGorp, needsBinary)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate migrate.gen.go for %s", goPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    goPath + "/migrate.gen.go",
			Content: content,
		})

		// Generate transform templates for schema changes.
		for _, e := range entries {
			if e.SchemaChange == nil {
				continue
			}
			templateFile := fmt.Sprintf("%s/v%d_migrate.go", goPath, e.SchemaChange.Version)
			templateFullPath := filepath.Join(req.RepoRoot, templateFile)
			if _, statErr := os.Stat(templateFullPath); os.IsNotExist(statErr) {
				tc, err := renderTransformTemplate(pkg, e.GoName, e.SchemaChange.Version)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to generate transform template for %s", e.GoName)
				}
				resp.Files = append(resp.Files, plugin.File{Path: templateFile, Content: tc})
			}
		}
	}

	return resp, nil
}

func detectSchemaChange(
	newType resolution.Type,
	req *plugin.Request,
) (*schemaChange, error) {
	oldType, found := req.OldResolutions.Get(newType.QualifiedName)
	if !found {
		return nil, nil
	}

	oldFields := resolution.UnifiedFields(oldType, req.OldResolutions)
	newFields := resolution.UnifiedFields(newType, req.Resolutions)

	if fieldsEqual(oldFields, newFields) {
		return nil, nil
	}

	oldLayout, err := BuildLayout(oldType, req.OldResolutions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build old layout")
	}
	newLayout, err := BuildLayout(newType, req.Resolutions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build new layout")
	}

	return &schemaChange{
		Version:     req.SnapshotVersion + 1,
		OldLayoutGo: layoutToGo(oldLayout, "\t\t\t"),
		NewLayoutGo: layoutToGo(newLayout, "\t\t\t"),
	}, nil
}

func fieldsEqual(a, b []resolution.Field) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name || a[i].Type.Name != b[i].Type.Name ||
			a[i].IsOptional != b[i].IsOptional || a[i].IsHardOptional != b[i].IsHardOptional {
			return false
		}
	}
	return true
}

func hasMigrateAnnotation(typ resolution.Type) bool {
	domain, ok := typ.Domains["go"]
	if !ok {
		return false
	}
	for _, expr := range domain.Expressions {
		if expr.Name == "migrate" {
			return true
		}
	}
	return false
}

func getGoName(t resolution.Type) string {
	if domain, ok := t.Domains["go"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return t.Name
}

func findKeyFieldGoName(typ resolution.Type, table *resolution.Table) string {
	fields := resolution.UnifiedFields(typ, table)
	for _, f := range fields {
		if _, hasKey := f.Domains["key"]; hasKey {
			return naming.GetFieldName(f)
		}
	}
	return "Key"
}

// --- Migration Registration Template ---

var migrateTmpl = template.Must(template.New("migrate").Parse(
	`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import (
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/gorp"
)
{{range .Entries}}
func {{.GoName}}Migrations(codec binary.Codec) []gorp.Migration {
	return []gorp.Migration{
		gorp.NewCodecTransition[Key, {{.GoName}}]("msgpack_to_binary", codec),
{{- range .SchemaChanges}}
		gorp.NewSchemaResolution("v{{.Version}}_schema_change",
			{{.OldLayoutGo}},
			{{.NewLayoutGo}},
		),
{{- end}}
	}
}
{{end}}`))

type migrateTemplateData struct {
	Package string
	Entries []migrateTemplateEntry
}

type migrateTemplateEntry struct {
	GoName        string
	SchemaChanges []schemaChange
}

func renderMigrateFile(pkg string, entries []migrationEntry, _, _ bool) ([]byte, error) {
	var tmplEntries []migrateTemplateEntry
	for _, e := range entries {
		te := migrateTemplateEntry{GoName: e.GoName}
		if e.SchemaChange != nil {
			te.SchemaChanges = []schemaChange{*e.SchemaChange}
		}
		tmplEntries = append(tmplEntries, te)
	}
	var buf bytes.Buffer
	if err := migrateTmpl.Execute(&buf, migrateTemplateData{
		Package: pkg,
		Entries: tmplEntries,
	}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// --- Developer Transform Template ---

var transformTmpl = template.Must(template.New("transform").Parse(
	`// Generated by oracle as a template. Edit this file.

package {{.Package}}

import "context"

// Migrate{{.GoName}}V{{.Version}} sets defaults for new fields after schema resolution.
// The schema resolver has already transformed the binary layout. This function
// operates on the current Go type with new fields at zero values.
func Migrate{{.GoName}}V{{.Version}}(_ context.Context, old {{.GoName}}) ({{.GoName}}, error) {
	// TODO: Set defaults for new/changed fields.
	return old, nil
}
`))

type transformTemplateData struct {
	Package string
	GoName  string
	Version int
}

func renderTransformTemplate(pkg, goName string, version int) ([]byte, error) {
	var buf bytes.Buffer
	if err := transformTmpl.Execute(&buf, transformTemplateData{
		Package: pkg,
		GoName:  goName,
		Version: version,
	}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
