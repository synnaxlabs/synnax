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
		return nil, fmt.Errorf("failed to load templates: %w", err)
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
		return fmt.Errorf("failed to create types directory: %w", err)
	}

	// Load or create state file
	stateFile, err := state.Load(typesDir)
	if err != nil {
		return fmt.Errorf("failed to load state file: %w", err)
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

	// Update state
	if !exists || migrationType != "" {
		typeState.Package = parsed.PackagePath
		typeState.FieldOrder = fieldOrder
		typeState.AddVersion(state.VersionHistory{
			Version:          version,
			CreatedAt:        time.Now(),
			StructHash:       structHash,
			DependencyHashes: preDepHashes,
			CompositeHash:    compositeHash,
			MigrationType:    migrationType,
			Fields:           fields,
		})
	}
	// Always update field numbers in state (in case new fields added)
	stateFile.SetTypeState(parsed.Name, typeState)

	// Generate proto file
	if err := g.generateProto(parsed, version, typesDir); err != nil {
		return fmt.Errorf("failed to generate proto: %w", err)
	}

	// Note: Version snapshot generation is disabled for now.
	// The proto types (v1.pb.go, v2.pb.go) serve as the versioned types,
	// and migrations work directly with those proto types.
	// If native Go snapshots are needed in the future for complex migrations,
	// re-enable: g.generateVersionSnapshot(parsed, version, typesDir)

	// Generate current.go aliases
	if err := g.generateCurrent(parsed, version, typesDir); err != nil {
		return fmt.Errorf("failed to generate current aliases: %w", err)
	}

	// Generate gorp methods in parent package
	if err := g.generateGorp(parsed, g.outputDir); err != nil {
		return fmt.Errorf("failed to generate gorp methods: %w", err)
	}

	// Generate migrator in parent package
	if err := g.generateMigrator(parsed, version, g.outputDir); err != nil {
		return fmt.Errorf("failed to generate migrator: %w", err)
	}

	// Generate migrations.go
	if err := g.generateMigrations(parsed, version, typesDir, typeState); err != nil {
		return fmt.Errorf("failed to generate migrations: %w", err)
	}

	// Save state file
	if err := stateFile.Save(typesDir); err != nil {
		return fmt.Errorf("failed to save state file: %w", err)
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
				// Generate proto import path: package/types/typename_vN.proto
				protoImport := fmt.Sprintf("%s/types/%s_v%d.proto",
					info.PackagePath, toSnakeCase(info.TypeName), info.CurrentVersion)
				imports[protoImport] = true
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

// CurrentTemplateData contains data for current.go template rendering.
type CurrentTemplateData struct {
	TypeName string
	Version  int
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
}

// MigrationData contains data for a single migration.
type MigrationData struct {
	TypeName      string
	FromVersion   int
	ToVersion     int
	CommonFields  []string // Fields present in both versions
	AddedFields   []string // Fields added in ToVersion
	RemovedFields []string // Fields removed from FromVersion
}

func (g *Generator) generateMigrations(parsed parse.ParsedStruct, version int, typesDir string, typeState state.TypeState) error {
	data := MigrationsTemplateData{
		TypeName:       parsed.Name,
		CurrentVersion: version,
	}

	// Add migrations for each version transition
	for i := 0; i < version; i++ {
		migration := MigrationData{
			TypeName:    parsed.Name,
			FromVersion: i,
			ToVersion:   i + 1,
		}

		// Compute field differences for non-bootstrap migrations
		if i > 0 {
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

	// Default: direct copy
	return fieldRef, pbFieldRef, false, nil
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
