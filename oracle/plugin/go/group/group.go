// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package group generates the Go table of the record types that can sit in a group,
// from the @group fields of the schemas. The table goes next to the group record.
package group

import (
	"bytes"
	"slices"
	"text/template"

	"github.com/synnaxlabs/oracle/domain/group"
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	goprimitives "github.com/synnaxlabs/oracle/plugin/go/primitives"
	"github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// Options configures the go/group plugin.
type Options struct {
	// FileNamePattern is the basename written next to the group record.
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options { return Options{FileNamePattern: "members.gen.go"} }

// Plugin generates the Go table of group members.
type Plugin struct {
	// Options sets the name of the generated file.
	Options Options
}

// New creates a new go/group plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "go/group" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"go"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"go/types"} }

// Generate emits one file next to the group record when any field carries @group.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{}
	data, err := group.Extract(req.Resolutions)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return resp, nil
	}
	outputPath := output.GetPath(data.Record, "go")
	if outputPath == "" {
		return resp, nil
	}
	mgr := imports.NewManager()
	td := templateData{Manager: mgr, Package: naming.DerivePackageName(outputPath)}
	enumPath := enum.FindOutputPath(data.Enum, req.Resolutions, "go")
	var qualifier string
	if enumPath != outputPath {
		alias := naming.DerivePackageName(enumPath)
		mgr.AddInternal(alias, gomod.ResolveImportPath(
			enumPath,
			req.RepoRoot,
			gomod.DefaultModulePrefix,
		))
		qualifier = alias + "."
	}
	td.Enum = qualifier + naming.GetGoName(data.Enum)
	constant := func(v resolution.EnumValue) string {
		return td.Enum + naming.ToPascalCase(v.Name)
	}
	for _, m := range data.Members {
		td.Members = append(td.Members, memberData{
			Type:      constant(m.Type),
			Scope:     constant(m.Scope),
			Pinned:    m.Pinned,
			Namespace: m.Namespace,
		})
		if m.Namespace && !slices.Contains(td.NamespaceScopes, constant(m.Scope)) {
			td.NamespaceScopes = append(td.NamespaceScopes, constant(m.Scope))
		}
	}
	if len(td.NamespaceScopes) > 0 {
		mgr.AddExternal("github.com/synnaxlabs/gorp")
		td.Group = naming.GetGoName(data.Record)
		td.Key = goKeyType(data.Record, outputPath, mgr, req)
	}
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, td); err != nil {
		return nil, err
	}
	resp.Files = append(resp.Files, plugin.File{
		Path:    outputPath + "/" + p.Options.FileNamePattern,
		Content: buf.Bytes(),
	})
	for _, m := range data.Members {
		if !m.Namespace {
			continue
		}
		file, err := siblingFile(m, data, outputPath, req)
		if err != nil {
			return nil, err
		}
		if file != nil {
			resp.Files = append(resp.Files, *file)
		}
	}
	return resp, nil
}

// siblingFile returns the file that holds the sibling index of m in the package of its
// record, or nil when the record has no Go output.
func siblingFile(
	m group.Member,
	data *group.Data,
	groupPath string,
	req *plugin.Request,
) (*plugin.File, error) {
	memberPath := output.GetPath(m.Record, "go")
	if memberPath == "" {
		return nil, nil
	}
	mgr := imports.NewManager()
	mgr.AddExternal("github.com/synnaxlabs/gorp")
	groupAlias := naming.DerivePackageName(groupPath)
	mgr.AddInternal(groupAlias, gomod.ResolveImportPath(
		groupPath,
		req.RepoRoot,
		gomod.DefaultModulePrefix,
	))
	enumPath := enum.FindOutputPath(data.Enum, req.Resolutions, "go")
	enumQualifier := ""
	if enumPath != memberPath {
		alias := naming.DerivePackageName(enumPath)
		mgr.AddInternal(alias, gomod.ResolveImportPath(
			enumPath,
			req.RepoRoot,
			gomod.DefaultModulePrefix,
		))
		enumQualifier = alias + "."
	}
	keyType := goKeyType(m.Record, memberPath, mgr, req)
	if keyType == "" {
		return nil, errors.Newf(
			"@group namespace needs %s to have a @key field or a @retrieve key",
			m.Record.QualifiedName,
		)
	}
	scope := naming.GetGoName(data.Enum) + naming.ToPascalCase(m.Type.Name)
	td := siblingData{
		Manager:  mgr,
		Package:  naming.DerivePackageName(memberPath),
		Group:    groupAlias,
		Record:   naming.GetGoName(m.Record),
		Key:      keyType,
		Scope:    enumQualifier + scope,
		Field:    naming.GetFieldName(m.Field),
		Optional: m.Field.Optional,
	}
	var buf bytes.Buffer
	if err := siblingTemplate.Execute(&buf, td); err != nil {
		return nil, err
	}
	return &plugin.File{
		Path:    memberPath + "/siblings.gen.go",
		Content: buf.Bytes(),
	}, nil
}

