// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package generate provides code generation for jerky.
package generate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/jerky/deps"
	"github.com/synnaxlabs/x/jerky/detect"
	"github.com/synnaxlabs/x/jerky/parse"
	"github.com/synnaxlabs/x/jerky/state"
	"github.com/synnaxlabs/x/jerky/templates"
	"github.com/synnaxlabs/x/jerky/typemap"
)

// Generator generates all jerky artifacts for a parsed struct.
type Generator struct {
	templates   *template.Template
	registry    *typemap.Registry
	depRegistry *deps.Registry
	outputDir   string
}

// NewGenerator creates a new Generator.
func NewGenerator(outputDir string, registry *typemap.Registry) (*Generator, error) {
	return NewGeneratorWithDeps(outputDir, registry, nil)
}

// NewGeneratorWithDeps creates a new Generator with dependency tracking.
func NewGeneratorWithDeps(outputDir string, registry *typemap.Registry, depRegistry *deps.Registry) (*Generator, error) {
	tmpl, err := templates.Load()
	if err != nil {
		return nil, errors.Newf("failed to load templates: %w", err)
	}

	if registry == nil {
		registry = typemap.DefaultRegistry()
	}

	if depRegistry == nil {
		depRegistry = deps.NewRegistry()
	}

	return &Generator{
		templates:   tmpl,
		registry:    registry,
		depRegistry: depRegistry,
		outputDir:   outputDir,
	}, nil
}

// Generate generates all artifacts for a parsed struct.
func (g *Generator) Generate(parsed parse.ParsedStruct) error {
	// Create types directory
	typesDir := filepath.Join(g.outputDir, "types")
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		return errors.Newf("failed to create types directory: %w", err)
	}

	// Load or create state file
	stateFile, err := state.Load(typesDir)
	if err != nil {
		return errors.Newf("failed to load state file: %w", err)
	}

	// Determine version
	typeState, exists := stateFile.GetTypeState(parsed.Name)
	version := 1
	migrationType := "bootstrap"

	// Pre-compute dependency hashes for version detection
	preDepHashes := make(map[string]string)
	jerkyDeps := parsed.JerkyDependencies()
	for _, dep := range jerkyDeps {
		if info, ok := g.depRegistry.Get(dep); ok {
			preDepHashes[dep] = fmt.Sprintf("v%d:%s", info.CurrentVersion, info.CompositeHash)
		}
	}

	if exists {
		// Check if struct changed
		currentHash := detect.ComputeStructHash(parsed)
		latestVersion := typeState.LatestVersion()

		if latestVersion != nil && latestVersion.StructHash == currentHash {
			// Struct didn't change, but check if dependencies changed
			depsChanged := false
			if len(preDepHashes) != len(latestVersion.DependencyHashes) {
				depsChanged = true
			} else {
				for k, v := range preDepHashes {
					if latestVersion.DependencyHashes[k] != v {
						depsChanged = true
						break
					}
				}
			}

			if depsChanged {
				// Dependencies changed, increment version
				version = typeState.CurrentVersion + 1
				migrationType = "dependency"
			} else {
				// No changes, regenerate current version only
				version = typeState.CurrentVersion
				migrationType = ""
			}
		} else {
			// Struct changed, increment version
			version = typeState.CurrentVersion + 1
			migrationType = "auto"
		}
	}

	// Assign stable field numbers using state tracking
	// Field numbers are preserved across versions - once assigned, never changed
	fields := make(map[string]state.FieldInfo)
	fieldOrder := make([]string, 0, len(parsed.Fields))
	for i := range parsed.Fields {
		fieldNum := typeState.GetFieldNumber(parsed.Fields[i].Name)
		// Update the parsed field with the stable field number
		parsed.Fields[i].FieldNumber = fieldNum
		fields[parsed.Fields[i].Name] = state.FieldInfo{
			Type:        parsed.Fields[i].GoType.String(),
			FieldNumber: fieldNum,
			Tags: map[string]string{
				"json":    parsed.Fields[i].Tags.JSON,
				"msgpack": parsed.Fields[i].Tags.Msgpack,
			},
		}
		fieldOrder = append(fieldOrder, parsed.Fields[i].Name)
	}

	// Compute hash including dependencies (use preDepHashes computed earlier)
	structHash := detect.ComputeStructHash(parsed)
	compositeHash := detect.ComputeCompositeHash(structHash, preDepHashes)

	// Collect nested type version info for this version
	nestedVersions := g.collectNestedTypeVersions(parsed)

	// Update state
	if !exists || migrationType != "" {
		typeState.Package = parsed.PackagePath
		typeState.FieldOrder = fieldOrder
		typeState.IsEmbedded = parsed.IsEmbedded
		typeState.AddVersion(state.VersionHistory{
			Version:            version,
			CreatedAt:          time.Now(),
			StructHash:         structHash,
			DependencyHashes:   preDepHashes,
			CompositeHash:      compositeHash,
			MigrationType:      migrationType,
			Fields:             fields,
			NestedTypeVersions: nestedVersions,
		})
	}
	// Always update field numbers and embedded status in state
	typeState.IsEmbedded = parsed.IsEmbedded
	stateFile.SetTypeState(parsed.Name, typeState)

	// Generate proto file
	if err := g.generateProto(parsed, version, typesDir); err != nil {
		return errors.Newf("failed to generate proto: %w", err)
	}

	// Generate v0 struct for bootstrap migration (only on first run)
	if migrationType == "bootstrap" {
		if err := g.generateV0(parsed, typesDir); err != nil {
			return errors.Newf("failed to generate v0 struct: %w", err)
		}
	}

	// Note: Version snapshot generation is disabled for now.
	// The proto types (v1.pb.go, v2.pb.go) serve as the versioned types,
	// and migrations work directly with those proto types.
	// If native Go snapshots are needed in the future for complex migrations,
	// re-enable: g.generateVersionSnapshot(parsed, version, typesDir)

	// Generate current.go aliases
	if err := g.generateCurrent(parsed, version, typesDir); err != nil {
		return errors.Newf("failed to generate current aliases: %w", err)
	}

	if parsed.IsEmbedded {
		// Embedded types: generate translation functions in parent package
		// (not types/ to avoid circular imports)
		if err := g.generateTranslate(parsed, g.outputDir); err != nil {
			return errors.Newf("failed to generate translate functions: %w", err)
		}
	} else {
		// Storage types: generate gorp methods and migrator in parent package
		if err := g.generateGorp(parsed, g.outputDir); err != nil {
			return errors.Newf("failed to generate gorp methods: %w", err)
		}

		if err := g.generateMigrator(parsed, version, g.outputDir); err != nil {
			return errors.Newf("failed to generate migrator: %w", err)
		}
	}

	// Generate migration files based on type:
	// - Storage types: migrations.go (byte-to-byte for data on disk)
	// - Embedded types: struct_migrate.go (struct-to-struct for nested types)
	if parsed.IsEmbedded {
		if err := g.generateStructMigrate(parsed, version, typesDir, typeState); err != nil {
			return errors.Newf("failed to generate struct migrations: %w", err)
		}
	} else {
		if err := g.generateMigrations(parsed, version, typesDir, typeState); err != nil {
			return errors.Newf("failed to generate migrations: %w", err)
		}
	}

	// Save state file
	if err := stateFile.Save(typesDir); err != nil {
		return errors.Newf("failed to save state file: %w", err)
	}

	return nil
}

