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
	"regexp"
	"sort"
	"strconv"
	"strings"
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
	"github.com/synnaxlabs/x/set"
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
	GoName           string
	GoPath           string
	SchemaChange     *schemaChange
	ExistingVersions []int
}

type schemaChange struct {
	Version int
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
		mEntry := migrationEntry{GoName: getGoName(entry), GoPath: goPath}
		existingVersions, err := discoverExistingVersions(goPath, req.RepoRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to discover existing migrations for %s", goPath)
		}
		mEntry.ExistingVersions = filterSchemaChangeVersions(goPath, existingVersions, req.RepoRoot)
		if req.OldResolutions != nil {
			change, err := detectSchemaChange(entry, req)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to detect schema change for %s", mEntry.GoName)
			}
			mEntry.SchemaChange = change
		}
		if _, exists := outputEntries[goPath]; !exists {
			outputOrder = append(outputOrder, goPath)
		}
		outputEntries[goPath] = append(outputEntries[goPath], mEntry)
	}

	// Collect all migration entry type names so sub-package codecs can mark
	// them with Adapter: true when they have their own migrations.
	migrateEntryNames := make(set.Set[string])
	for _, entries := range outputEntries {
		for _, e := range entries {
			migrateEntryNames.Add(e.GoName)
		}
	}

	for _, goPath := range outputOrder {
		entries := outputEntries[goPath]
		pkg := naming.DerivePackageName(goPath)

		regContent, err := renderMigrateFile(pkg, goPath, entries, req.RepoRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate migrate.gen.go for %s", goPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    goPath + "/migrate.gen.go",
			Content: regContent,
		})

		for _, entry := range entries {
			if err := p.generateForEntry(resp, entry, goPath, pkg, migrateEntryNames, req); err != nil {
				return nil, err
			}
		}
	}

	return resp, nil
}

func (p *Plugin) generateForEntry(
	resp *plugin.Response,
	entry migrationEntry,
	goPath, pkg string,
	migrateEntryNames set.Set[string],
	req *plugin.Request,
) error {
	if entry.SchemaChange == nil {
		return nil
	}
	change := entry.SchemaChange
	versionDir := fmt.Sprintf("v%d", change.Version)

	// If there's a previous migration at a DIFFERENT version, retarget its
	// developer transform into the new version's sub-package.
	if len(entry.ExistingVersions) > 0 {
		prevVersion := entry.ExistingVersions[len(entry.ExistingVersions)-1]
		if prevVersion != change.Version {
			retargeted, deleteFile, err := retargetTransform(goPath, change.Version, req.RepoRoot)
			if err != nil {
				return errors.Wrapf(err, "failed to retarget v%d transform for %s", prevVersion, entry.GoName)
			}
			if retargeted.Path != "" {
				resp.Files = append(resp.Files, retargeted)
				resp.Deletions = append(resp.Deletions, deleteFile)
			}
		}
	}

	pkgTypes, codecReachable := collectPackageTypes(change.OldType, req.OldResolutions)

	pathMap := make(map[string]string, len(pkgTypes))
	for origPath := range pkgTypes {
		pathMap[origPath] = origPath + "/migrations/" + versionDir
	}
	rewrittenOldTable := rewriteOutputPaths(req.OldResolutions, pathMap)

	newEntry, _ := req.Resolutions.Get(change.OldType.QualifiedName)
	schemaDiff := SchemaDiff(change.OldType, newEntry, req.OldResolutions, req.Resolutions)

	for origPath, types := range pkgTypes {
		mirroredPath := pathMap[origPath]
		typeContent, err := generateFrozenTypesFile(types, rewrittenOldTable, mirroredPath, req.RepoRoot)
		if err != nil {
			return errors.Wrapf(err, "failed to generate frozen types for %s", origPath)
		}
		entryMethods := generateGorpEntryMethods(types, migrateEntryNames)
		if len(entryMethods) > 0 {
			typeContent = append(typeContent, entryMethods...)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    mirroredPath + "/types.gen.go",
			Content: typeContent,
		})

		// Generate frozen codec for this package with per-type Encode/Decode
		// functions. The entry type's package gets the Codec adapter.
		codecEntries := codecEntriesForTypes(types, migrateEntryNames, codecReachable)
		if len(codecEntries) > 0 {
			codecContent, err := gomarshal.GenerateCodecFile(
				versionDir, mirroredPath,
				codecEntries,
				rewrittenOldTable,
				req.RepoRoot,
			)
			if err != nil {
				return errors.Wrapf(err, "failed to generate frozen codec for %s", origPath)
			}
			resp.Files = append(resp.Files, plugin.File{
				Path:    mirroredPath + "/codec.gen.go",
				Content: codecContent,
			})
		}

		if origPath != goPath && needsAutoMigrate(types, schemaDiff) {
			if err := p.generateSubPackageMigration(resp, versionDir, mirroredPath, types, schemaDiff, rewrittenOldTable, req); err != nil {
				return err
			}
		}
	}

	// Top-level auto-copy for the entry type's package.
	entryTypes := pkgTypes[goPath]
	if needsAutoMigrate(entryTypes, schemaDiff) {
		autoCopyContent, err := generateAutoCopy(
			pkg, goPath, req.RepoRoot,
			entryTypes, schemaDiff, rewrittenOldTable, req.Resolutions,
		)
		if err != nil {
			return errors.Wrapf(err, "failed to generate top-level auto-copy for %s", goPath)
		}
		if autoCopyContent != nil {
			resp.Files = append(resp.Files, plugin.File{
				Path:    goPath + "/migrate_auto.gen.go",
				Content: autoCopyContent,
			})
		}
	}

	// Developer transform template.
	entryMirrorPath := goPath + "/migrations/" + versionDir
	templateFile := goPath + "/migrate.go"
	templateFullPath := filepath.Join(req.RepoRoot, templateFile)
	if _, statErr := os.Stat(templateFullPath); os.IsNotExist(statErr) {
		entryMirrorImport, _ := resolveImportPath(entryMirrorPath, req.RepoRoot)
		tc, err := renderTransformTemplate(pkg, entry.GoName, change.Version, versionDir, entryMirrorImport)
		if err != nil {
			return errors.Wrapf(err, "failed to generate transform template")
		}
		resp.Files = append(resp.Files, plugin.File{Path: templateFile, Content: tc})
	}

	return nil
}

