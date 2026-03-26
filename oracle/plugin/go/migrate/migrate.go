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
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/plugin/gomod"
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

			// Discover all packages in the dependency tree that need mirroring.
			pkgTypes := collectPackageTypes(sc.OldType, req.OldResolutions)

			// Build a path mapping: original output path → mirrored path.
			pathMap := make(map[string]string, len(pkgTypes))
			for origPath := range pkgTypes {
				pathMap[origPath] = origPath + "/migrations/" + vDir
			}

			// Rewrite the resolution table so types appear to belong to
			// their mirrored packages. This makes all existing import
			// resolution work without any special override logic.
			rewrittenTable := rewriteOutputPaths(req.OldResolutions, pathMap)

			// Generate frozen types in each source package's migrations/vN/ directory.
			for origPath, types := range pkgTypes {
				mirroredPath := pathMap[origPath]
				typeContent, err := generateFrozenTypesFile(types, rewrittenTable, mirroredPath, req.RepoRoot)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to generate frozen types for %s", origPath)
				}
				resp.Files = append(resp.Files, plugin.File{
					Path:    mirroredPath + "/types.gen.go",
					Content: typeContent,
				})
			}

			// Generate frozen codec for the entry type in its own migrations/vN/ directory.
			entryMirrorPath := goPath + "/migrations/" + vDir
			codecContent, err := gomarshal.GenerateCodecFile(
				vDir, entryMirrorPath,
				[]gomarshal.CodecEntry{{GoName: e.GoName, Type: sc.OldType}},
				rewrittenTable,
				req.RepoRoot,
			)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate frozen codec for %s", e.GoName)
			}
			resp.Files = append(resp.Files, plugin.File{
				Path:    entryMirrorPath + "/codec.gen.go",
				Content: codecContent,
			})

			// Generate developer transform template in the parent package.
			templateFile := fmt.Sprintf("%s/v%d_migrate.go", goPath, sc.Version)
			templateFullPath := filepath.Join(req.RepoRoot, templateFile)
			if _, statErr := os.Stat(templateFullPath); os.IsNotExist(statErr) {
				entryMirrorImport, _ := resolveImportPath(entryMirrorPath, req.RepoRoot)
				tc, err := renderTransformTemplate(pkg, e.GoName, sc.Version, vDir, entryMirrorImport)
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
	if schemasEqual(oldType, newType, req.OldResolutions, req.Resolutions) {
		return nil, nil
	}
	return &schemaChange{
		Version: req.SnapshotVersion,
		OldType: oldType,
	}, nil
}

// collectPackageTypes walks the entry type's dependency tree and groups all
// Oracle-defined struct types by their @go output path. Returns a map from
// output path → list of types in that package.
func collectPackageTypes(
	entryType resolution.Type,
	table *resolution.Table,
) map[string][]resolution.Type {
	result := make(map[string][]resolution.Type)
	visited := make(map[string]bool)
	collectPkgTypesWalk(entryType, table, result, visited)
	// Expand each collected package: if any type from a package is needed,
	// include ALL types from that package. This handles cases where types are
	// referenced through type parameters, constraints, or defaults that the
	// walker can't follow directly.
	for goPath := range result {
		expanded := make(map[string]bool)
		for _, t := range result[goPath] {
			expanded[t.QualifiedName] = true
		}
		for _, t := range table.TypesWithDomain("go") {
			if expanded[t.QualifiedName] {
				continue
			}
			if output.GetPath(t, "go") == goPath {
				switch t.Form.(type) {
				case resolution.StructForm, resolution.AliasForm, resolution.DistinctForm, resolution.EnumForm:
					result[goPath] = append(result[goPath], t)
				}
			}
		}
	}
	return result
}

func collectPkgTypesWalk(
	typ resolution.Type,
	table *resolution.Table,
	result map[string][]resolution.Type,
	visited map[string]bool,
) {
	if visited[typ.QualifiedName] {
		return
	}
	visited[typ.QualifiedName] = true

	goPath := output.GetPath(typ, "go")
	if goPath != "" {
		switch typ.Form.(type) {
		case resolution.StructForm, resolution.AliasForm, resolution.DistinctForm, resolution.EnumForm:
			result[goPath] = append(result[goPath], typ)
		}
	}

	// Walk parent types from struct extensions (embedding).
	if sf, ok := typ.Form.(resolution.StructForm); ok {
		for _, ext := range sf.Extends {
			walkRefForPkgTypes(ext, table, result, visited)
		}
	}

	// Walk fields to find nested types.
	fields := resolution.UnifiedFields(typ, table)
	for _, f := range fields {
		walkRefForPkgTypes(f.Type, table, result, visited)
	}
}