// ProtoTemplateData contains data for proto template rendering.
type ProtoTemplateData struct {
	ProtoPackage string
	GoPackage    string
	Imports      []string
	TypeName     string
	Version      int
	Fields       []ProtoFieldData
}

// ProtoFieldData contains data for a single proto field.
type ProtoFieldData struct {
	ProtoType   string
	ProtoName   string
	FieldNumber int
}

func (g *Generator) generateProto(parsed parse.ParsedStruct, version int, typesDir string) error {
	data := ProtoTemplateData{
		ProtoPackage: parsed.PackageName + ".types",
		GoPackage:    parsed.PackagePath + "/types",
		TypeName:     parsed.Name,
		Version:      version,
	}

	// Collect proto imports from jerky dependencies
	protoImports := make(map[string]bool)
	g.collectJerkyImports(parsed.Fields, protoImports)
	for imp := range protoImports {
		data.Imports = append(data.Imports, imp)
	}

	for _, f := range parsed.Fields {
		protoType, _ := g.getProtoType(f.GoType)
		protoName := toSnakeCase(f.Name)

		data.Fields = append(data.Fields, ProtoFieldData{
			ProtoType:   protoType,
			ProtoName:   protoName,
			FieldNumber: f.FieldNumber,
		})
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "proto.tmpl", data); err != nil {
		return err
	}

	filename := filepath.Join(typesDir, fmt.Sprintf("%s_v%d.proto", toSnakeCase(parsed.Name), version))
	return os.WriteFile(filename, buf.Bytes(), 0644)
}

// collectJerkyImports recursively collects proto imports for jerky-managed field types.
func (g *Generator) collectJerkyImports(fields []parse.ParsedField, imports map[string]bool) {
	var collectFromType func(t parse.GoType)
	collectFromType = func(t parse.GoType) {
		if t.IsJerky && t.PackagePath != "" {
			tName := typeName(t.Name)
			if info, ok := g.depRegistry.GetByPackageAndType(t.PackagePath, tName); ok {
				// Generate proto import path using the StateDir (file system path)
				protoPath := computeProtoImportPath(info.StateDir, info.TypeName, info.CurrentVersion)
				if protoPath != "" {
					imports[protoPath] = true
				}
			}
		}
		if t.Elem != nil {
			collectFromType(*t.Elem)
		}
		if t.Key != nil {
			collectFromType(*t.Key)
		}
	}

	for _, f := range fields {
		collectFromType(f.GoType)
	}
}

