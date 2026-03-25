// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package migrate provides an Oracle plugin that generates gorp migration registration
// files. For the initial generation (codec transition), it produces a migrate.gen.go
// file in the parent service package containing a function that returns the codec
// transition migration. No sub-package or frozen types are needed for codec transitions.
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
	goprimitives "github.com/synnaxlabs/oracle/plugin/go/primitives"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
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

type migrateEntry struct {
	GoName  string
	KeyName string
}

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	outputEntries := make(map[string][]migrateEntry)
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
		goName := getGoName(entry)
		keyName := findKeyFieldGoName(entry, req.Resolutions)
		if _, exists := outputEntries[goPath]; !exists {
			outputOrder = append(outputOrder, goPath)
		}
		outputEntries[goPath] = append(outputEntries[goPath], migrateEntry{
			GoName:  goName,
			KeyName: keyName,
		})
	}

	for _, goPath := range outputOrder {
		entries := outputEntries[goPath]
		pkg := naming.DerivePackageName(goPath)
		content, err := renderMigrateFile(pkg, entries)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate migrate for %s", goPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    goPath + "/migrate.gen.go",
			Content: content,
		})
	}

	// If we have old resolutions, check for schema changes and generate
	// frozen types + migration templates for changed types.
	if req.OldResolutions != nil {
		changedFiles, err := p.generateSchemaChangeMigrations(req, outputEntries)
		if err != nil {
			return nil, err
		}
		resp.Files = append(resp.Files, changedFiles...)
	}

	return resp, nil
}

func (p *Plugin) generateSchemaChangeMigrations(
	req *plugin.Request,
	outputEntries map[string][]migrateEntry,
) ([]plugin.File, error) {
	var files []plugin.File

	for _, newType := range req.Resolutions.StructTypes() {
		if !hasMigrateAnnotation(newType) {
			continue
		}
		form, ok := newType.Form.(resolution.StructForm)
		if !ok || !form.HasKeyDomain {
			continue
		}

		// Find the old version of this type.
		oldType, found := req.OldResolutions.Get(newType.QualifiedName)
		if !found {
			continue // New type, no migration needed.
		}

		// Compare fields.
		oldFields := resolution.UnifiedFields(oldType, req.OldResolutions)
		newFields := resolution.UnifiedFields(newType, req.Resolutions)

		if fieldsMatch(oldFields, newFields) {
			continue // No change.
		}

		goPath := output.GetPath(newType, "go")
		if goPath == "" {
			continue
		}
		goName := getGoName(newType)

		// Generate frozen type for old version in migrations sub-package.
		frozenFiles, err := p.generateFrozenType(
			oldType, goPath, goName, req,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate frozen type for %s", goName)
		}
		files = append(files, frozenFiles...)

		version := req.SnapshotVersion + 1

		// Generate developer transform template in the PARENT package
		// so it can reference both frozen types (via import) and current types.
		parentPkg := naming.DerivePackageName(goPath)
		templateFile := fmt.Sprintf("%s/v%d_migrate.go", goPath, version)
		migrationsImport := gomod.ResolveImportPath(goPath+"/migrations", req.RepoRoot, "github.com/synnaxlabs/synnax/")
		templateContent, err := renderTransformTemplate(parentPkg, goName, version, migrationsImport)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate transform template for %s", goName)
		}
		// Check if file already exists (don't overwrite developer code).
		templateFullPath := filepath.Join(req.RepoRoot, templateFile)
		if _, statErr := os.Stat(templateFullPath); os.IsNotExist(statErr) {
			files = append(files, plugin.File{
				Path:    templateFile,
				Content: templateContent,
			})
		}
	}

	return files, nil
}

func fieldsMatch(a, b []resolution.Field) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name {
			return false
		}
		if a[i].Type.Name != b[i].Type.Name {
			return false
		}
		if a[i].IsOptional != b[i].IsOptional {
			return false
		}
		if a[i].IsHardOptional != b[i].IsHardOptional {
			return false
		}
	}
	return true
}

