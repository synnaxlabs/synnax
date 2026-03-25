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
	"github.com/synnaxlabs/oracle/plugin/gomod"
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
	Version       int
	VersionName   string           // e.g., "V2"
	VersionedName string           // e.g., "ArcArcV3" (namespace-prefixed)
	OldType       resolution.Type  // old type from snapshot
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

			// Build name overrides: map every Oracle-defined type reachable from
			// the entry to a versioned local name. This flattens nested types
			// into the parent package so the frozen codec is self-contained.
			overrides := buildNameOverrides(sc.OldType, req.OldResolutions, sc.VersionName)

			// Get the entry's versioned name from the overrides (uses namespace-prefixed naming).
			versionedName := overrides[sc.OldType.QualifiedName]
			if versionedName == "" {
				versionedName = e.GoName + sc.VersionName
			}

			// Generate frozen types for the entry AND all nested Oracle-defined types.
			typeContent, err := renderFrozenTypes(pkg, sc.OldType, req, overrides)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate frozen types for %s", versionedName)
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
				overrides,
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
	versionSuffix := fmt.Sprintf("V%d", version)
	// Compute the namespace-prefixed versioned name for the entry type.
	parts := strings.Split(oldType.QualifiedName, ".")
	var entryVersionedName string
	for _, p := range parts {
		entryVersionedName += naming.ToPascalCase(p)
	}
	entryVersionedName += versionSuffix

	return &schemaChange{
		Version:       version,
		VersionName:   versionSuffix,
		VersionedName: entryVersionedName,
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

// buildNameOverrides walks the entry type's dependency tree and creates a
// name override map that maps every Oracle-defined struct type to a versioned
// local name. This flattens nested types into the parent package.
func buildNameOverrides(
	entryType resolution.Type,
	table *resolution.Table,
	versionSuffix string,
) gomarshal.NameOverrides {
	overrides := make(gomarshal.NameOverrides)
	visited := make(map[string]bool)
	walkTypeTree(entryType, table, versionSuffix, overrides, visited)
	// Second pass: propagate overrides to alias/distinct types that resolve
	// to overridden structs. Must run after all structs are discovered.
	for _, typ := range table.Types {
		if _, alreadyOverridden := overrides[typ.QualifiedName]; alreadyOverridden {
			continue
		}
		switch form := typ.Form.(type) {
		case resolution.AliasForm:
			target, ok := form.Target.Resolve(table)
			if ok {
				if override, exists := overrides[target.QualifiedName]; exists {
					overrides[typ.QualifiedName] = override
				}
			}
		case resolution.DistinctForm:
			target, ok := form.Base.Resolve(table)
			if ok {
				if override, exists := overrides[target.QualifiedName]; exists {
					overrides[typ.QualifiedName] = override
				}
			}
		}
	}
	return overrides
}

func walkTypeTree(
	typ resolution.Type,
	table *resolution.Table,
	suffix string,
	overrides gomarshal.NameOverrides,
	visited map[string]bool,
) {
	if visited[typ.QualifiedName] {
		return
	}
	visited[typ.QualifiedName] = true

	// Override struct types with versioned names.
	if _, ok := typ.Form.(resolution.StructForm); ok {
		parts := strings.Split(typ.QualifiedName, ".")
		var name string
		for _, p := range parts {
			name += naming.ToPascalCase(p)
		}
		overrides[typ.QualifiedName] = name + suffix
	}
	// Alias/distinct override propagation is handled in the second pass
	// of buildNameOverrides, after all structs are discovered.

	// Walk fields to find nested Oracle-defined types.
	fields := resolution.UnifiedFields(typ, table)
	for _, f := range fields {
		walkFieldType(f.Type, table, suffix, overrides, visited)
	}
}

func walkFieldType(
	ref resolution.TypeRef,
	table *resolution.Table,
	suffix string,
	overrides gomarshal.NameOverrides,
	visited map[string]bool,
) {
	resolved, ok := ref.Resolve(table)
	if !ok {
		return
	}
	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		walkTypeTree(resolved, table, suffix, overrides, visited)
	case resolution.AliasForm:
		walkFieldType(form.Target, table, suffix, overrides, visited)
	case resolution.DistinctForm:
		walkFieldType(form.Base, table, suffix, overrides, visited)
	case resolution.BuiltinGenericForm:
		for _, arg := range ref.TypeArgs {
			walkFieldType(arg, table, suffix, overrides, visited)
		}
	}
	// Also walk ALL type args on the ref, regardless of resolved form.
	// This catches cases like aliases to Array<Function> where the type
	// args are on the original ref, not the resolved form.
	for _, arg := range ref.TypeArgs {
		walkFieldType(arg, table, suffix, overrides, visited)
	}
}

