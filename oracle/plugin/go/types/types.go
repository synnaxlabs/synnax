// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/doc"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
)

// Plugin generates Go struct type definitions from Oracle schemas.
type Plugin struct{ Options Options }

// Options configures the Go types plugin.
type Options struct {
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		FileNamePattern: "types.gen.go",
	}
}

// New creates a new Go types plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "go/types" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"go"} }

// Requires returns plugin dependencies (none for this plugin).
func (p *Plugin) Requires() []string { return nil }

// Check verifies generated files are up-to-date.
func (p *Plugin) Check(req *plugin.Request) error {
	// TODO: Implement freshness check
	return nil
}

// Generate produces Go type definition files from the analyzed schemas.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	outputStructs := make(map[string][]*resolution.StructEntry)
	for _, entry := range req.Resolutions.AllStructs() {
		if outputPath := output.GetPath(entry, "go"); outputPath != "" {
			// Validate output path if RepoRoot is available
			if req.RepoRoot != "" {
				if err := req.ValidateOutputPath(outputPath); err != nil {
					return nil, fmt.Errorf("invalid output path for struct %s: %w", entry.Name, err)
				}
			}
			outputStructs[outputPath] = append(outputStructs[outputPath], entry)
		}
	}

	for outputPath, structs := range outputStructs {
		content, err := p.generateFile(outputPath, structs)
		if err != nil {
			return nil, fmt.Errorf("failed to generate %s: %w", outputPath, err)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
			Content: content,
		})
	}

	return resp, nil
}

// generateFile generates the Go source file for a set of structs.
func (p *Plugin) generateFile(outputPath string, structs []*resolution.StructEntry) ([]byte, error) {
	sort.Slice(structs, func(i, j int) bool { return structs[i].Name < structs[j].Name })

	data := &templateData{
		Package: derivePackageName(outputPath),
		Structs: make([]structData, 0, len(structs)),
		Imports: make(map[string]bool),
	}

	for _, entry := range structs {
		data.Structs = append(data.Structs, p.processStruct(entry, data))
	}

	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// derivePackageName extracts the package name from the output path.
// "core/pkg/service/user" -> "user"
func derivePackageName(outputPath string) string {
	return filepath.Base(outputPath)
}

// processStruct converts a StructEntry to template data.
func (p *Plugin) processStruct(entry *resolution.StructEntry, data *templateData) structData {
	sd := structData{
		Name:   entry.Name,
		Doc:    doc.Get(entry.Domains),
		Fields: make([]fieldData, 0, len(entry.Fields)),
	}
	for _, field := range entry.Fields {
		sd.Fields = append(sd.Fields, p.processField(field, data))
	}
	return sd
}

// processField converts a FieldEntry to template data.
func (p *Plugin) processField(field *resolution.FieldEntry, data *templateData) fieldData {
	goType := p.typeToGo(field.TypeRef, data)

	return fieldData{
		GoName:     toPascalCase(field.Name),
		GoType:     goType,
		JSONName:   field.Name,
		IsOptional: field.TypeRef.IsOptional,
		Doc:        doc.Get(field.Domains),
	}
}

// typeToGo converts an Oracle type reference to a Go type string.
func (p *Plugin) typeToGo(typeRef *resolution.TypeRef, data *templateData) string {
	var baseType string

	switch typeRef.Kind {
	case resolution.TypeKindPrimitive:
		mapping := primitiveGoTypes[typeRef.Primitive]
		baseType = mapping.goType
		if mapping.importPath != "" {
			data.Imports[mapping.importPath] = true
		}
	case resolution.TypeKindStruct:
		baseType = "any"
	case resolution.TypeKindEnum:
		baseType = "any"
	default:
		baseType = "any"
	}

	if typeRef.IsArray {
		baseType = "[]" + baseType
	}
	if typeRef.IsOptional && !typeRef.IsArray {
		baseType = "*" + baseType
	}

	return baseType
}

// toPascalCase converts snake_case to PascalCase.
func toPascalCase(s string) string {
	return lo.PascalCase(s)
}

// primitiveMapping defines how an Oracle primitive maps to Go.
type primitiveMapping struct {
	goType     string
	importPath string
}

// primitiveGoTypes maps Oracle primitives to Go types.
var primitiveGoTypes = map[string]primitiveMapping{
	"uuid":       {goType: "uuid.UUID", importPath: "github.com/google/uuid"},
	"string":     {goType: "string"},
	"bool":       {goType: "bool"},
	"int8":       {goType: "int8"},
	"int16":      {goType: "int16"},
	"int32":      {goType: "int32"},
	"int64":      {goType: "int64"},
	"uint8":      {goType: "uint8"},
	"uint16":     {goType: "uint16"},
	"uint32":     {goType: "uint32"},
	"uint64":     {goType: "uint64"},
	"float32":    {goType: "float32"},
	"float64":    {goType: "float64"},
	"timestamp":  {goType: "telem.TimeStamp", importPath: "github.com/synnaxlabs/x/telem"},
	"timespan":   {goType: "telem.TimeSpan", importPath: "github.com/synnaxlabs/x/telem"},
	"time_range": {goType: "telem.TimeRange", importPath: "github.com/synnaxlabs/x/telem"},
	"json":       {goType: "map[string]any"},
	"bytes":      {goType: "[]byte"},
}

// templateData holds data for the Go file template.
type templateData struct {
	Package string
	Structs []structData
	Imports map[string]bool
}

// SortedImports returns imports sorted for deterministic output.
func (d *templateData) SortedImports() []string {
	imports := make([]string, 0, len(d.Imports))
	for imp := range d.Imports {
		imports = append(imports, imp)
	}
	sort.Strings(imports)
	return imports
}

// structData holds data for a single struct definition.
type structData struct {
	Name   string
	Doc    string
	Fields []fieldData
}

// fieldData holds data for a single field definition.
type fieldData struct {
	GoName     string
	GoType     string
	JSONName   string
	IsOptional bool
	Doc        string
}

// tagSuffix returns the omitempty suffix if the field is optional.
func (f fieldData) TagSuffix() string {
	if f.IsOptional {
		return ",omitempty"
	}
	return ""
}

var templateFuncs = template.FuncMap{
	"join": strings.Join,
}

var fileTemplate = template.Must(template.New("go-types").Funcs(templateFuncs).Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}
{{- $imports := .SortedImports}}
{{- if $imports}}

import (
{{- range $imports}}
	"{{.}}"
{{- end}}
)
{{- end}}
{{range .Structs}}
{{if .Doc}}
// {{.Doc}}
{{end -}}
type {{.Name}} struct {
{{- range .Fields}}
{{- if .Doc}}
	// {{.Doc}}
{{- end}}
	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JSONName}}{{.TagSuffix}}" msgpack:"{{.JSONName}}{{.TagSuffix}}"` + "`" + `
{{- end}}
}
{{- end}}
`))