func (p *Plugin) generateFrozenType(
	oldType resolution.Type,
	goPath string,
	goName string,
	req *plugin.Request,
) ([]plugin.File, error) {
	version := req.SnapshotVersion + 1
	migrationsPath := goPath + "/migrations"
	pkg := naming.DerivePackageName(migrationsPath)

	imps := imports.NewManager()
	ctx := &resolver.Context{
		Table:                         req.OldResolutions,
		OutputPath:                    migrationsPath,
		Namespace:                     oldType.Namespace,
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

	parentPkg := naming.DerivePackageName(goPath)

	fields := resolution.UnifiedFields(oldType, req.OldResolutions)
	var keyGoType, keyFieldName string
	var frozenFields []frozenField
	for _, f := range fields {
		fGoName := naming.GetFieldName(f)
		fGoType := r.ResolveTypeRef(f.Type, ctx)
		// If the resolved type references the parent package (circular dep),
		// unwrap the alias/distinct type to the underlying primitive.
		if strings.HasPrefix(fGoType, parentPkg+".") {
			fGoType = unwrapToBase(f.Type, req.OldResolutions)
		}
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
		keyFieldName = frozenFields[0].GoName
	}

	// Filter out imports that reference the parent package (circular dep).
	parentImportPath := gomod.ResolveImportPath(goPath, req.RepoRoot, "github.com/synnaxlabs/synnax/")
	var filteredInternal []imports.InternalImportData
	for _, imp := range imps.InternalImports() {
		if imp.Path != parentImportPath {
			filteredInternal = append(filteredInternal, imp)
		}
	}

	data := frozenTypeData{
		Package:         "migrations",
		Version:         fmt.Sprintf("V%d", version),
		GoName:          goName,
		KeyGoType:       keyGoType,
		KeyFieldName:    keyFieldName,
		Fields:          frozenFields,
		ExternalImports: imps.ExternalImports(),
		InternalImports: filteredInternal,
	}

	var buf bytes.Buffer
	if err := frozenTypeTmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	fileName := fmt.Sprintf("v%d.gen.go", version)
	return []plugin.File{{
		Path:    migrationsPath + "/" + fileName,
		Content: buf.Bytes(),
	}}, nil
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
	}
}
{{end}}`))

type migrateTemplateData struct {
	Package string
	Entries []migrateEntry
}

func renderMigrateFile(pkg string, entries []migrateEntry) ([]byte, error) {
	var buf bytes.Buffer
	if err := migrateTmpl.Execute(&buf, migrateTemplateData{
		Package: pkg,
		Entries: entries,
	}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// unwrapToBase resolves a type reference to its underlying primitive Go type,
// following alias and distinct type chains. Used when the resolved type would
// reference the parent package (creating a circular dependency).
func unwrapToBase(ref resolution.TypeRef, table *resolution.Table) string {
	resolved, ok := ref.Resolve(table)
	if !ok {
		return "any"
	}
	return unwrapResolved(resolved, table)
}

func unwrapResolved(resolved resolution.Type, table *resolution.Table) string {
	switch form := resolved.Form.(type) {
	case resolution.PrimitiveForm:
		m := primitiveMapper.Map(form.Name)
		if m.TargetType == "" {
			return "any"
		}
		if form.Name == "record" {
			return "binary.MsgpackEncodedJSON"
		}
		return m.TargetType
	case resolution.AliasForm:
		inner, ok := form.Target.Resolve(table)
		if !ok {
			return "any"
		}
		return unwrapResolved(inner, table)
	case resolution.DistinctForm:
		inner, ok := form.Base.Resolve(table)
		if !ok {
			return "any"
		}
		return unwrapResolved(inner, table)
	default:
		return "any"
	}
}

// --- Frozen Type for Schema Change Migrations ---

type frozenField struct {
	GoName string
	GoType string
	Tags   string
}

type frozenTypeData struct {
	Package         string
	Version         string
	GoName          string
	KeyGoType       string
	KeyFieldName    string
	Fields          []frozenField
	ExternalImports []string
	InternalImports []imports.InternalImportData
}

// --- Developer Transform Template ---

var transformTmpl = template.Must(template.New("transform").Parse(
	`// Generated by oracle as a template. Edit this file.

package {{.Package}}

import (
	"context"

	"{{.MigrationsImport}}"
)

// Migrate{{.GoName}}V{{.Version}} transforms a frozen {{.GoName}}V{{.Version}} into the
// current {{.GoName}}. Set defaults for new/changed fields.
func Migrate{{.GoName}}V{{.Version}}(ctx context.Context, old migrations.{{.GoName}}V{{.Version}}) ({{.GoName}}, error) {
	return {{.GoName}}{
		// TODO: Copy fields from old and set defaults for new fields.
	}, nil
}
`))

type transformTemplateData struct {
	Package          string
	GoName           string
	Version          int
	MigrationsImport string
}

func renderTransformTemplate(pkg, goName string, version int, migrationsImport string) ([]byte, error) {
	var buf bytes.Buffer
	if err := transformTmpl.Execute(&buf, transformTemplateData{
		Package:          pkg,
		GoName:           goName,
		Version:          version,
		MigrationsImport: migrationsImport,
	}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var frozenTypeTmpl = template.Must(template.New("frozen").Parse(
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

type {{.GoName}}{{.Version}} struct {
{{- range .Fields}}
	{{.GoName}} {{.GoType}} {{.Tags}}
{{- end}}
}

type {{.GoName}}{{.Version}}Key = {{.KeyGoType}}

func (e {{.GoName}}{{.Version}}) GorpKey() {{.GoName}}{{.Version}}Key { return e.{{.KeyFieldName}} }
func (e {{.GoName}}{{.Version}}) SetOptions() []any { return nil }
`))
