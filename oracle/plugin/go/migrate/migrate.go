// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package migrate provides an Oracle plugin that generates migration files.
// For each schema change, it generates frozen types + codecs in a versioned
// sub-package (migrations/vN/) where types keep their original names. The
// package boundary provides namespacing, eliminating the need for renaming.
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
	gomarshal "github.com/synnaxlabs/oracle/plugin/go/marshal"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
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

type migrationEntry struct {
	GoName       string
	GoPath       string
	SchemaChange *schemaChange
}

type schemaChange struct {
	Version int              // major*1000+minor from core VERSION (e.g., 53 for 0.53.x)
	OldType resolution.Type
}

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
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
		me := migrationEntry{GoName: getGoName(entry), GoPath: goPath}
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

	for _, goPath := range outputOrder {
		entries := outputEntries[goPath]
		pkg := naming.DerivePackageName(goPath)

		// Generate migrate.gen.go in the parent package.
		regContent, err := renderMigrateFile(pkg, goPath, entries, req.RepoRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate migrate.gen.go for %s", goPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    goPath + "/migrate.gen.go",
			Content: regContent,
		})

		for _, e := range entries {
			if e.SchemaChange == nil {
				continue
			}
			sc := e.SchemaChange
			vDir := fmt.Sprintf("v%d", sc.Version)
			subPkg := fmt.Sprintf("%s/migrations/%s", goPath, vDir)

			// Generate frozen types in the sub-package. Types keep their
			// original names. The package boundary provides namespacing.
			// Use the marshal plugin's GenerateCodecFile which generates
			// codecs with correct local type references.
			codecContent, err := gomarshal.GenerateCodecFile(
				vDir, subPkg,
				[]gomarshal.CodecEntry{{GoName: e.GoName, Type: sc.OldType}},
				req.OldResolutions,
				req.RepoRoot,
				nil, // no name overrides needed
			)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate frozen codec for %s", e.GoName)
			}
			resp.Files = append(resp.Files, plugin.File{
				Path:    subPkg + "/codec.gen.go",
				Content: codecContent,
			})

			// Generate frozen types using the go/types plugin's type generation.
			// For now, generate the entry type's struct definition. The codec
			// helpers reference nested types from their original packages.
			// TODO: Generate all nested types in the sub-package for full isolation.

			// Generate developer transform template in the parent package.
			templateFile := fmt.Sprintf("%s/v%d_migrate.go", goPath, sc.Version)
			templateFullPath := filepath.Join(req.RepoRoot, templateFile)
			if _, statErr := os.Stat(templateFullPath); os.IsNotExist(statErr) {
				migrationsImport := gomod.ResolveImportPath(subPkg, req.RepoRoot, "github.com/synnaxlabs/synnax/")
				tc, err := renderTransformTemplate(pkg, e.GoName, sc.Version, vDir, migrationsImport)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to generate transform template")
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
	oldLayout, err := BuildLayout(oldType, req.OldResolutions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build old layout")
	}
	newLayout, err := BuildLayout(newType, req.Resolutions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build new layout")
	}
	if layoutsDeepEqual(oldLayout, newLayout) {
		return nil, nil
	}
	return &schemaChange{
		Version: req.SnapshotVersion, // frozen types represent this version
		OldType: oldType,
	}, nil
}

func layoutsDeepEqual(a, b []gorp.FieldLayout) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !gorp.LayoutsEqual(a[i], b[i]) {
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

// --- Migration Registration Template ---

var migrateTmpl = template.Must(template.New("migrate").Parse(
	`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import (
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/gorp"
{{- range .VersionImports}}
	{{.Alias}} "{{.Path}}"
{{- end}}
)
{{range $entry := .Entries}}
func {{$entry.GoName}}Migrations(codec binary.Codec) []gorp.Migration {
	return []gorp.Migration{
		gorp.NewCodecTransition[Key, {{$entry.GoName}}]("msgpack_to_binary", codec),
{{- range $entry.SchemaChanges}}
		gorp.NewTypedMigration[{{.ImportAlias}}.{{$entry.GoName}}, {{$entry.GoName}}](
			"v{{.Version}}_schema_migration",
			{{.ImportAlias}}.{{$entry.GoName}}Codec,
			codec,
			Migrate{{$entry.GoName}}V{{.Version}},
		),
{{- end}}
	}
}
{{end}}`))

type migrateTemplateData struct {
	Package        string
	Entries        []migrateTemplateEntry
	VersionImports []versionImport
}

type versionImport struct {
	Alias string
	Path  string
}

type migrateTemplateEntry struct {
	GoName        string
	SchemaChanges []migrateSchemaChange
}

type migrateSchemaChange struct {
	Version     int
	ImportAlias string
}

func renderMigrateFile(pkg, goPath string, entries []migrationEntry, repoRoot string) ([]byte, error) {
	var tmplEntries []migrateTemplateEntry
	importSet := make(map[string]versionImport)
	for _, e := range entries {
		te := migrateTemplateEntry{GoName: e.GoName}
		if e.SchemaChange != nil {
			sc := e.SchemaChange
			vDir := fmt.Sprintf("v%d", sc.Version)
			subPkg := fmt.Sprintf("%s/migrations/%s", goPath, vDir)
			importPath := gomod.ResolveImportPath(subPkg, repoRoot, "github.com/synnaxlabs/synnax/")
			importSet[vDir] = versionImport{Alias: vDir, Path: importPath}
			te.SchemaChanges = []migrateSchemaChange{{
				Version:     sc.Version,
				ImportAlias: vDir,
			}}
		}
		tmplEntries = append(tmplEntries, te)
	}
	var imports []versionImport
	for _, v := range importSet {
		imports = append(imports, v)
	}
	var buf bytes.Buffer
	if err := migrateTmpl.Execute(&buf, migrateTemplateData{
		Package:        pkg,
		Entries:        tmplEntries,
		VersionImports: imports,
	}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// --- Developer Transform Template ---

var transformTmpl = template.Must(template.New("transform").Parse(
	`// Generated by oracle as a template. Edit this file.
//
// This function transforms a frozen {{.GoName}} (from {{.VersionDir}}) into the
// current {{.GoName}}. Copy fields from old and set defaults for new fields.

package {{.Package}}

import (
	"context"

	{{.VersionDir}} "{{.MigrationsImport}}"
)

func Migrate{{.GoName}}V{{.Version}}(_ context.Context, old {{.VersionDir}}.{{.GoName}}) ({{.GoName}}, error) {
	panic("migration Migrate{{.GoName}}V{{.Version}} not implemented - edit this function")
}
`))

type transformTemplateData struct {
	Package          string
	GoName           string
	Version          int
	VersionDir       string
	MigrationsImport string
}

func renderTransformTemplate(pkg, goName string, version int, vDir, migrationsImport string) ([]byte, error) {
	var buf bytes.Buffer
	if err := transformTmpl.Execute(&buf, transformTemplateData{
		Package:          pkg,
		GoName:           goName,
		Version:          version,
		VersionDir:       vDir,
		MigrationsImport: migrationsImport,
	}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// filterImports removes the given paths from a list of import strings.
func filterImports(imports []string, exclude ...string) []string {
	excludeSet := make(map[string]bool, len(exclude))
	for _, e := range exclude {
		excludeSet[e] = true
	}
	var result []string
	for _, imp := range imports {
		if !excludeSet[imp] {
			result = append(result, imp)
		}
	}
	return result
}
