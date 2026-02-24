// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package migrate provides an Oracle plugin that generates gorp migration files.
// For each gorp entry type (annotated with @go marshal and @key), it generates:
//   - A v1 legacy type snapshot in migrations/v1/v1.gen.go
//   - A migration registration file in migrations/migrate.gen.go
package migrate

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/deps"
	"github.com/synnaxlabs/oracle/diff"
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

// Plugin generates gorp migration files for structs annotated with @go marshal
// that also have a @key field and @pb domain.
type Plugin struct{ Options Options }

// Options configures the go/migrate plugin.
type Options struct {
	MigrateFileName string
	V1FileName      string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		MigrateFileName: "migrate.gen.go",
		V1FileName:      "v1.gen.go",
	}
}

// New creates a new go/migrate plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

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

func getGoName(s resolution.Type) string {
	if domain, ok := s.Domains["go"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
}

type migrateEntry struct {
	GoName    string
	GoPath    string
	KeyField  keyFieldInfo
	Fields    []fieldInfo
	Namespace string
	Type      resolution.Type
}

type keyFieldInfo struct {
	FieldGoName      string
	KeyTypeName      string
	KeyBaseType      string
	KeyIsAlias       bool
	KeyImports       []importInfo
	MigrateKeyType   string
	MigrateKeyImport *importInfo
}

type importInfo struct {
	Category string
	Path     string
	Alias    string
}

type fieldInfo struct {
	GoName  string
	GoType  string
	JSONTag string
}

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	if req.OldResolutions == nil {
		return p.generateInitial(req)
	}
	return p.generateWithDiff(req)
}

func (p *Plugin) generateInitial(req *plugin.Request) (*plugin.Response, error) {
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

		me, err := p.buildMigrateEntry(entry, goPath, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to build migrate entry for %s", entry.Name)
		}

		if _, exists := outputEntries[goPath]; !exists {
			outputOrder = append(outputOrder, goPath)
		}
		outputEntries[goPath] = append(outputEntries[goPath], me)
	}

	for _, goPath := range outputOrder {
		entries := outputEntries[goPath]

		v1Content, err := p.generateV1File(goPath, entries, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate v1 for %s", goPath)
		}
		if len(v1Content) > 0 {
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/migrations/v1/%s", goPath, p.Options.V1FileName),
				Content: v1Content,
			})
		}

		migrateContent, err := p.generateMigrateFile(goPath, entries, nil, 0)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate migrate for %s", goPath)
		}
		if len(migrateContent) > 0 {
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/%s", goPath, p.Options.MigrateFileName),
				Content: migrateContent,
			})
		}
	}

	return resp, nil
}

// migrationMode classifies what kind of migration code should be generated.
type migrationMode int

const (
	modeSkeleton    migrationMode = iota // direct schema change on gorp entry
	modePropagation                      // nested type changed, parent affected
)

// versionMigration holds migration info for a single entry at a specific version.
type versionMigration struct {
	Entry migrateEntry
	Mode  migrationMode
	Diff  *diff.TypeDiff
}