// resolveFieldTypeWithOverrides resolves a field's Go type, replacing any
// Oracle-defined struct types with their versioned names from the overrides map.
// Follows through aliases and generic types to find nested structs.
func resolveFieldTypeWithOverrides(
	ref resolution.TypeRef,
	table *resolution.Table,
	overrides gomarshal.NameOverrides,
	r *resolver.Resolver,
	ctx *resolver.Context,
) string {
	// Try to resolve the type. TypeRefs may have unqualified names, so
	// try both the direct lookup and a namespace-qualified lookup.
	resolved, ok := ref.Resolve(table)
	if !ok {
		// Try qualified with namespace from context.
		qualifiedRef := resolution.TypeRef{
			Name:      ctx.Namespace + "." + ref.Name,
			TypeArgs:  ref.TypeArgs,
			ArraySize: ref.ArraySize,
		}
		resolved, ok = qualifiedRef.Resolve(table)
		if !ok {
			return r.ResolveTypeRef(ref, ctx)
		}
	}
	// Direct struct match.
	if override, ok := overrides[resolved.QualifiedName]; ok {
		if ref.ArraySize != nil {
			return "[]" + override
		}
		return override
	}
	// Unwrap aliases to find arrays/maps of overridden types.
	result := resolveTypeWithOverrides(resolved, ref, table, overrides, r, ctx)
	if ref.ArraySize != nil && !strings.HasPrefix(result, "[]") {
		return "[]" + result
	}
	return result
}

