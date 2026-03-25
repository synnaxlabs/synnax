// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package migrate provides an Oracle plugin that generates migration files.
// For each gorp entry type with @go migrate, it generates:
//   - migrate.gen.go: migration chain registration (codec transition + schema migrations)
//   - v{N}_types.gen.go: frozen Go types for old schema versions
//   - v{N}_codec.gen.go: frozen codecs for old schema versions (reuses marshal plugin)
//   - v{N}_migrate.go: developer transform template
package migrate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	gomarshal "github.com/synnaxlabs/oracle/plugin/go/marshal"
	goprimitives "github.com/synnaxlabs/oracle/plugin/go/primitives"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

var primitiveMapper = goprimitives.Mapper()

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
	Version     int
	VersionName string           // e.g., "V2"
	OldType     resolution.Type  // old type from snapshot
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
		me := migrationEntry{
			GoName: getGoName(entry),
			GoPath: goPath,
		}
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

		// Generate migrate.gen.go (migration chain registration).
		regContent, err := renderMigrateFile(pkg, entries)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate migrate.gen.go for %s", goPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    goPath + "/migrate.gen.go",
			Content: regContent,
		})

		// For each schema change, generate frozen types + codecs + transform template.
		for _, e := range entries {
			if e.SchemaChange == nil {
				continue
			}
			sc := e.SchemaChange
			versionedName := e.GoName + sc.VersionName // e.g., "WorkspaceV2"

			// Generate frozen type.
			typeContent, err := renderFrozenType(pkg, versionedName, sc.OldType, req)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate frozen type for %s", versionedName)
			}
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/v%d_types.gen.go", goPath, sc.Version),
				Content: typeContent,
			})

			// Generate frozen codec using the marshal plugin's code generation.
			codecContent, err := gomarshal.GenerateCodecFile(
				pkg, goPath,
				[]gomarshal.CodecEntry{{GoName: versionedName, Type: sc.OldType}},
				req.OldResolutions,
				req.RepoRoot,
			)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate frozen codec for %s", versionedName)
			}
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/v%d_codec.gen.go", goPath, sc.Version),
				Content: codecContent,
			})

			// Generate developer transform template.
			templateFile := fmt.Sprintf("%s/v%d_migrate.go", goPath, sc.Version)
			templateFullPath := filepath.Join(req.RepoRoot, templateFile)
			if _, statErr := os.Stat(templateFullPath); os.IsNotExist(statErr) {
				tc, err := renderTransformTemplate(pkg, e.GoName, versionedName, sc.Version)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to generate transform template")
				}
				resp.Files = append(resp.Files, plugin.File{
					Path:    templateFile,
					Content: tc,
				})
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

	// Deep comparison using layout trees.
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

	version := req.SnapshotVersion + 1
	return &schemaChange{
		Version:     version,
		VersionName: fmt.Sprintf("V%d", version),
		OldType:     oldType,
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

func findKeyFieldGoName(typ resolution.Type, table *resolution.Table) string {
	fields := resolution.UnifiedFields(typ, table)
	for _, f := range fields {
		if _, hasKey := f.Domains["key"]; hasKey {
			return naming.GetFieldName(f)
		}
	}
	return "Key"
}

// --- Frozen Type Template ---

func renderFrozenType(
	pkg, versionedName string,
	typ resolution.Type,
	req *plugin.Request,
) ([]byte, error) {
	goPath := output.GetPath(typ, "go")
	imps := imports.NewManager()
	ctx := &resolver.Context{
		Table:                         req.OldResolutions,
		OutputPath:                    goPath,
		Namespace:                     typ.Namespace,
		RepoRoot:                      req.RepoRoot,
		DomainName:                    "go",
		SubstituteDefaultedTypeParams: true,
	}
	r := &resolver.Resolver{
		Formatter:       gotypes.GoFormatter(),
		ImportResolver:  &gotypes.GoImportResolver{RepoRoot: req.RepoRoot, CurrentPackage: pkg},
		ImportAdder:     imps,
		PrimitiveMapper: primitiveMapper,
	}

	fields := resolution.UnifiedFields(typ, req.OldResolutions)
	var keyGoType, keyFieldName string
	var frozenFields []frozenField
	for _, f := range fields {
		fGoName := naming.GetFieldName(f)
		fGoType := r.ResolveTypeRef(f.Type, ctx)
		if f.IsHardOptional && !strings.HasPrefix(fGoType, "[]") &&
			!strings.HasPrefix(fGoType, "map[") &&
			!strings.HasPrefix(fGoType, "binary.MsgpackEncodedJSON") {
			fGoType = "*" + fGoType
		}
		tags := fmt.Sprintf("`json:\"%s\" msgpack:\"%s\"`", lo.SnakeCase(f.Name), lo.SnakeCase(f.Name))
		frozenFields = append(frozenFields, frozenField{
			GoName: fGoName,
			GoType: fGoType,
			Tags:   tags,
		})
		if _, hasKey := f.Domains["key"]; hasKey {
			keyGoType = fGoType
			keyFieldName = fGoName
		}
	}
	if keyGoType == "" {
		keyGoType = "string"
		if len(frozenFields) > 0 {
			keyFieldName = frozenFields[0].GoName
		}
	}

	data := frozenTypeData{
		Package:         pkg,
		GoName:          versionedName,
		KeyGoType:       keyGoType,
		KeyFieldName:    keyFieldName,
		Fields:          frozenFields,
		ExternalImports: imps.ExternalImports(),
		InternalImports: imps.InternalImports(),
	}
	var buf bytes.Buffer
	if err := frozenTypeTmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type frozenField struct {
	GoName string
	GoType string
	Tags   string
}

type frozenTypeData struct {
	Package         string
	GoName          string
	KeyGoType       string
	KeyFieldName    string
	Fields          []frozenField
	ExternalImports []string
	InternalImports []imports.InternalImportData
}

var frozenTypeTmpl = template.Must(template.New("frozenType").Parse(
	`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import (
{{- range .ExternalImports}}
	"{{.}}"
{{- end}}
{{- range .InternalImports}}
{{- if .NeedsAlias}}
	{{.Alias}} "{{.Path}}"
{{- else}}
	"{{.Path}}"
{{- end}}
{{- end}}
)

type {{.GoName}} struct {
{{- range .Fields}}
	{{.GoName}} {{.GoType}} {{.Tags}}
{{- end}}
}

func (e {{.GoName}}) GorpKey() {{.KeyGoType}} { return e.{{.KeyFieldName}} }
func (e {{.GoName}}) SetOptions() []any { return nil }
`))

// --- Migration Registration Template ---

var migrateTmpl = template.Must(template.New("migrate").Parse(
	`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import (
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/gorp"
)
{{range $entry := .Entries}}
func {{$entry.GoName}}Migrations(codec binary.Codec) []gorp.Migration {
	return []gorp.Migration{
		gorp.NewCodecTransition[Key, {{$entry.GoName}}]("msgpack_to_binary", codec),
{{- range $entry.SchemaChanges}}
		gorp.NewTypedMigration[{{.VersionedName}}, {{$entry.GoName}}](
			"v{{.Version}}_schema_migration",
			{{.VersionedName}}Codec,
			codec,
			Migrate{{$entry.GoName}}V{{.Version}},
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
	SchemaChanges []migrateSchemaChange
}

type migrateSchemaChange struct {
	Version       int
	VersionedName string
}

func renderMigrateFile(pkg string, entries []migrationEntry) ([]byte, error) {
	var tmplEntries []migrateTemplateEntry
	for _, e := range entries {
		te := migrateTemplateEntry{GoName: e.GoName}
		if e.SchemaChange != nil {
			te.SchemaChanges = []migrateSchemaChange{{
				Version:       e.SchemaChange.Version,
				VersionedName: e.GoName + e.SchemaChange.VersionName,
			}}
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
//
// This function transforms a frozen {{.VersionedName}} (old schema) into the
// current {{.GoName}}. Copy fields from old and set defaults for new fields.
//
// If zero defaults are acceptable, replace the panic with:
//   return {{.GoName}}{Key: old.Key, Name: old.Name, ...}, nil

package {{.Package}}

import "context"

func Migrate{{.GoName}}V{{.Version}}(_ context.Context, old {{.VersionedName}}) ({{.GoName}}, error) {
	panic("migration Migrate{{.GoName}}V{{.Version}} not implemented - edit this function")
}
`))

type transformTemplateData struct {
	Package       string
	GoName        string
	VersionedName string
	Version       int
}

func renderTransformTemplate(pkg, goName, versionedName string, version int) ([]byte, error) {
	var buf bytes.Buffer
	if err := transformTmpl.Execute(&buf, transformTemplateData{
		Package:       pkg,
		GoName:        goName,
		VersionedName: versionedName,
		Version:       version,
	}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