func (p *Plugin) generateWithDiff(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	tableDiffs := diff.DiffTables(req.OldResolutions, req.Resolutions)
	if len(tableDiffs) == 0 {
		return p.generateInitial(req)
	}

	changedTypeNames := make([]string, len(tableDiffs))
	changedTypeMap := make(map[string]*diff.TypeDiff, len(tableDiffs))
	for i, td := range tableDiffs {
		changedTypeNames[i] = td.TypeName
		tdCopy := td
		changedTypeMap[td.TypeName] = &tdCopy
	}

	graph := deps.Build(req.Resolutions)
	affectedEntries := graph.AffectedEntries(changedTypeNames)

	if len(affectedEntries) == 0 {
		return p.generateInitial(req)
	}

	version := req.SnapshotVersion + 1
	versionDir := fmt.Sprintf("v%d", version)

	// Classify affected entries into version migrations.
	outputMigrations := make(map[string][]versionMigration)
	var migrationOutputOrder []string

	for _, qname := range affectedEntries {
		entry, ok := req.Resolutions.Get(qname)
		if !ok {
			continue
		}
		if !hasMigrateAnnotation(entry) {
			continue
		}
		goPath := output.GetPath(entry, "go")
		if goPath == "" {
			continue
		}

		me, err := p.buildMigrateEntry(entry, goPath, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to build migrate entry for %s", entry.Name)
		}

		mode := modePropagation
		var td *diff.TypeDiff
		if d, directlyChanged := changedTypeMap[qname]; directlyChanged {
			mode = modeSkeleton
			td = d
		}
		if _, exists := outputMigrations[goPath]; !exists {
			migrationOutputOrder = append(migrationOutputOrder, goPath)
		}
		outputMigrations[goPath] = append(outputMigrations[goPath], versionMigration{
			Entry: me, Mode: mode, Diff: td,
		})
	}

	// Collect ALL migrate entries grouped by goPath (for v1 + migrate.gen.go).
	allEntries := make(map[string][]migrateEntry)
	var allOrder []string
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
		me, err := p.buildMigrateEntry(entry, goPath, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to build migrate entry for %s", entry.Name)
		}
		if _, exists := allEntries[goPath]; !exists {
			allOrder = append(allOrder, goPath)
		}
		allEntries[goPath] = append(allEntries[goPath], me)
	}

	// For every goPath that has migrate entries, generate v1 snapshot + migrate.gen.go.
	for _, goPath := range allOrder {
		entries := allEntries[goPath]

		// v1 snapshot (frozen initial types).
		v1Content, err := p.generateV1File(goPath, entries, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate v1 for %s", goPath)
		}
		if len(v1Content) > 0 {
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/migrations/v1/%s", goPath, p.Options.V1FileName),
				Content: v1Content,
			})
		}

		// migrate.gen.go — includes version migrations if this path is affected.
		var migrations []versionMigration
		if m, ok := outputMigrations[goPath]; ok {
			migrations = m
		}
		migrateContent, err := p.generateMigrateFile(goPath, entries, migrations, req.SnapshotVersion)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate migrate for %s", goPath)
		}
		if len(migrateContent) > 0 {
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/%s", goPath, p.Options.MigrateFileName),
				Content: migrateContent,
			})
		}
	}

	// For affected paths, generate vN snapshot + auto + post files.
	prevVersionDir := fmt.Sprintf("v%d", req.SnapshotVersion)

	for _, goPath := range migrationOutputOrder {
		migrations := outputMigrations[goPath]

		// Frozen old struct snapshot (vN.gen.go) from OLD resolutions.
		oldEntries := make([]migrateEntry, 0, len(migrations))
		for _, vm := range migrations {
			oldType, ok := req.OldResolutions.Get(vm.Entry.Type.QualifiedName)
			if !ok {
				continue
			}
			oldMe, err := p.buildMigrateEntry(oldType, goPath, &plugin.Request{
				Resolutions: req.OldResolutions,
				RepoRoot:    req.RepoRoot,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to build old migrate entry for %s", vm.Entry.GoName)
			}
			oldEntries = append(oldEntries, oldMe)
		}

		if len(oldEntries) > 0 {
			snapshotContent, err := p.generateV1File(goPath, oldEntries, &plugin.Request{
				Resolutions: req.OldResolutions,
				RepoRoot:    req.RepoRoot,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate snapshot for %s", goPath)
			}
			if len(snapshotContent) > 0 {
				resp.Files = append(resp.Files, plugin.File{
					Path:    fmt.Sprintf("%s/migrations/%s/%s.gen.go", goPath, versionDir, versionDir),
					Content: snapshotContent,
				})
			}
		}

		// auto.gen.go — references previous version package for the old type.
		autoContent, err := p.generateAutoFile(goPath, migrations, versionDir, prevVersionDir, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate auto for %s", goPath)
		}
		if len(autoContent) > 0 {
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/migrations/%s/auto.gen.go", goPath, versionDir),
				Content: autoContent,
			})
		}

		// migrate.go — post-migrate template referencing previous version.
		postContent, err := p.generatePostFile(goPath, migrations, versionDir, prevVersionDir)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate post for %s", goPath)
		}
		if len(postContent) > 0 {
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/migrations/%s/migrate.go", goPath, versionDir),
				Content: postContent,
			})
		}
	}

	return resp, nil
}