func resolveTypeWithOverrides(
	resolved resolution.Type,
	ref resolution.TypeRef,
	table *resolution.Table,
	overrides gomarshal.NameOverrides,
	r *resolver.Resolver,
	ctx *resolver.Context,
) string {
	var targetRef resolution.TypeRef
	switch form := resolved.Form.(type) {
	case resolution.AliasForm:
		targetRef = form.Target
	case resolution.DistinctForm:
		targetRef = form.Base
	case resolution.BuiltinGenericForm:
		if form.Name == "Array" && len(ref.TypeArgs) > 0 {
			elemType, ok := ref.TypeArgs[0].Resolve(table)
			if ok {
				if override, ok := overrides[elemType.QualifiedName]; ok {
					return "[]" + override
				}
			}
		}
		return r.ResolveTypeRef(ref, ctx)
	default:
		return r.ResolveTypeRef(ref, ctx)
	}

	// Resolve the alias/distinct target.
	target, ok := targetRef.Resolve(table)
	if !ok {
		return r.ResolveTypeRef(ref, ctx)
	}
	isArray := targetRef.ArraySize != nil

	if override, ok := overrides[target.QualifiedName]; ok {
		if isArray {
			return "[]" + override
		}
		return override
	}

	// Check if target is Array/Map of overridden types.
	if bg, ok := target.Form.(resolution.BuiltinGenericForm); ok {
		if bg.Name == "Array" && len(targetRef.TypeArgs) > 0 {
			elemType, ok := targetRef.TypeArgs[0].Resolve(table)
			if ok {
				if override, ok := overrides[elemType.QualifiedName]; ok {
					return "[]" + override
				}
			}
		}
		if bg.Name == "Map" && len(targetRef.TypeArgs) >= 2 {
			keyType := r.ResolveTypeRef(targetRef.TypeArgs[0], ctx)
			valType, ok := targetRef.TypeArgs[1].Resolve(table)
			if ok {
				if override, ok := overrides[valType.QualifiedName]; ok {
					return "map[" + keyType + "]" + override
				}
			}
		}
	}

	// Recurse.
	inner := resolveTypeWithOverrides(target, targetRef, table, overrides, r, ctx)
	if isArray && !strings.HasPrefix(inner, "[]") {
		return "[]" + inner
	}
	return inner
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

// renderFrozenTypes generates frozen Go type definitions for the entry type
// AND all Oracle-defined nested types reachable from it. Each type gets a
// versioned name from the overrides map.
func renderFrozenTypes(
	pkg string,
	entryType resolution.Type,
	req *plugin.Request,
	overrides gomarshal.NameOverrides,
) ([]byte, error) {
	// Collect all types that need frozen definitions.
	var allTypes []frozenTypeData
	imps := imports.NewManager()
	goPath := output.GetPath(entryType, "go")
	ctx := &resolver.Context{
		Table:                         req.OldResolutions,
		OutputPath:                    goPath,
		Namespace:                     entryType.Namespace,
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

	visited := make(map[string]bool)
	collectFrozenTypes(entryType, req.OldResolutions, overrides, r, ctx, &allTypes, visited)

	data := frozenTypesFileData{
		Package:         pkg,
		Types:           allTypes,
		ExternalImports: imps.ExternalImports(),
		InternalImports: filterParentImports(imps.InternalImports(), goPath, req.RepoRoot),
	}
	var buf bytes.Buffer
	if err := frozenTypesFileTmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func collectFrozenTypes(
	typ resolution.Type,
	table *resolution.Table,
	overrides gomarshal.NameOverrides,
	r *resolver.Resolver,
	baseCtx *resolver.Context,
	result *[]frozenTypeData,
	visited map[string]bool,
) {
	if visited[typ.QualifiedName] {
		return
	}
	visited[typ.QualifiedName] = true

	versionedName, isOverridden := overrides[typ.QualifiedName]
	if !isOverridden {
		return // not an Oracle-defined struct, skip
	}

	// Create a context for this type's namespace so field TypeRefs resolve correctly.
	ctx := *baseCtx
	ctx.Namespace = typ.Namespace

	fields := resolution.UnifiedFields(typ, table)
	var keyGoType, keyFieldName string
	var frozenFields []frozenField
	for _, f := range fields {
		fGoName := naming.GetFieldName(f)
		fGoType := resolveFieldTypeWithOverrides(f.Type, table, overrides, r, &ctx)
		if f.IsHardOptional && !strings.HasPrefix(fGoType, "[]") &&
			!strings.HasPrefix(fGoType, "map[") &&
			!strings.HasPrefix(fGoType, "binary.MsgpackEncodedJSON") {
			fGoType = "*" + fGoType
		}
		tags := fmt.Sprintf("`json:\"%s\" msgpack:\"%s\"`", lo.SnakeCase(f.Name), lo.SnakeCase(f.Name))
		frozenFields = append(frozenFields, frozenField{GoName: fGoName, GoType: fGoType, Tags: tags})
		if _, hasKey := f.Domains["key"]; hasKey {
			keyGoType = fGoType
			keyFieldName = fGoName
		}
	}
	if keyGoType == "" {
		keyGoType = "any"
		if len(frozenFields) > 0 {
			keyFieldName = frozenFields[0].GoName
		}
	}
	*result = append(*result, frozenTypeData{
		GoName:       versionedName,
		KeyGoType:    keyGoType,
		KeyFieldName: keyFieldName,
		Fields:       frozenFields,
	})

	// Recurse into nested types, following through aliases and generics.
	for _, f := range fields {
		collectNestedTypes(f.Type, table, overrides, r, baseCtx, result, visited)
	}
}

func collectNestedTypes(
	ref resolution.TypeRef,
	table *resolution.Table,
	overrides gomarshal.NameOverrides,
	r *resolver.Resolver,
	ctx *resolver.Context,
	result *[]frozenTypeData,
	visited map[string]bool,
) {
	resolved, ok := ref.Resolve(table)
	if !ok {
		return
	}
	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		collectFrozenTypes(resolved, table, overrides, r, ctx, result, visited)
	case resolution.AliasForm:
		collectNestedTypes(form.Target, table, overrides, r, ctx, result, visited)
	case resolution.DistinctForm:
		collectNestedTypes(form.Base, table, overrides, r, ctx, result, visited)
	case resolution.BuiltinGenericForm:
		_ = form
		for _, arg := range ref.TypeArgs {
			collectNestedTypes(arg, table, overrides, r, ctx, result, visited)
		}
	}
}

// filterParentImports removes imports that reference the parent package to
// avoid circular dependencies.
func filterParentImports(
	imps []imports.InternalImportData,
	goPath, repoRoot string,
) []imports.InternalImportData {
	// For types in the parent package, the override map handles them as local.
	// But the resolver might have added imports before the override was applied.
	// Filter any that match the parent package.
	parentImport := gomod.ResolveImportPath(goPath, repoRoot, "github.com/synnaxlabs/synnax/")
	var filtered []imports.InternalImportData
	for _, imp := range imps {
		if imp.Path != parentImport {
			filtered = append(filtered, imp)
		}
	}
	return filtered
}

type frozenTypesFileData struct {
	Package         string
	Types           []frozenTypeData
	ExternalImports []string
	InternalImports []imports.InternalImportData
}

var frozenTypesFileTmpl = template.Must(template.New("frozenTypes").Parse(
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
{{range .Types}}
type {{.GoName}} struct {
{{- range .Fields}}
	{{.GoName}} {{.GoType}} {{.Tags}}
{{- end}}
}

func (e {{.GoName}}) GorpKey() {{.KeyGoType}} { return e.{{.KeyFieldName}} }
func (e {{.GoName}}) SetOptions() []any { return nil }
{{end}}`))

// DEPRECATED: renderFrozenType generates a single frozen type. Use renderFrozenTypes instead.
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
				VersionedName: e.SchemaChange.VersionedName,
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