// goKeyType returns the Go type of the key of typ, in the package at outputPath: the
// type that `@retrieve key` names for a computed key, or else the type of its @key
// field.
func goKeyType(
	typ resolution.Type,
	outputPath string,
	mgr *imports.Manager,
	req *plugin.Request,
) string {
	ctx := &resolver.Context{
		Table:                         req.Resolutions,
		OutputPath:                    outputPath,
		Namespace:                     typ.Namespace,
		RepoRoot:                      req.RepoRoot,
		DomainName:                    "go",
		SubstituteDefaultedTypeParams: true,
	}
	r := &resolver.Resolver{
		Formatter: types.GoFormatter(),
		ImportResolver: &types.GoImportResolver{
			RepoRoot:       req.RepoRoot,
			CurrentPackage: naming.DerivePackageName(outputPath),
		},
		ImportAdder:     mgr,
		PrimitiveMapper: goprimitives.Mapper(),
	}
	if ref, ok := resolution.RetrieveKey(typ); ok {
		return r.ResolveTypeRef(ref, ctx)
	}
	for _, f := range resolution.UnifiedFields(typ, req.Resolutions) {
		if key.HasKey(f) {
			return r.ResolveTypeRef(f.Type, ctx)
		}
	}
	return ""
}

type templateData struct {
	*imports.Manager
	// Package is the Go package of the generated file.
	Package string
	// Enum is the Go type of a resource type, such as resource.Type.
	Enum string
	// Members are the record types that can sit in a group.
	Members []memberData
	// NamespaceScopes are the constants of the scopes of the namespace members.
	NamespaceScopes []string
	// Group and Key are the Go types of the group record and of its key.
	Group, Key string
}

type siblingData struct {
	*imports.Manager
	// Package is the Go package of the member record.
	Package string
	// Group is the name of the group package.
	Group string
	// Record and Key are the Go types of the member record and of its key.
	Record, Key string
	// Scope is the constant of the record's resource type.
	Scope string
	// Field is the Go field that holds the record's group.
	Field string
	// Optional is true when the field is a pointer.
	Optional bool
}

type memberData struct {
	// Type is the constant of the record's resource type.
	Type string
	// Scope is the constant of the scope of the groups that hold the record.
	Scope string
	// Pinned is true when the top parent of the group must be the record's holder.
	Pinned bool
	// Namespace is true when names in one group are unique.
	Namespace bool
}

var fileTemplate = template.Must(
	template.New("go-group").Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}
{{- if .HasImports}}

import (
{{- range .NonStdImports}}
{{- if .NeedsAlias}}
	{{.Alias}} "{{.Path}}"
{{- else}}
	"{{.Path}}"
{{- end}}
{{- end}}
)
{{- end}}

// memberRule says which groups can hold a record of one type.
type memberRule struct {
	// scope is the scope of the groups that hold the type.
	scope {{.Enum}}
	// pinned is true when the top parent of the group must be the record's holder, or
	// none when the record has no holder.
	pinned bool
	// namespace is true when the names of the records and the groups in one group are
	// unique.
	namespace bool
}

// memberRules holds the rule of each type that can sit in a group.
var memberRules = map[{{.Enum}}]memberRule{
{{- range .Members}}
	{{.Type}}: {scope: {{.Scope}}
{{- if .Pinned}}, pinned: true{{end}}
{{- if .Namespace}}, namespace: true{{end}}},
{{- end}}
}
{{- if .NamespaceScopes}}

// siblingIndexes returns the index of the names of the groups of each scope whose names
// are unique among siblings.
func siblingIndexes() []gorp.Index[{{.Key}}, {{.Group}}] {
	return []gorp.Index[{{.Key}}, {{.Group}}]{
{{- range .NamespaceScopes}}
		newSiblingIndex({{.}}),
{{- end}}
	}
}
{{- end}}
`),
)

var siblingTemplate = template.Must(
	template.New("go-group-sibling").Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import (
{{- range .NonStdImports}}
{{- if .NeedsAlias}}
	{{.Alias}} "{{.Path}}"
{{- else}}
	"{{.Path}}"
{{- end}}
{{- end}}
)

// newSiblingIndex returns the index that keeps the name of each {{.Record}} unique
// among the records and the groups in its group.
func newSiblingIndex() *gorp.UniqueIndex[{{.Key}}, {{.Record}}, {{.Group}}.Sibling] {
	return gorp.NewUniqueIndex(
		"sibling",
		{{.Group}}.SiblingSpace({{.Scope}}),
		func(e *{{.Record}}) ({{.Group}}.Sibling, bool) {
{{- if .Optional}}
			s := {{.Group}}.Sibling{Name: e.Name}
			if e.{{.Field}} != nil {
				s.Parent = *e.{{.Field}}
			}
			return s, true
{{- else}}
			return {{.Group}}.Sibling{Parent: e.{{.Field}}, Name: e.Name}, true
{{- end}}
		},
	)
}
`),
)