func (p *Plugin) buildMigrateEntry(
	entry resolution.Type,
	goPath string,
	req *plugin.Request,
) (migrateEntry, error) {
	goName := getGoName(entry)
	if goName == "" {
		goName = entry.Name
	}

	me := migrateEntry{
		GoName:    goName,
		GoPath:    goPath,
		Namespace: entry.Namespace,
		Type:      entry,
	}

	fields := resolution.UnifiedFields(entry, req.Resolutions)
	for _, field := range fields {
		_, isKey := field.Domains["key"]
		if isKey {
			me.KeyField = p.resolveKeyField(field, entry, req)
		}
	}

	v1OutputPath := goPath + "/migrations/v1"
	v1Pkg := "v1"
	mgr := imports.NewManager()

	ctx := &resolver.Context{
		Table:                         req.Resolutions,
		OutputPath:                    v1OutputPath,
		Namespace:                     entry.Namespace,
		RepoRoot:                      req.RepoRoot,
		DomainName:                    "go",
		SubstituteDefaultedTypeParams: true,
	}

	r := &resolver.Resolver{
		Formatter:       gotypes.GoFormatter(),
		ImportResolver:  &gotypes.GoImportResolver{RepoRoot: req.RepoRoot, CurrentPackage: v1Pkg},
		ImportAdder:     mgr,
		PrimitiveMapper: primitiveMapper,
	}

	for _, field := range fields {
		_, isKey := field.Domains["key"]
		var goType string
		if isKey {
			goType = me.KeyField.KeyTypeName
		} else {
			goType = r.ResolveTypeRef(field.Type, ctx)
			if field.IsHardOptional && !strings.HasPrefix(goType, "[]") &&
				!strings.HasPrefix(goType, "map[") &&
				!strings.HasPrefix(goType, "binary.MsgpackEncodedJSON") {
				goType = "*" + goType
			}
		}
		fieldGoName := toPascalCase(field.Name)
		jsonTag := lo.SnakeCase(field.Name)
		if field.IsHardOptional {
			jsonTag += ",omitempty"
		}
		me.Fields = append(me.Fields, fieldInfo{
			GoName:  fieldGoName,
			GoType:  goType,
			JSONTag: jsonTag,
		})
	}

	return me, nil
}

func (p *Plugin) resolveKeyField(
	field resolution.Field,
	entry resolution.Type,
	req *plugin.Request,
) keyFieldInfo {
	kf := keyFieldInfo{
		FieldGoName: toPascalCase(field.Name),
		KeyTypeName: "Key",
	}

	resolved, ok := field.Type.Resolve(req.Resolutions)
	if !ok {
		kf.KeyBaseType = resolveKeyPrimitive(field.Type.Name)
		kf.KeyIsAlias = false
		kf.MigrateKeyType = kf.KeyBaseType
		return kf
	}

	switch form := resolved.Form.(type) {
	case resolution.AliasForm:
		kf.KeyIsAlias = true
		kf.KeyBaseType, kf.KeyImports = resolveBaseType(form.Target, req)
	case resolution.DistinctForm:
		kf.KeyIsAlias = false
		kf.KeyBaseType, kf.KeyImports = resolveBaseType(form.Base, req)
	default:
		kf.KeyBaseType = resolveKeyPrimitive(field.Type.Name)
	}

	goName := getGoName(resolved)
	if goName == "" {
		goName = resolved.Name
	}
	entryGoPath := output.GetPath(entry, "go")
	keyGoPath := output.GetPath(resolved, "go")
	if keyGoPath == "" || keyGoPath == entryGoPath {
		kf.MigrateKeyType = goName
		kf.MigrateKeyImport = nil
	} else {
		keyAlias := naming.DerivePackageAlias(keyGoPath, "migrations")
		keyImportPath := resolveGoImportPath(keyGoPath, req.RepoRoot)
		kf.MigrateKeyType = keyAlias + "." + goName
		kf.MigrateKeyImport = &importInfo{
			Category: "internal",
			Path:     keyImportPath,
			Alias:    keyAlias,
		}
	}

	return kf
}