// computeProtoImportPath computes the proto import path from the state directory path.
// It finds the repository root and returns the relative path.
func computeProtoImportPath(stateDir, typeName string, version int) string {
	// Find the repository root by looking for buf.yaml
	repoRoot := findRepoRoot(stateDir)
	if repoRoot == "" {
		return ""
	}

	// Compute relative path from repo root to the proto file
	relPath, err := filepath.Rel(repoRoot, stateDir)
	if err != nil {
		return ""
	}

	// Construct the proto import path
	return filepath.ToSlash(filepath.Join(relPath, fmt.Sprintf("%s_v%d.proto", toSnakeCase(typeName), version)))
}

// findRepoRoot walks up the directory tree to find the repository root (buf.yaml location).
func findRepoRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "buf.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// CurrentTemplateData contains data for current.go template rendering.
type CurrentTemplateData struct {
	TypeName string
	Version  int
}

// V0TemplateData contains data for v0.go template rendering.
type V0TemplateData struct {
	TypeName string
	Fields   []V0FieldData
}

// V0FieldData contains data for a single v0 field.
type V0FieldData struct {
	Name       string // Go field name
	GoType     string // Proto-compatible Go type (string, int64, etc.)
	MsgpackTag string // Msgpack tag value
}

func (g *Generator) generateCurrent(parsed parse.ParsedStruct, version int, typesDir string) error {
	data := CurrentTemplateData{
		TypeName: parsed.Name,
		Version:  version,
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "current.go.tmpl", data); err != nil {
		return err
	}

	filename := filepath.Join(typesDir, fmt.Sprintf("%s_current.go", toSnakeCase(parsed.Name)))
	return os.WriteFile(filename, buf.Bytes(), 0644)
}

// generateV0 generates the v0 struct for msgpack bootstrap migration.
// This is only generated once during the initial jerky adoption (bootstrap).
func (g *Generator) generateV0(parsed parse.ParsedStruct, typesDir string) error {
	data := V0TemplateData{
		TypeName: parsed.Name,
	}

	for _, f := range parsed.Fields {
		// Get the proto-compatible Go type
		goType := g.getV0GoType(f.GoType)

		// Get msgpack tag (use json tag as fallback, which is typical for msgpack)
		msgpackTag := f.Tags.Msgpack
		if msgpackTag == "" {
			msgpackTag = f.Tags.JSON
		}
		if msgpackTag == "" {
			msgpackTag = toSnakeCase(f.Name)
		}
		// Strip options like omitempty
		if idx := strings.Index(msgpackTag, ","); idx != -1 {
			msgpackTag = msgpackTag[:idx]
		}

		data.Fields = append(data.Fields, V0FieldData{
			Name:       f.Name,
			GoType:     goType,
			MsgpackTag: msgpackTag,
		})
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "v0.go.tmpl", data); err != nil {
		return err
	}

	filename := filepath.Join(typesDir, fmt.Sprintf("%s_v0.go", toSnakeCase(parsed.Name)))
	return os.WriteFile(filename, buf.Bytes(), 0644)
}

// getV0GoType returns the proto-compatible Go type for use in v0 struct.
// This matches what proto uses, allowing direct field copying in migrations.
func (g *Generator) getV0GoType(goType parse.GoType) string {
	// Check if this is a jerky-managed type
	if goType.IsJerky && goType.PackagePath != "" {
		tName := typeName(goType.Name)
		if info, ok := g.depRegistry.GetByPackageAndType(goType.PackagePath, tName); ok {
			return fmt.Sprintf("*%sV%d", info.TypeName, info.CurrentVersion)
		}
	}

	// Check if there's a direct mapping
	if mapping, ok := g.registry.Get(goType.Name); ok {
		return protoToGoType(mapping.ProtoType)
	}

	// Check underlying type for named types
	if goType.Underlying != nil {
		if mapping, ok := g.registry.Get(goType.Underlying.Name); ok {
			return protoToGoType(mapping.ProtoType)
		}
	}

	// Handle slices
	if goType.Kind == parse.KindSlice && goType.Elem != nil {
		elemType := g.getV0GoType(*goType.Elem)
		return "[]" + elemType
	}

	// Default to string for unknown types
	return "string"
}

// protoToGoType converts a proto type to its Go equivalent.
func protoToGoType(protoType string) string {
	switch protoType {
	case "string":
		return "string"
	case "int32", "sint32", "sfixed32":
		return "int32"
	case "int64", "sint64", "sfixed64":
		return "int64"
	case "uint32", "fixed32":
		return "uint32"
	case "uint64", "fixed64":
		return "uint64"
	case "double":
		return "float64"
	case "float":
		return "float32"
	case "bool":
		return "bool"
	case "bytes":
		return "[]byte"
	default:
		return "string"
	}
}

// GorpTemplateData contains data for gorp.go template rendering.
type GorpTemplateData struct {
	PackageName string
	TypesImport string
	TypeName    string
	Imports     []string
	Fields      []GorpFieldData
}