func codecEntriesForTypes(
	types []resolution.Type,
	migrateEntryNames set.Set[string],
	reachable set.Set[string],
) []gomarshal.CodecEntry {
	var entries []gomarshal.CodecEntry
	for _, t := range types {
		if _, ok := t.Form.(resolution.StructForm); !ok {
			continue
		}
		if !reachable.Contains(t.QualifiedName) {
			continue
		}
		goName := goNameFromType(t)
		ce := gomarshal.CodecEntry{GoName: goName, Type: t}
		if migrateEntryNames.Contains(goName) {
			ce.Adapter = true
		}
		entries = append(entries, ce)
	}
	return entries
}

func goNameFromType(t resolution.Type) string {
	if domain, ok := t.Domains["go"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return naming.ToPascalCase(t.Name)
}

func (p *Plugin) generateSubPackageMigration(
	resp *plugin.Response,
	versionDir, mirroredPath string,
	types []resolution.Type,
	schemaDiff map[string]TypeDiff,
	rewrittenOldTable *resolution.Table,
	req *plugin.Request,
) error {
	autoCopyContent, err := generateAutoCopy(
		versionDir, mirroredPath, req.RepoRoot,
		types, schemaDiff, rewrittenOldTable, req.Resolutions,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to generate auto-copy for %s", mirroredPath)
	}
	if autoCopyContent != nil {
		resp.Files = append(resp.Files, plugin.File{
			Path:    mirroredPath + "/migrate_auto.gen.go",
			Content: autoCopyContent,
		})
	}

	migrateFile := mirroredPath + "/migrate.go"
	migrateFullPath := filepath.Join(req.RepoRoot, migrateFile)
	if _, statErr := os.Stat(migrateFullPath); os.IsNotExist(statErr) {
		tc, err := renderTypeMigrateTemplate(versionDir, mirroredPath, types, schemaDiff, req.Resolutions, req.RepoRoot)
		if err != nil {
			return errors.Wrapf(err, "failed to generate type migrate template for %s", mirroredPath)
		}
		if tc != nil {
			resp.Files = append(resp.Files, plugin.File{Path: migrateFile, Content: tc})
		}
	}
	return nil
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
	return &schemaChange{Version: req.SnapshotVersion, OldType: oldType}, nil
}

// collectPackageTypes walks the entry type's dependency tree and groups all
// Oracle-defined struct types by their @go output path. It also returns the
// set of types that are directly reachable from the entry type's serialization
// tree (before package expansion), which determines which types need codec
// functions.
func collectPackageTypes(
	entryType resolution.Type,
	table *resolution.Table,
) (pkgTypes map[string][]resolution.Type, serializationReachable set.Set[string]) {
	result := make(map[string][]resolution.Type)
	visited := make(set.Set[string])
	collectPkgTypesWalk(entryType, table, result, visited)
	serializationReachable = visited
	// Expand each collected package: if any type from a package is needed,
	// include ALL types from that package (for complete frozen types files).
	for goPath := range result {
		expanded := make(set.Set[string])
		for _, t := range result[goPath] {
			expanded.Add(t.QualifiedName)
		}
		for _, t := range table.TypesWithDomain("go") {
			if expanded.Contains(t.QualifiedName) {
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
	return result, serializationReachable
}

func collectPkgTypesWalk(
	typ resolution.Type,
	table *resolution.Table,
	result map[string][]resolution.Type,
	visited set.Set[string],
) {
	if visited.Contains(typ.QualifiedName) {
		return
	}
	visited.Add(typ.QualifiedName)

	goPath := output.GetPath(typ, "go")
	if goPath != "" {
		switch typ.Form.(type) {
		case resolution.StructForm, resolution.AliasForm, resolution.DistinctForm, resolution.EnumForm:
			result[goPath] = append(result[goPath], typ)
		}
	}

	if sf, ok := typ.Form.(resolution.StructForm); ok {
		for _, ext := range sf.Extends {
			walkRefForPkgTypes(ext, table, result, visited)
		}
	}

	fields := resolution.UnifiedFields(typ, table)
	for _, f := range fields {
		walkRefForPkgTypes(f.Type, table, result, visited)
	}
}

func walkRefForPkgTypes(
	ref resolution.TypeRef,
	table *resolution.Table,
	result map[string][]resolution.Type,
	visited set.Set[string],
) {
	resolved, ok := ref.Resolve(table)
	if !ok {
		return
	}
	// Always walk type arguments (e.g., Status[StatusDetails] needs StatusDetails).
	for _, arg := range ref.TypeArgs {
		walkRefForPkgTypes(arg, table, result, visited)
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
	}
}

func resolveImportPath(outputPath, repoRoot string) (string, error) {
	return gomod.ResolveImportPath(outputPath, repoRoot, "github.com/synnaxlabs/synnax/"), nil
}

func discoverExistingVersions(goPath, repoRoot string) ([]int, error) {
	migrationsDir := filepath.Join(repoRoot, goPath, "migrations")
	dirEntries, err := os.ReadDir(migrationsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var versions []int
	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() || !strings.HasPrefix(dirEntry.Name(), "v") {
			continue
		}
		v, err := strconv.Atoi(dirEntry.Name()[1:])
		if err != nil {
			continue
		}
		codecPath := filepath.Join(migrationsDir, dirEntry.Name(), "codec.gen.go")
		if _, statErr := os.Stat(codecPath); statErr == nil {
			versions = append(versions, v)
		}
	}
	sort.Ints(versions)
	return versions, nil
}

// filterSchemaChangeVersions removes versions that don't represent actual schema
// changes for this entry type. A version directory may exist solely because a
// parent type's migration created frozen dependency types there (e.g., Arc's
// migration freezing Label types), not because this type's own schema changed.
//
// A version is considered a real schema change if a migrate.go template exists:
//   - For the latest version: at the package level (goPath/migrate.go)
//   - For earlier versions: retargeted into the version directory (migrations/vN/migrate.go)
func filterSchemaChangeVersions(goPath string, versions []int, repoRoot string) []int {
	if len(versions) == 0 {
		return versions
	}
	hasPkgMigrate := false
	if _, err := os.Stat(filepath.Join(repoRoot, goPath, "migrate.go")); err == nil {
		hasPkgMigrate = true
	}
	var filtered []int
	for i, v := range versions {
		isLast := i == len(versions)-1
		if isLast && hasPkgMigrate {
			filtered = append(filtered, v)
			continue
		}
		vMigrate := filepath.Join(repoRoot, goPath, "migrations", fmt.Sprintf("v%d", v), "migrate.go")
		if _, err := os.Stat(vMigrate); err == nil {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// retargetTransform reads the existing top-level transform file, rewrites its
// package declaration to target the new version sub-package, and returns the
// new file content plus the old file path to delete.
func retargetTransform(goPath string, newVersion int, repoRoot string) (plugin.File, string, error) {
	oldFile := goPath + "/migrate.go"
	content, err := os.ReadFile(filepath.Join(repoRoot, oldFile))
	if os.IsNotExist(err) {
		return plugin.File{}, "", nil
	}
	if err != nil {
		return plugin.File{}, "", errors.Wrapf(err, "failed to read transform %s", oldFile)
	}
	src := string(content)
	newPkg := fmt.Sprintf("v%d", newVersion)
	src = regexp.MustCompile(`package \w+`).ReplaceAllString(src, "package "+newPkg)
	src = strings.Replace(src,
		"// Generated by oracle as a template. Edit this file.",
		"// Retargeted by oracle. Edit freely.",
		1)
	newPath := fmt.Sprintf("%s/migrations/v%d/migrate.go", goPath, newVersion)
	return plugin.File{Path: newPath, Content: []byte(src)}, oldFile, nil
}

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

func generateGorpEntryMethods(types []resolution.Type, migrateEntryNames set.Set[string]) []byte {
	var buf bytes.Buffer
	for _, typ := range types {
		if !migrateEntryNames.Contains(getGoName(typ)) {
			continue
		}
		form, ok := typ.Form.(resolution.StructForm)
		if !ok || !form.HasKeyDomain {
			continue
		}
		goName := getGoName(typ)
		keyFieldGoName := findKeyFieldGoName(form)
		if keyFieldGoName == "" {
			continue
		}
		buf.WriteString(fmt.Sprintf(
			"\nfunc (e %s) GorpKey() Key { return e.%s }\n",
			goName, keyFieldGoName,
		))
		buf.WriteString(fmt.Sprintf(
			"\nfunc (e %s) SetOptions() []any { return nil }\n",
			goName,
		))
	}
	return buf.Bytes()
}

func findKeyFieldGoName(form resolution.StructForm) string {
	for _, f := range form.Fields {
		if _, ok := f.Domains["key"]; ok {
			return naming.GetFieldName(f)
		}
	}
	return ""
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

// --- Templates ---

var migrateTmpl = template.Must(template.New("migrate").Parse(
	`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import (
	"github.com/synnaxlabs/x/gorp"
{{- range .VersionImports}}
	{{.Alias}} "{{.Path}}"
{{- end}}
)
{{range $entry := .Entries}}
func {{$entry.GoName}}Migrations() []gorp.Migration {
	return []gorp.Migration{
		gorp.NewCodecTransition[Key, {{$entry.GoName}}]("msgpack_to_binary", {{$entry.GoName}}Codec),
{{- range $entry.SchemaChanges}}
{{- if .IsIntermediate}}
		gorp.WithDependencies(gorp.NewTypedMigration[Key, Key, {{.ImportAlias}}.{{$entry.GoName}}, {{.NextImportAlias}}.{{$entry.GoName}}](
			"v{{.Version}}_schema_migration",
			{{.ImportAlias}}.{{$entry.GoName}}Codec,
			{{.NextImportAlias}}.{{$entry.GoName}}Codec,
			{{.NextImportAlias}}.Migrate{{$entry.GoName}},
		), "{{.DependsOn}}"),
{{- else}}
		gorp.WithDependencies(gorp.NewTypedMigration[Key, Key, {{.ImportAlias}}.{{$entry.GoName}}, {{$entry.GoName}}](
			"v{{.Version}}_schema_migration",
			{{.ImportAlias}}.{{$entry.GoName}}Codec,
			{{$entry.GoName}}Codec,
			Migrate{{$entry.GoName}},
		), "{{.DependsOn}}"),
{{- end}}
{{- end}}
	}
}
{{end}}`))

type migrateTemplateData struct {
	Package        string
	Entries        []migrateTemplateEntry
	VersionImports []versionImport
}

type versionImport struct{ Alias, Path string }

type migrateTemplateEntry struct {
	GoName        string
	SchemaChanges []migrateSchemaChange
}

type migrateSchemaChange struct {
	Version         int
	ImportAlias     string
	IsIntermediate  bool
	NextImportAlias string
	DependsOn       string
}

func renderMigrateFile(pkg, goPath string, entries []migrationEntry, repoRoot string) ([]byte, error) {
	var templateEntries []migrateTemplateEntry
	importSet := make(map[string]versionImport)
	for _, entry := range entries {
		te := migrateTemplateEntry{GoName: entry.GoName}
		allVersions := append([]int{}, entry.ExistingVersions...)
		if entry.SchemaChange != nil {
			v := entry.SchemaChange.Version
			if len(allVersions) == 0 || allVersions[len(allVersions)-1] != v {
				allVersions = append(allVersions, v)
			}
		}
		for i, version := range allVersions {
			vDir := fmt.Sprintf("v%d", version)
			subPkg := fmt.Sprintf("%s/migrations/%s", goPath, vDir)
			importPath := gomod.ResolveImportPath(subPkg, repoRoot, "github.com/synnaxlabs/synnax/")
			importSet[vDir] = versionImport{Alias: vDir, Path: importPath}
			isLast := i == len(allVersions)-1
			dependsOn := "msgpack_to_binary"
			if i > 0 {
				dependsOn = fmt.Sprintf("v%d_schema_migration", allVersions[i-1])
			}
			sc := migrateSchemaChange{
				Version: version, ImportAlias: vDir,
				IsIntermediate: !isLast, DependsOn: dependsOn,
			}
			if !isLast {
				sc.NextImportAlias = fmt.Sprintf("v%d", allVersions[i+1])
			}
			te.SchemaChanges = append(te.SchemaChanges, sc)
		}
		templateEntries = append(templateEntries, te)
	}
	var imports []versionImport
	for _, v := range importSet {
		imports = append(imports, v)
	}
	var buf bytes.Buffer
	if err := migrateTmpl.Execute(&buf, migrateTemplateData{
		Package: pkg, Entries: templateEntries, VersionImports: imports,
	}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var transformTmpl = template.Must(template.New("transform").Parse(
	`// Generated by oracle as a template. Edit this file.
//
// AutoMigrate handles field copying. Customize non-zero defaults below.

package {{.Package}}

import (
	"context"

	{{.VersionDir}} "{{.MigrationsImport}}"
)

func Migrate{{.GoName}}(ctx context.Context, old {{.VersionDir}}.{{.GoName}}) ({{.GoName}}, error) {
	return AutoMigrate{{.GoName}}(ctx, old)
}
`))

func renderTransformTemplate(pkg, goName string, version int, vDir, migrationsImport string) ([]byte, error) {
	var buf bytes.Buffer
	err := transformTmpl.Execute(&buf, struct {
		Package, GoName, VersionDir, MigrationsImport string
		Version                                       int
	}{pkg, goName, vDir, migrationsImport, version})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var typeMigrateTmpl = template.Must(template.New("typeMigrate").Parse(
	`// Generated by oracle as a template. Edit this file.

package {{.Package}}

import (
	"context"
{{range .Imports}}
	{{.Alias}} "{{.Path}}"
{{- end}}
)
{{range .Functions}}
func Migrate{{.GoName}}(ctx context.Context, old {{.OldTypeName}}) ({{.NewTypeName}}, error) {
	migrated, err := AutoMigrate{{.GoName}}(ctx, old)
	if err != nil {
		return {{.NewTypeName}}{}, err
	}
	// New/changed fields - set non-zero defaults if needed:
{{- range .NewFields}}
	// migrated.{{.}} is zero-valued
{{- end}}
	return migrated, nil
}
{{end}}`))

func renderTypeMigrateTemplate(
	pkg, mirroredPath string,
	types []resolution.Type,
	diff map[string]TypeDiff,
	newTable *resolution.Table,
	repoRoot string,
) ([]byte, error) {
	type tmplFunc struct {
		GoName, OldTypeName, NewTypeName string
		NewFields                        []string
	}
	type tmplData struct {
		Package   string
		Imports   []versionImport
		Functions []tmplFunc
	}
	data := tmplData{Package: pkg}
	importSet := make(map[string]versionImport)
	for _, typ := range types {
		td, ok := diff[typ.QualifiedName]
		if !ok || td.Kind != TypeChanged {
			continue
		}
		goName := getGoName(typ)
		newType, _ := newTable.Get(typ.QualifiedName)
		newGoPath := output.GetPath(newType, "go")
		newTypeName := getGoName(newType)
		if newGoPath != mirroredPath {
			ip := gomod.ResolveImportPath(newGoPath, repoRoot, gomod.DefaultModulePrefix)
			alias := naming.DerivePackageAlias(newGoPath, pkg)
			importSet[ip] = versionImport{Alias: alias, Path: ip}
			newTypeName = alias + "." + newTypeName
		}
		var newFields []string
		for _, fd := range td.ChangedFields {
			if fd.Kind == FieldKindAdded {
				newFields = append(newFields, naming.GetFieldName(*fd.NewField))
			}
		}
		data.Functions = append(data.Functions, tmplFunc{
			GoName: goName, OldTypeName: goName, NewTypeName: newTypeName, NewFields: newFields,
		})
	}
	if len(data.Functions) == 0 {
		return nil, nil
	}
	for _, v := range importSet {
		data.Imports = append(data.Imports, v)
	}
	var buf bytes.Buffer
	if err := typeMigrateTmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