func resolveBaseType(ref resolution.TypeRef, _ *plugin.Request) (string, []importInfo) {
	mapping := primitiveMapper.Map(ref.Name)
	if mapping.TargetType != "" {
		var imps []importInfo
		for _, imp := range mapping.Imports {
			imps = append(imps, importInfo{
				Category: imp.Category,
				Path:     imp.Path,
				Alias:    imp.Name,
			})
		}
		return mapping.TargetType, imps
	}
	return resolveKeyPrimitive(ref.Name), nil
}

func resolveKeyPrimitive(name string) string {
	switch name {
	case "string":
		return "string"
	case "uint32":
		return "uint32"
	case "uint64":
		return "uint64"
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	case "uuid":
		return "uuid.UUID"
	default:
		return name
	}
}

func toPascalCase(s string) string {
	if naming.IsScreamingCase(s) {
		return s
	}
	return lo.PascalCase(s)
}

func resolveGoImportPath(outputPath, repoRoot string) string {
	return gomod.ResolveImportPath(outputPath, repoRoot, gomod.DefaultModulePrefix)
}

// generateV1File generates the legacy type snapshot file (v1.gen.go).
func (p *Plugin) generateV1File(
	goPath string,
	entries []migrateEntry,
	req *plugin.Request,
) ([]byte, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	v1OutputPath := goPath + "/migrations/v1"
	v1Pkg := "v1"
	mgr := imports.NewManager()

	ctx := &resolver.Context{
		Table:                         req.Resolutions,
		OutputPath:                    v1OutputPath,
		Namespace:                     entries[0].Namespace,
		RepoRoot:                      req.RepoRoot,
		DomainName:                    "go",
		SubstituteDefaultedTypeParams: true,
	}

	r := &resolver.Resolver{
		Formatter:       gotypes.GoFormatter(),
		ImportResolver:  &gotypes.GoImportResolver{RepoRoot: req.RepoRoot, CurrentPackage: v1Pkg},
		ImportAdder:     mgr,
		PrimitiveMapper: primitiveMapper,
	}

	type v1Entry struct {
		GoName   string
		KeyField keyFieldInfo
		Fields   []fieldInfo
	}

	var v1Entries []v1Entry
	for _, entry := range entries {
		fields := resolution.UnifiedFields(entry.Type, req.Resolutions)

		var v1Fields []fieldInfo
		for _, field := range fields {
			_, isKey := field.Domains["key"]
			var goType string
			if isKey {
				goType = entry.KeyField.KeyTypeName
			} else {
				goType = r.ResolveTypeRef(field.Type, ctx)
				if field.IsHardOptional && !strings.HasPrefix(goType, "[]") &&
					!strings.HasPrefix(goType, "map[") &&
					!strings.HasPrefix(goType, "binary.MsgpackEncodedJSON") {
					goType = "*" + goType
				}
			}
			jsonTag := lo.SnakeCase(field.Name)
			if field.IsHardOptional {
				jsonTag += ",omitempty"
			}
			v1Fields = append(v1Fields, fieldInfo{
				GoName:  toPascalCase(field.Name),
				GoType:  goType,
				JSONTag: jsonTag,
			})
		}

		for _, imp := range entry.KeyField.KeyImports {
			mgr.AddImport(imp.Category, imp.Path, imp.Alias)
		}

		v1Entries = append(v1Entries, v1Entry{
			GoName:   entry.GoName,
			KeyField: entry.KeyField,
			Fields:   v1Fields,
		})
	}

	data := struct {
		Package         string
		ExternalImports []string
		InternalImports []imports.InternalImportData
		HasImports      bool
		Entries         []v1Entry
	}{
		Package:         "v1",
		ExternalImports: mgr.ExternalImports(),
		InternalImports: mgr.InternalImports(),
		HasImports:      mgr.HasImports(),
		Entries:         v1Entries,
	}

	tmpl, err := template.New("v1").Parse(v1Template)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse v1 template")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, errors.Wrap(err, "failed to execute v1 template")
	}
	_ = r // keep reference
	return buf.Bytes(), nil
}