// GorpFieldData contains data for a single field's gorp conversion.
type GorpFieldData struct {
	Name         string
	ProtoName    string
	ForwardExpr  string
	BackwardExpr string
	CanFail      bool // True if backward translation can return an error
}

func (g *Generator) generateGorp(parsed parse.ParsedStruct, outputDir string) error {
	imports := make(map[string]bool)

	data := GorpTemplateData{
		PackageName: parsed.PackageName,
		TypesImport: parsed.PackagePath + "/types",
		TypeName:    parsed.Name,
	}

	for _, f := range parsed.Fields {
		forwardExpr, backwardExpr, canFail, fieldImports := g.getTranslationExprs(f, parsed.PackageName, parsed.PackagePath)

		for _, imp := range fieldImports {
			imports[imp] = true
		}

		// Convert Go field name to proto field name (PascalCase for generated Go proto code)
		protoName := f.Name

		data.Fields = append(data.Fields, GorpFieldData{
			Name:         f.Name,
			ProtoName:    protoName,
			ForwardExpr:  forwardExpr,
			BackwardExpr: backwardExpr,
			CanFail:      canFail,
		})
	}

	// Convert imports map to slice
	for imp := range imports {
		data.Imports = append(data.Imports, imp)
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "gorp.go.tmpl", data); err != nil {
		return err
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("%s_gorp.go", toSnakeCase(parsed.Name)))
	return os.WriteFile(filename, buf.Bytes(), 0644)
}

// TranslateTemplateData contains data for translate.go template rendering.
type TranslateTemplateData struct {
	PackageName string
	TypesImport string
	TypeName    string
	Imports     []string
	Fields      []TranslateFieldData
	HasErrors   bool // True if any backward translation can fail
}

// TranslateFieldData contains data for a single field's translation.
type TranslateFieldData struct {
	Name         string
	ProtoName    string
	ForwardExpr  string
	BackwardExpr string
	CanFail      bool
}

func (g *Generator) generateTranslate(parsed parse.ParsedStruct, outputDir string) error {
	imports := make(map[string]bool)
	hasErrors := false

	data := TranslateTemplateData{
		PackageName: parsed.PackageName,
		TypesImport: parsed.PackagePath + "/types",
		TypeName:    parsed.Name,
	}

	for _, f := range parsed.Fields {
		forwardExpr, backwardExpr, canFail, fieldImports := g.getTranslationExprs(f, parsed.PackageName, parsed.PackagePath)

		for _, imp := range fieldImports {
			imports[imp] = true
		}

		if canFail {
			hasErrors = true
		}

		data.Fields = append(data.Fields, TranslateFieldData{
			Name:         f.Name,
			ProtoName:    f.Name,
			ForwardExpr:  forwardExpr,
			BackwardExpr: backwardExpr,
			CanFail:      canFail,
		})
	}

	data.HasErrors = hasErrors

	// Convert imports map to slice
	for imp := range imports {
		data.Imports = append(data.Imports, imp)
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "translate.go.tmpl", data); err != nil {
		return err
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("%s_translate.go", toSnakeCase(parsed.Name)))
	return os.WriteFile(filename, buf.Bytes(), 0644)
}

// MigratorTemplateData contains data for migrator.go template rendering.
type MigratorTemplateData struct {
	PackageName    string
	TypesImport    string
	TypeName       string
	CurrentVersion int
	Imports        []string
}

func (g *Generator) generateMigrator(parsed parse.ParsedStruct, version int, outputDir string) error {
	data := MigratorTemplateData{
		PackageName:    parsed.PackageName,
		TypesImport:    parsed.PackagePath + "/types",
		TypeName:       parsed.Name,
		CurrentVersion: version,
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "migrator.go.tmpl", data); err != nil {
		return err
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("%s_migrator.go", toSnakeCase(parsed.Name)))
	return os.WriteFile(filename, buf.Bytes(), 0644)
}

// MigrationsTemplateData contains data for migrations.go template rendering.
type MigrationsTemplateData struct {
	TypeName       string
	CurrentVersion int
	Migrations     []MigrationData
	// Parent package info for v0 bootstrap migration (no longer used but kept for compatibility)
	ParentPkg     string // Package name (e.g., "example")
	ParentPkgPath string // Full import path
}

// MigrationData contains data for a single migration.
type MigrationData struct {
	TypeName      string
	FromVersion   int
	ToVersion     int
	CommonFields  []string // Fields present in both versions (for backward compat)
	AddedFields   []string // Fields added in ToVersion
	RemovedFields []string // Fields removed from FromVersion
	// V1Fields contains field translations for v0→v1 bootstrap migration
	V1Fields []BootstrapFieldData
	// Fields contains per-field migration info for non-bootstrap migrations
	Fields []MigrationFieldData
}

// BootstrapFieldData contains data for v0→v1 field translation.
type BootstrapFieldData struct {
	Name string // Field name (e.g., "Key")
}