func walkRefForPkgTypes(
	ref resolution.TypeRef,
	table *resolution.Table,
	result map[string][]resolution.Type,
	visited map[string]bool,
) {
	resolved, ok := ref.Resolve(table)
	if !ok {
		return
	}
	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		collectPkgTypesWalk(resolved, table, result, visited)
	case resolution.AliasForm:
		collectPkgTypesWalk(resolved, table, result, visited)
		walkRefForPkgTypes(form.Target, table, result, visited)
	case resolution.DistinctForm:
		collectPkgTypesWalk(resolved, table, result, visited)
		walkRefForPkgTypes(form.Base, table, result, visited)
	case resolution.EnumForm:
		collectPkgTypesWalk(resolved, table, result, visited)
	case resolution.BuiltinGenericForm:
		// Walk the generic type's fields by looking up the struct definition.
		// BuiltinGenericForm marks the type as generic, but the underlying struct
		// may have fields that reference other types (e.g., Status has a Variant field).
		collectPkgTypesWalk(resolved, table, result, visited)
		for _, arg := range ref.TypeArgs {
			walkRefForPkgTypes(arg, table, result, visited)
		}
	}
}

func resolveImportPath(outputPath, repoRoot string) (string, error) {
	return gomod.ResolveImportPath(outputPath, repoRoot, "github.com/synnaxlabs/synnax/"), nil
}

// rewriteOutputPaths creates a shallow copy of the resolution table where
// each type's @go output path is rewritten according to the path map.
// This allows the standard type/codec generators to produce correct imports
// for mirrored migration packages without any special override logic.
func rewriteOutputPaths(table *resolution.Table, pathMap map[string]string) *resolution.Table {
	clone := &resolution.Table{
		Imports:    table.Imports,
		Namespaces: table.Namespaces,
		Types:      make([]resolution.Type, 0, len(table.Types)),
	}
	for _, typ := range table.Types {
		goPath := output.GetPath(typ, "go")
		mirroredPath, needsRewrite := pathMap[goPath]
		if !needsRewrite {
			clone.Types = append(clone.Types, typ)
			continue
		}
		// Clone the type with rewritten @go output.
		newDomains := make(map[string]resolution.Domain, len(typ.Domains))
		for k, v := range typ.Domains {
			if k == "go" {
				newExprs := make(resolution.Expressions, len(v.Expressions))
				for i, expr := range v.Expressions {
					if expr.Name == "output" && len(expr.Values) > 0 {
						newVals := make([]resolution.ExpressionValue, len(expr.Values))
						copy(newVals, expr.Values)
						newVals[0] = resolution.ExpressionValue{StringValue: mirroredPath}
						newExprs[i] = resolution.Expression{AST: expr.AST, Name: expr.Name, Values: newVals}
					} else {
						newExprs[i] = expr
					}
				}
				newDomains[k] = resolution.Domain{AST: v.AST, Name: v.Name, Expressions: newExprs}
			} else {
				newDomains[k] = v
			}
		}
		rewritten := typ
		rewritten.Domains = newDomains
		clone.Types = append(clone.Types, rewritten)
	}
	return clone
}

// generateFrozenTypesFile generates a Go source file with type definitions
// using the types plugin's full type generation. Import overrides redirect
// cross-package references to the mirrored package structure.
func generateFrozenTypesFile(
	types []resolution.Type,
	table *resolution.Table,
	outputPath, repoRoot string,
) ([]byte, error) {
	var structs, enums, typeDefs []resolution.Type
	for _, typ := range types {
		switch typ.Form.(type) {
		case resolution.StructForm:
			structs = append(structs, typ)
		case resolution.EnumForm:
			enums = append(enums, typ)
		default:
			typeDefs = append(typeDefs, typ)
		}
	}
	return gotypes.GenerateGoFile(outputPath, structs, enums, typeDefs, table, repoRoot)
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