// generateMigrateFile generates the migration registration file (migrate.gen.go).
// The file is placed in the same package as the service to avoid import cycles.
// The codec is accepted as a parameter rather than imported from pb/.
// When versionMigrations is non-nil, additional NewTypedMigration entries are
// appended after the initial codec transition.
func (p *Plugin) generateMigrateFile(
	goPath string,
	entries []migrateEntry,
	versionMigrations []versionMigration,
	snapshotVersion int,
) ([]byte, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	pkg := naming.DerivePackageName(goPath)

	type versionMigrateData struct {
		GoName      string
		KeyType     string
		OldVersion  string
		NewVersion  string
		OldPkg      string
		NewPkg      string
		ImportPath  string
		OldImport   string
		NewImport   string
	}

	type migrateData struct {
		GoName            string
		KeyType           string
		VersionMigrations []versionMigrateData
	}

	migrateEntries := make([]migrateData, len(entries))
	for i, e := range entries {
		md := migrateData{
			GoName:  e.GoName,
			KeyType: e.KeyField.KeyTypeName,
		}
		for _, vm := range versionMigrations {
			if vm.Entry.GoName == e.GoName {
				prevVersion := fmt.Sprintf("v%d", snapshotVersion)
				nextVersion := fmt.Sprintf("v%d", snapshotVersion+1)
				importBase := resolveGoImportPath(goPath+"/migrations", "")
				md.VersionMigrations = append(md.VersionMigrations, versionMigrateData{
					GoName:     e.GoName,
					KeyType:    e.KeyField.KeyTypeName,
					OldVersion: prevVersion,
					NewVersion: nextVersion,
					OldPkg:     prevVersion,
					NewPkg:     nextVersion,
					OldImport:  importBase + "/" + prevVersion,
					NewImport:  importBase + "/" + nextVersion,
				})
			}
		}
		migrateEntries[i] = md
	}

	data := struct {
		Package           string
		Entries           []migrateData
		HasVersionImports bool
	}{
		Package:           pkg,
		Entries:           migrateEntries,
		HasVersionImports: len(versionMigrations) > 0,
	}

	tmpl, err := template.New("migrate").Parse(migrateTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse migrate template")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, errors.Wrap(err, "failed to execute migrate template")
	}
	return buf.Bytes(), nil
}


func (p *Plugin) generateAutoFile(
	goPath string,
	migrations []versionMigration,
	versionDir string,
	prevVersionDir string,
	req *plugin.Request,
) ([]byte, error) {
	if len(migrations) == 0 {
		return nil, nil
	}

	oldImportPath := resolveGoImportPath(goPath+"/migrations/"+prevVersionDir, req.RepoRoot)

	type autoField struct {
		GoName    string
		Unchanged bool
		Added     bool
		Removed   bool
		Changed   bool
		OldType   string
		NewType   string
	}

	type autoEntry struct {
		GoName string
		Mode   string
		Fields []autoField
	}

	var entries []autoEntry
	for _, vm := range migrations {
		ae := autoEntry{
			GoName: vm.Entry.GoName,
		}
		if vm.Mode == modeSkeleton {
			ae.Mode = "skeleton"
			if vm.Diff != nil {
				for _, fd := range vm.Diff.Fields {
					af := autoField{
						GoName:    toPascalCase(fd.Name),
						Unchanged: fd.Kind == diff.FieldUnchanged,
						Added:     fd.Kind == diff.FieldAdded,
						Removed:   fd.Kind == diff.FieldRemoved,
						Changed:   fd.Kind == diff.FieldTypeChanged,
						OldType:   fd.OldType,
						NewType:   fd.NewType,
					}
					ae.Fields = append(ae.Fields, af)
				}
			}
		} else {
			ae.Mode = "propagation"
			fields := resolution.UnifiedFields(vm.Entry.Type, req.Resolutions)
			for _, f := range fields {
				ae.Fields = append(ae.Fields, autoField{
					GoName:    toPascalCase(f.Name),
					Unchanged: true,
				})
			}
		}
		entries = append(entries, ae)
	}

	data := struct {
		Package    string
		OldPkg     string
		OldImport  string
		Entries    []autoEntry
	}{
		Package:   versionDir,
		OldPkg:    prevVersionDir,
		OldImport: oldImportPath,
		Entries:   entries,
	}

	tmpl, err := template.New("auto").Parse(autoTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse auto template")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, errors.Wrap(err, "failed to execute auto template")
	}
	return buf.Bytes(), nil
}

