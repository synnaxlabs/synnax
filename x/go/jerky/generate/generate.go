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

	"github.com/synnaxlabs/x/jerky/detect"
	"github.com/synnaxlabs/x/jerky/parse"
	"github.com/synnaxlabs/x/jerky/state"
	"github.com/synnaxlabs/x/jerky/templates"
	"github.com/synnaxlabs/x/jerky/typemap"
)

// Generator generates all jerky artifacts for a parsed struct.
type Generator struct {
	templates *template.Template
	registry  *typemap.Registry
	outputDir string
}

// NewGenerator creates a new Generator.
func NewGenerator(outputDir string, registry *typemap.Registry) (*Generator, error) {
	tmpl, err := templates.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	if registry == nil {
		registry = typemap.DefaultRegistry()
	}

	return &Generator{
		templates: tmpl,
		registry:  registry,
		outputDir: outputDir,
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

	if exists {
		// Check if struct changed
		currentHash := detect.ComputeStructHash(parsed)
		latestVersion := typeState.LatestVersion()

		if latestVersion != nil && latestVersion.StructHash == currentHash {
			// No changes, regenerate current version only
			version = typeState.CurrentVersion
			migrationType = ""
		} else {
			// Struct changed, increment version
			version = typeState.CurrentVersion + 1
			migrationType = "auto"
		}
	}

	// Build field info map for state
	fields := make(map[string]state.FieldInfo)
	fieldOrder := make([]string, 0, len(parsed.Fields))
	for _, f := range parsed.Fields {
		fields[f.Name] = state.FieldInfo{
			Type: f.GoType.String(),
			Tags: map[string]string{
				"json":    f.Tags.JSON,
				"msgpack": f.Tags.Msgpack,
			},
		}
		fieldOrder = append(fieldOrder, f.Name)
	}

	// Compute hash
	structHash := detect.ComputeStructHash(parsed)
	compositeHash := detect.ComputeCompositeHash(structHash, nil)

	// Update state
	if !exists || migrationType != "" {
		typeState.Package = parsed.PackagePath
		typeState.FieldOrder = fieldOrder
		typeState.AddVersion(state.VersionHistory{
			Version:          version,
			CreatedAt:        time.Now(),
			StructHash:       structHash,
			DependencyHashes: nil,
			CompositeHash:    compositeHash,
			MigrationType:    migrationType,
			Fields:           fields,
		})
		stateFile.SetTypeState(parsed.Name, typeState)
	}

	// Generate proto file
	if err := g.generateProto(parsed, version, typesDir); err != nil {
		return fmt.Errorf("failed to generate proto: %w", err)
	}

	// Generate current.go aliases
	if err := g.generateCurrent(parsed, version, typesDir); err != nil {
		return fmt.Errorf("failed to generate current aliases: %w", err)
	}

	// Generate gorp methods in parent package
	if err := g.generateGorp(parsed, g.outputDir); err != nil {
		return fmt.Errorf("failed to generate gorp methods: %w", err)
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

	filename := filepath.Join(typesDir, fmt.Sprintf("v%d.proto", version))
	return os.WriteFile(filename, buf.Bytes(), 0644)
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

	filename := filepath.Join(typesDir, "current.go")
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
}

func (g *Generator) generateGorp(parsed parse.ParsedStruct, outputDir string) error {
	imports := make(map[string]bool)

	data := GorpTemplateData{
		PackageName: parsed.PackageName,
		TypesImport: parsed.PackagePath + "/types",
		TypeName:    parsed.Name,
	}

	for _, f := range parsed.Fields {
		forwardExpr, backwardExpr, fieldImports := g.getTranslationExprs(f, parsed.PackageName, parsed.PackagePath)

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

	filename := filepath.Join(typesDir, "migrations.go")
	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return err
	}

	// Generate individual hook files for each migration (only if they don't exist)
	for _, migration := range data.Migrations {
		if migration.FromVersion == 0 {
			continue // V0 bootstrap doesn't need a hook
		}
		hookFilename := filepath.Join(typesDir, fmt.Sprintf("v%d_to_v%d.go", migration.FromVersion, migration.ToVersion))
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
func (g *Generator) getTranslationExprs(f parse.ParsedField, parentPkg string, parentPath string) (forward, backward string, imports []string) {
	fieldRef := "m." + f.Name
	pbFieldRef := "pb." + f.Name

	// Check if there's a direct mapping
	if mapping, ok := g.registry.Get(f.GoType.Name); ok {
		forward = strings.ReplaceAll(mapping.ForwardExpr, "{{.Field}}", fieldRef)
		backward = strings.ReplaceAll(mapping.BackwardExpr, "{{.Field}}", pbFieldRef)
		return forward, backward, mapping.NeedsImport
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
			return forward, backward, imports
		}
	}

	// Default: direct copy
	return fieldRef, pbFieldRef, nil
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