// MigrationFieldData contains per-field migration info for template rendering.
type MigrationFieldData struct {
	Name             string // Field name
	IsJerky          bool   // True if this is a jerky-managed type
	JerkyTypeName    string // e.g., "Address"
	FromJerkyVersion int    // Nested type version in FromVersion
	ToJerkyVersion   int    // Nested type version in ToVersion
	VersionChanged   bool   // True if nested type version changed
	IsSlice          bool   // True for []JerkyType
	IsMap            bool   // True for map[K]JerkyType
	MapKeyType       string // Key type for maps (e.g., "string")
}

// StructMigrateTemplateData contains data for struct_migrate.go.tmpl rendering.
type StructMigrateTemplateData struct {
	TypeName       string
	CurrentVersion int
	Migrations     []StructMigrationData
}

// StructMigrationData contains data for a single struct migration.
type StructMigrationData struct {
	TypeName      string
	FromVersion   int
	ToVersion     int
	CommonFields  []string
	AddedFields   []string
	RemovedFields []string
}

func (g *Generator) generateMigrations(parsed parse.ParsedStruct, version int, typesDir string, typeState state.TypeState) error {
	data := MigrationsTemplateData{
		TypeName:       parsed.Name,
		CurrentVersion: version,
		ParentPkg:      parsed.PackageName,
		ParentPkgPath:  parsed.PackagePath,
	}

	// Add migrations for each version transition
	for i := 0; i < version; i++ {
		migration := MigrationData{
			TypeName:    parsed.Name,
			FromVersion: i,
			ToVersion:   i + 1,
		}

		// For v0→v1 bootstrap migration, collect field names
		if i == 0 {
			v1Fields := typeState.GetVersion(1)
			if v1Fields != nil {
				for _, fieldName := range typeState.FieldOrder {
					if _, exists := v1Fields.Fields[fieldName]; exists {
						migration.V1Fields = append(migration.V1Fields, BootstrapFieldData{
							Name: fieldName,
						})
					}
				}
			}
		}

		// Compute field differences for non-bootstrap migrations
		if i > 0 {
			fromVH := typeState.GetVersion(i)
			toVH := typeState.GetVersion(i + 1)
			fromFields := getVersionFields(typeState, i)
			toFields := getVersionFields(typeState, i+1)

			for field := range toFields {
				if _, exists := fromFields[field]; exists {
					migration.CommonFields = append(migration.CommonFields, field)
				} else {
					migration.AddedFields = append(migration.AddedFields, field)
				}
			}
			for field := range fromFields {
				if _, exists := toFields[field]; !exists {
					migration.RemovedFields = append(migration.RemovedFields, field)
				}
			}

			// Compute per-field migration info (including jerky nested type info)
			if fromVH != nil && toVH != nil {
				migration.Fields = g.computeMigrationFields(typeState, fromVH, toVH)
			}
		}

		data.Migrations = append(data.Migrations, migration)
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "migrations.go.tmpl", data); err != nil {
		return err
	}

	filename := filepath.Join(typesDir, fmt.Sprintf("%s_migrations.go", toSnakeCase(parsed.Name)))
	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return err
	}

	// Generate individual hook files for each migration (only if they don't exist)
	for _, migration := range data.Migrations {
		if migration.FromVersion == 0 {
			continue // V0 bootstrap doesn't need a hook
		}
		hookFilename := filepath.Join(typesDir, fmt.Sprintf("%s_v%d_to_v%d.go", toSnakeCase(parsed.Name), migration.FromVersion, migration.ToVersion))
		if _, err := os.Stat(hookFilename); os.IsNotExist(err) {
			var hookBuf bytes.Buffer
			if err := g.templates.ExecuteTemplate(&hookBuf, "migrate_hook.go.tmpl", migration); err != nil {
				return err
			}
			if err := os.WriteFile(hookFilename, hookBuf.Bytes(), 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

// generateStructMigrate generates struct-to-struct migration functions for embedded types.
func (g *Generator) generateStructMigrate(parsed parse.ParsedStruct, version int, typesDir string, typeState state.TypeState) error {
	data := StructMigrateTemplateData{
		TypeName:       parsed.Name,
		CurrentVersion: version,
	}

	// Build migration data for each version transition (excluding v0 bootstrap)
	for i := 1; i < version; i++ {
		fromVH := typeState.GetVersion(i)
		toVH := typeState.GetVersion(i + 1)

		if fromVH == nil || toVH == nil {
			continue
		}

		// Get field differences
		fromFields := getVersionFields(typeState, i)
		toFields := getVersionFields(typeState, i+1)

		var commonFields, addedFields, removedFields []string
		for field := range toFields {
			if fromFields[field] {
				commonFields = append(commonFields, field)
			} else {
				addedFields = append(addedFields, field)
			}
		}
		for field := range fromFields {
			if !toFields[field] {
				removedFields = append(removedFields, field)
			}
		}

		data.Migrations = append(data.Migrations, StructMigrationData{
			TypeName:      parsed.Name,
			FromVersion:   i,
			ToVersion:     i + 1,
			CommonFields:  commonFields,
			AddedFields:   addedFields,
			RemovedFields: removedFields,
		})
	}

	// Generate struct_migrate.go
	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "struct_migrate.go.tmpl", data); err != nil {
		return err
	}
	filename := filepath.Join(typesDir, fmt.Sprintf("%s_struct_migrate.go", toSnakeCase(parsed.Name)))
	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return err
	}

	// Generate hook files for each migration (if they don't exist)
	for _, migration := range data.Migrations {
		hookFilename := filepath.Join(typesDir, fmt.Sprintf("%s_v%d_to_v%d.go",
			toSnakeCase(parsed.Name), migration.FromVersion, migration.ToVersion))
		if _, err := os.Stat(hookFilename); os.IsNotExist(err) {
			var hookBuf bytes.Buffer
			if err := g.templates.ExecuteTemplate(&hookBuf, "migrate_hook.go.tmpl", migration); err != nil {
				return err
			}
			if err := os.WriteFile(hookFilename, hookBuf.Bytes(), 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

// getVersionFields returns a map of field names for a specific version.
func getVersionFields(typeState state.TypeState, version int) map[string]bool {
	fields := make(map[string]bool)
	for _, vh := range typeState.History {
		if vh.Version == version {
			for fieldName := range vh.Fields {
				fields[fieldName] = true
			}
			break
		}
	}
	return fields
}

// collectNestedTypeVersions collects nested type version info for all jerky-managed fields.
func (g *Generator) collectNestedTypeVersions(parsed parse.ParsedStruct) map[string]state.NestedTypeInfo {
	nestedVersions := make(map[string]state.NestedTypeInfo)
	for _, f := range parsed.Fields {
		// Direct jerky field (e.g., Address Address)
		if f.GoType.IsJerky {
			tName := typeName(f.GoType.Name)
			if info, ok := g.depRegistry.GetByPackageAndType(f.GoType.PackagePath, tName); ok {
				nestedVersions[f.Name] = state.NestedTypeInfo{
					TypeName: info.TypeName,
					Version:  info.CurrentVersion,
				}
			}
		}
		// Slice of jerky type (e.g., Addresses []Address)
		if f.GoType.Kind == parse.KindSlice && f.GoType.Elem != nil && f.GoType.Elem.IsJerky {
			tName := typeName(f.GoType.Elem.Name)
			if info, ok := g.depRegistry.GetByPackageAndType(f.GoType.Elem.PackagePath, tName); ok {
				nestedVersions[f.Name] = state.NestedTypeInfo{
					TypeName: info.TypeName,
					Version:  info.CurrentVersion,
					IsSlice:  true,
				}
			}
		}
		// Map with jerky value type (e.g., AddressesByName map[string]Address)
		if f.GoType.Kind == parse.KindMap && f.GoType.Elem != nil && f.GoType.Elem.IsJerky {
			tName := typeName(f.GoType.Elem.Name)
			if info, ok := g.depRegistry.GetByPackageAndType(f.GoType.Elem.PackagePath, tName); ok {
				keyType := "string"
				if f.GoType.Key != nil {
					keyType = f.GoType.Key.Name
				}
				nestedVersions[f.Name] = state.NestedTypeInfo{
					TypeName:   info.TypeName,
					Version:    info.CurrentVersion,
					IsMap:      true,
					MapKeyType: keyType,
				}
			}
		}
	}
	return nestedVersions
}

// computeMigrationFields computes per-field migration info for a version transition.
func (g *Generator) computeMigrationFields(typeState state.TypeState, fromVH, toVH *state.VersionHistory) []MigrationFieldData {
	var fields []MigrationFieldData
	for _, fieldName := range typeState.FieldOrder {
		_, inFrom := fromVH.Fields[fieldName]
		_, inTo := toVH.Fields[fieldName]
		// Only include fields that exist in both versions (common fields)
		if !inFrom || !inTo {
			continue
		}
		fieldData := MigrationFieldData{Name: fieldName}
		// Check for jerky nested type
		fromNested := fromVH.NestedTypeVersions[fieldName]
		toNested := toVH.NestedTypeVersions[fieldName]
		if fromNested.TypeName != "" && toNested.TypeName != "" {
			fieldData.IsJerky = true
			fieldData.JerkyTypeName = fromNested.TypeName
			fieldData.FromJerkyVersion = fromNested.Version
			fieldData.ToJerkyVersion = toNested.Version
			fieldData.VersionChanged = fromNested.Version != toNested.Version
			fieldData.IsSlice = fromNested.IsSlice || toNested.IsSlice
			fieldData.IsMap = fromNested.IsMap || toNested.IsMap
			if fromNested.MapKeyType != "" {
				fieldData.MapKeyType = fromNested.MapKeyType
			} else {
				fieldData.MapKeyType = toNested.MapKeyType
			}
		}
		fields = append(fields, fieldData)
	}
	return fields
}

// getProtoType returns the proto type for a Go type.
func (g *Generator) getProtoType(goType parse.GoType) (string, bool) {
	// Check if this is a jerky-managed type
	if goType.IsJerky && goType.PackagePath != "" {
		tName := typeName(goType.Name)
		if info, ok := g.depRegistry.GetByPackageAndType(goType.PackagePath, tName); ok {
			// Return fully qualified proto type: package.types.TypeNameVN
			return fmt.Sprintf("%s.types.%sV%d", info.PackageName, info.TypeName, info.CurrentVersion), false
		}
	}

	// Check if there's a direct mapping
	if mapping, ok := g.registry.Get(goType.Name); ok {
		return mapping.ProtoType, mapping.CanFail
	}

	// Check underlying type for named types
	if goType.Underlying != nil {
		if mapping, ok := g.registry.Get(goType.Underlying.Name); ok {
			return mapping.ProtoType, mapping.CanFail
		}
	}

	// Handle slices
	if goType.Kind == parse.KindSlice && goType.Elem != nil {
		elemProto, _ := g.getProtoType(*goType.Elem)
		return "repeated " + elemProto, false
	}

	// Handle maps
	if goType.Kind == parse.KindMap && goType.Key != nil && goType.Elem != nil {
		keyProto, _ := g.getProtoType(*goType.Key)
		valProto, _ := g.getProtoType(*goType.Elem)
		return fmt.Sprintf("map<%s, %s>", keyProto, valProto), false
	}

	// Default to string for unknown types
	return "string", false
}

// getTranslationExprs returns forward and backward translation expressions for a field.
// For gorp: forward uses m.Field (domain), backward uses pb.Field (proto).
// canFail indicates if the backward translation can return an error.
func (g *Generator) getTranslationExprs(f parse.ParsedField, parentPkg string, parentPath string) (forward, backward string, canFail bool, imports []string) {
	fieldRef := "m." + f.Name
	pbFieldRef := "pb." + f.Name

	// Check if there's a direct mapping
	if mapping, ok := g.registry.Get(f.GoType.Name); ok {
		forward = strings.ReplaceAll(mapping.ForwardExpr, "{{.Field}}", fieldRef)
		backward = strings.ReplaceAll(mapping.BackwardExpr, "{{.Field}}", pbFieldRef)
		return forward, backward, mapping.CanFail, mapping.NeedsImport
	}

	// Check underlying type for named types (e.g., type Key uint32)
	if f.GoType.Underlying != nil {
		if mapping, ok := g.registry.Get(f.GoType.Underlying.Name); ok {
			// For forward: cast from named type to underlying, then apply mapping
			// e.g., UserID -> uint32 -> uint32 (proto)
			underlyingRef := fmt.Sprintf("%s(%s)", f.GoType.Underlying.Name, fieldRef)
			forward = strings.ReplaceAll(mapping.ForwardExpr, "{{.Field}}", underlyingRef)

			// Determine package for backward cast
			// Local types (same package) don't need a package prefix
			pkgPrefix := f.GoType.PackageName
			pkgPath := f.GoType.PackagePath
			if pkgPrefix == "" || pkgPath == "" || pkgPath == parentPath {
				// Local type - just use the type name, no import needed
				backward = fmt.Sprintf("%s(%s)",
					typeName(f.GoType.Name),
					strings.ReplaceAll(mapping.BackwardExpr, "{{.Field}}", pbFieldRef))
				imports = append(imports, mapping.NeedsImport...)
			} else {
				// External type - need package prefix and import
				backward = fmt.Sprintf("%s.%s(%s)",
					pkgPrefix, typeName(f.GoType.Name),
					strings.ReplaceAll(mapping.BackwardExpr, "{{.Field}}", pbFieldRef))
				imports = append(imports, pkgPath)
				imports = append(imports, mapping.NeedsImport...)
			}
			return forward, backward, mapping.CanFail, imports
		}
	}

	// Check if this is a jerky-managed type (direct field)
	if f.GoType.IsJerky {
		return g.getJerkyTranslationExprs(f.GoType, fieldRef, pbFieldRef, parentPath)
	}

	// Handle slices that may contain jerky types
	if f.GoType.Kind == parse.KindSlice && f.GoType.Elem != nil {
		if f.GoType.Elem.IsJerky {
			return g.getJerkySliceTranslationExprs(f.GoType.Elem, fieldRef, pbFieldRef, parentPath)
		}
	}

	// Handle maps that may contain jerky types as values
	if f.GoType.Kind == parse.KindMap && f.GoType.Elem != nil {
		if f.GoType.Elem.IsJerky {
			return g.getJerkyMapTranslationExprs(f.GoType.Key, f.GoType.Elem, fieldRef, pbFieldRef, parentPath)
		}
	}

	// Default: direct copy
	return fieldRef, pbFieldRef, false, nil
}

// getJerkyTranslationExprs returns translation expressions for a jerky-managed type field.
func (g *Generator) getJerkyTranslationExprs(goType parse.GoType, fieldRef, pbFieldRef, parentPath string) (forward, backward string, canFail bool, imports []string) {
	tName := typeName(goType.Name)

	// Determine if this is a local or external type
	pkgPrefix := goType.PackageName
	pkgPath := goType.PackagePath

	if pkgPrefix == "" || pkgPath == "" || pkgPath == parentPath {
		// Local type - call translation functions directly
		forward = fmt.Sprintf("%sToProto(%s)", tName, fieldRef)
		backward = fmt.Sprintf("%sFromProto(%s)", tName, pbFieldRef)
	} else {
		// External type - need package prefix and import
		forward = fmt.Sprintf("%s.%sToProto(%s)", pkgPrefix, tName, fieldRef)
		backward = fmt.Sprintf("%s.%sFromProto(%s)", pkgPrefix, tName, pbFieldRef)
		imports = append(imports, pkgPath)
	}

	// Jerky translation functions may return errors (FromProto variant)
	// For now, we'll assume they don't fail - this can be enhanced based on the type's info
	// from the dependency registry
	if info, ok := g.depRegistry.GetByPackageAndType(pkgPath, tName); ok {
		_ = info // Could check if the type has error-prone conversions
	}

	return forward, backward, false, imports
}

// getJerkySliceTranslationExprs returns translation expressions for a slice of jerky-managed types.
func (g *Generator) getJerkySliceTranslationExprs(elemType *parse.GoType, fieldRef, pbFieldRef, parentPath string) (forward, backward string, canFail bool, imports []string) {
	tName := typeName(elemType.Name)

	// Determine if this is a local or external type
	pkgPrefix := elemType.PackageName
	pkgPath := elemType.PackagePath

	var toProtoFunc, fromProtoFunc string
	if pkgPrefix == "" || pkgPath == "" || pkgPath == parentPath {
		toProtoFunc = fmt.Sprintf("%sToProto", tName)
		fromProtoFunc = fmt.Sprintf("%sFromProto", tName)
	} else {
		toProtoFunc = fmt.Sprintf("%s.%sToProto", pkgPrefix, tName)
		fromProtoFunc = fmt.Sprintf("%s.%sFromProto", pkgPrefix, tName)
		imports = append(imports, pkgPath)
	}

	// Generate slice conversion using functional approach
	// Forward: convert []Domain to []*types.Proto
	forward = fmt.Sprintf("func() []*types.%s { result := make([]*types.%s, len(%s)); for i, v := range %s { result[i] = %s(v) }; return result }()",
		tName, tName, fieldRef, fieldRef, toProtoFunc)

	// Backward: convert []*types.Proto to []Domain
	backward = fmt.Sprintf("func() []%s { result := make([]%s, len(%s)); for i, v := range %s { result[i] = %s(v) }; return result }()",
		tName, tName, pbFieldRef, pbFieldRef, fromProtoFunc)

	return forward, backward, false, imports
}

// getJerkyMapTranslationExprs returns translation expressions for a map with jerky-managed values.
func (g *Generator) getJerkyMapTranslationExprs(keyType, valueType *parse.GoType, fieldRef, pbFieldRef, parentPath string) (forward, backward string, canFail bool, imports []string) {
	tName := typeName(valueType.Name)

	// Get key type for the map
	keyTypeName := "string" // default
	if keyType != nil {
		keyTypeName = keyType.Name
	}

	// Determine if this is a local or external type
	pkgPrefix := valueType.PackageName
	pkgPath := valueType.PackagePath

	var toProtoFunc, fromProtoFunc string
	if pkgPrefix == "" || pkgPath == "" || pkgPath == parentPath {
		toProtoFunc = fmt.Sprintf("%sToProto", tName)
		fromProtoFunc = fmt.Sprintf("%sFromProto", tName)
	} else {
		toProtoFunc = fmt.Sprintf("%s.%sToProto", pkgPrefix, tName)
		fromProtoFunc = fmt.Sprintf("%s.%sFromProto", pkgPrefix, tName)
		imports = append(imports, pkgPath)
	}

	// Generate map conversion using functional approach
	// Forward: convert map[K]Domain to map[K]*types.Proto
	forward = fmt.Sprintf("func() map[%s]*types.%s { result := make(map[%s]*types.%s, len(%s)); for k, v := range %s { result[k] = %s(v) }; return result }()",
		keyTypeName, tName, keyTypeName, tName, fieldRef, fieldRef, toProtoFunc)

	// Backward: convert map[K]*types.Proto to map[K]Domain
	backward = fmt.Sprintf("func() map[%s]%s { result := make(map[%s]%s, len(%s)); for k, v := range %s { result[k] = %s(v) }; return result }()",
		keyTypeName, tName, keyTypeName, tName, pbFieldRef, pbFieldRef, fromProtoFunc)

	return forward, backward, false, imports
}

// toSnakeCase converts PascalCase to snake_case.
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// typeName extracts the type name without package prefix.
func typeName(fullName string) string {
	parts := strings.Split(fullName, ".")
	return parts[len(parts)-1]
}