func (p *Plugin) generatePostFile(
	goPath string,
	migrations []versionMigration,
	versionDir string,
	prevVersionDir string,
) ([]byte, error) {
	if len(migrations) == 0 {
		return nil, nil
	}

	oldImportPath := resolveGoImportPath(goPath+"/migrations/"+prevVersionDir, "")

	type postEntry struct {
		GoName string
	}

	var entries []postEntry
	for _, vm := range migrations {
		entries = append(entries, postEntry{GoName: vm.Entry.GoName})
	}

	data := struct {
		Package   string
		OldPkg    string
		OldImport string
		Entries   []postEntry
	}{
		Package:   versionDir,
		OldPkg:    prevVersionDir,
		OldImport: oldImportPath,
		Entries:   entries,
	}

	tmpl, err := template.New("post").Parse(postTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse post template")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, errors.Wrap(err, "failed to execute post template")
	}
	return buf.Bytes(), nil
}

const v1Template = `// Code generated by oracle. DO NOT EDIT.

package {{.Package}}
{{- if .HasImports}}

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
{{- end}}
{{range .Entries}}
{{- if .KeyField.KeyIsAlias}}
type Key = {{.KeyField.KeyBaseType}}
{{- else}}
type Key {{.KeyField.KeyBaseType}}
{{- end}}

type {{.GoName}} struct {
{{- range .Fields}}
	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JSONTag}}" msgpack:"{{.JSONTag}}"` + "`" + `
{{- end}}
}

func (s {{.GoName}}) GorpKey() Key { return s.{{.KeyField.FieldGoName}} }

func (s {{.GoName}}) SetOptions() []any { return nil }
{{end}}`

const migrateTemplate = `// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import (
	"github.com/synnaxlabs/x/gorp"
{{- range .Entries}}
{{- range .VersionMigrations}}
	{{.OldPkg}} "{{.OldImport}}"
	{{.NewPkg}} "{{.NewImport}}"
{{- end}}
{{- end}}
)
{{range .Entries}}
func {{.GoName}}Migrations(codec gorp.Codec[{{.GoName}}]) []gorp.Migration {
	return []gorp.Migration{
{{- range .VersionMigrations}}
		gorp.NewTypedMigration[{{.OldPkg}}.{{.GoName}}, {{.NewPkg}}.{{.GoName}}](
			"{{.OldVersion}}_to_{{.NewVersion}}",
			nil, nil,
			{{.NewPkg}}.Auto{{.GoName}}, {{.NewPkg}}.Post{{.GoName}},
		),
{{- end}}
		gorp.NewCodecTransition[{{.KeyType}}, {{.GoName}}](
			"msgpack_to_binary",
			codec,
		),
	}
}
{{end}}`

const autoTemplate = `// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import (
	"context"

	{{.OldPkg}} "{{.OldImport}}"
)
{{range .Entries}}
func Auto{{.GoName}}(_ context.Context, old {{$.OldPkg}}.{{.GoName}}) ({{.GoName}}, error) {
	return {{.GoName}}{
{{- range .Fields}}
{{- if .Unchanged}}
		{{.GoName}}: old.{{.GoName}},
{{- end}}
{{- if .Added}}
		// TODO: {{.GoName}} was added (type: {{.NewType}}). Provide a default value.
		// {{.GoName}}: ???,
{{- end}}
{{- if .Changed}}
		// TODO: {{.GoName}} type changed from {{.OldType}} to {{.NewType}}.
		// {{.GoName}}: convert(old.{{.GoName}}),
{{- end}}
{{- end}}
	}, nil
}
{{- range .Fields}}
{{- if .Removed}}
// NOTE: Field {{.GoName}} was removed (was type: {{.OldType}}).
{{- end}}
{{- end}}
{{end}}`

const postTemplate = `package {{.Package}}

import (
	"context"

	{{.OldPkg}} "{{.OldImport}}"
)
{{range .Entries}}
func Post{{.GoName}}(_ context.Context, _ *{{.GoName}}, _ {{$.OldPkg}}.{{.GoName}}) error {
	// TODO: Implement any post-migration logic for {{.GoName}}.
	return nil
}
{{end}}`
