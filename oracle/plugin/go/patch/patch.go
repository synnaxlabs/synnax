// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package patch provides an Oracle plugin that generates the Patch methods that apply
// path actions to typed documents. A document is a struct with @patch fields, and
// path actions edit only those fields. Every struct that a @patch field reaches gets a
// Patch method over all of its fields.
package patch

import (
	"bytes"
	"context"
	"maps"
	"slices"
	"strings"
	"text/template"

	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/typemap"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

const patchDomain = "patch"

// Plugin generates Patch methods for the structs that @patch fields reach.
type Plugin struct{ Options Options }

// Options configures the go/patch plugin.
type Options struct {
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options { return Options{FileNamePattern: "patch.gen.go"} }

// New creates a new go/patch plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (*Plugin) Name() string       { return "go/patch" }
func (*Plugin) Domains() []string  { return []string{"go"} }
func (*Plugin) Requires() []string { return []string{"go/types"} }

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	targets, err := collect(req.Resolutions)
	if err != nil {
		return nil, err
	}
	byPath := make(map[string][]target)
	for _, t := range targets {
		path, err := definerPath(req, t.typ)
		if err != nil {
			return nil, err
		}
		byPath[path] = append(byPath[path], t)
	}
	resp := &plugin.Response{}
	for _, path := range slices.Sorted(maps.Keys(byPath)) {
		ts := byPath[path]
		slices.SortFunc(ts, func(a, b target) int {
			return strings.Compare(a.typ.QualifiedName, b.typ.QualifiedName)
		})
		content, err := render(naming.DerivePackageName(path), ts, req.Resolutions)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"failed to generate patch methods for %s",
				path,
			)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    path + "/" + p.Options.FileNamePattern,
			Content: content,
		})
	}
	deletions, err := stale(req, byPath)
	if err != nil {
		return nil, err
	}
	for _, path := range deletions {
		resp.Deletions = append(resp.Deletions, path+"/"+p.Options.FileNamePattern)
	}
	return resp, nil
}

// target is a struct that gets a Patch method over its editable fields.
type target struct {
	typ    resolution.Type
	fields []resolution.Field
}

// collect returns the documents and every struct that their @patch fields reach.
func collect(table *resolution.Table) ([]target, error) {
	var (
		targets []target
		queue   []resolution.Type
		roots   = make(set.Set[string])
		seen    = make(set.Set[string])
	)
	for _, t := range table.StructTypes() {
		if !document(t) {
			continue
		}
		fields := slices.DeleteFunc(
			slices.Clone(resolution.UnifiedFields(t, table)),
			func(f resolution.Field) bool {
				return !domain.HasDomainFromField(f, patchDomain)
			},
		)
		if len(fields) == 0 {
			continue
		}
		reached, err := reach(t, fields, table)
		if err != nil {
			return nil, err
		}
		roots.Add(t.QualifiedName)
		seen.Add(t.QualifiedName)
		targets = append(targets, target{typ: t, fields: fields})
		queue = append(queue, reached...)
	}
	for len(queue) > 0 {
		t := queue[0]
		queue = queue[1:]
		if roots.Contains(t.QualifiedName) {
			return nil, errors.Newf(
				"%s has @patch fields, so no @patch field can reach it",
				t.QualifiedName,
			)
		}
		if seen.Contains(t.QualifiedName) {
			continue
		}
		seen.Add(t.QualifiedName)
		fields := slices.DeleteFunc(
			slices.Clone(resolution.UnifiedFields(t, table)),
			wireOmitted,
		)
		reached, err := reach(t, fields, table)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target{typ: t, fields: fields})
		queue = append(queue, reached...)
	}
	return targets, nil
}

// document reports whether t declares path actions: an action that extends patch.Set.
// A struct that only inherits @patch fields, such as a create request, is not one.
func document(t resolution.Type) bool {
	form, ok := t.Form.(resolution.StructForm)
	return ok && slices.ContainsFunc(form.Actions, func(a resolution.Action) bool {
		return slices.ContainsFunc(a.Extends, func(r resolution.TypeRef) bool {
			return r.Name == "patch.Set"
		})
	})
}

// reach returns the structs that the fields of t hold, and an error for a field that
// path actions cannot edit.
func reach(
	t resolution.Type,
	fields []resolution.Field,
	table *resolution.Table,
) ([]resolution.Type, error) {
	var reached []resolution.Type
	for _, f := range fields {
		if wireOmitted(f) {
			return nil, errors.Newf(
				"%s.%s: a field without a JSON form cannot be @patch",
				t.QualifiedName, f.Name,
			)
		}
		if f.Optional && typemap.Nilable(f.Type, table) {
			return nil, errors.Newf(
				"%s.%s: path actions do not support an optional list or map",
				t.QualifiedName, f.Name,
			)
		}
		if s, ok := patchable(f, table); ok {
			reached = append(reached, s)
		}
	}
	return reached, nil
}

// patchable returns the struct that f holds when it gets a Patch method. Any other
// field is a value that only a whole set replaces.
func patchable(f resolution.Field, table *resolution.Table) (resolution.Type, bool) {
	if f.Type.IsTypeParam() || f.Type.ArraySize != nil {
		return resolution.Type{}, false
	}
	typ, _, ok := resolution.Dealias(f.Type, table)
	if !ok || omit.IsSkipped(typ, "go") {
		return resolution.Type{}, false
	}
	form, ok := typ.Form.(resolution.StructForm)
	if !ok || form.IsGeneric() {
		return resolution.Type{}, false
	}
	return typ, true
}

func wireOmitted(f resolution.Field) bool {
	return domain.GetStringFromField(f, "go", "wire") == "omit"
}

// definerPath returns the @go output path of the package that declares t: the version
// that defines it for a type in a version chain, or its own output path otherwise.
func definerPath(req *plugin.Request, t resolution.Type) (string, error) {
	root := output.GetPath(t, "go")
	if root == "" {
		return "", errors.Newf("%s has no @go output path", t.QualifiedName)
	}
	if req.Versions == nil {
		return root, nil
	}
	livePath := strings.TrimSuffix(t.FilePath, ".oracle")
	chain, ok := req.Versions.Chains()[livePath]
	if !ok {
		return root, nil
	}
	surface, err := req.Versions.Surface(
		context.Background(),
		livePath,
		chain.Current(),
	)
	if err != nil {
		return "", err
	}
	if def, ok := surface[t.Name]; ok {
		return versioning.VersionedPath(root, def.Version), nil
	}
	return versioning.VersionedPath(root, chain.Current()), nil
}

// stale returns every package path that could hold a patch file from an earlier run and
// holds none in this one.
func stale(req *plugin.Request, emitted map[string][]target) ([]string, error) {
	candidates := make(set.Set[string])
	for _, t := range req.Resolutions.StructTypes() {
		root := output.GetPath(t, "go")
		if root == "" {
			continue
		}
		candidates.Add(root)
		if req.RepoRoot == "" {
			continue
		}
		versions, err := versioning.VersionDirs(req.RepoRoot, root)
		if err != nil {
			return nil, err
		}
		for _, k := range versions {
			candidates.Add(versioning.VersionedPath(root, k))
		}
	}
	var paths []string
	for _, path := range slices.Sorted(maps.Keys(candidates)) {
		if _, ok := emitted[path]; !ok {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

type fileData struct {
	Package string
	Types   []typeData
}

type typeData struct {
	GoName   string
	Receiver string
	Fields   []fieldData
}

type fieldData struct {
	JSONName string
	GoName   string
	Kind     kind
}

// kind is how a Patch method applies an action to a field.
type kind string

const (
	kindValue          kind = "value"
	kindOptionalValue  kind = "optionalValue"
	kindStruct         kind = "struct"
	kindOptionalStruct kind = "optionalStruct"
)

func render(pkg string, targets []target, table *resolution.Table) ([]byte, error) {
	data := fileData{Package: pkg}
	for _, t := range targets {
		goName := naming.GetGoName(t.typ)
		td := typeData{GoName: goName, Receiver: receiver(goName)}
		for _, f := range t.fields {
			td.Fields = append(td.Fields, fieldData{
				JSONName: casing.FieldSnake(f.Name),
				GoName:   naming.GetFieldName(f),
				Kind:     fieldKind(f, table),
			})
		}
		data.Types = append(data.Types, td)
	}
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func fieldKind(f resolution.Field, table *resolution.Table) kind {
	_, isStruct := patchable(f, table)
	switch {
	case isStruct && f.Optional:
		return kindOptionalStruct
	case isStruct:
		return kindStruct
	case f.Optional:
		return kindOptionalValue
	default:
		return kindValue
	}
}

// receiver returns the initials of goName, with a "v" suffix when they would shadow a
// name that a Patch method uses.
func receiver(goName string) string {
	var initials []byte
	for i, c := range goName {
		if i == 0 || (c >= 'A' && c <= 'Z') {
			initials = append(initials, byte(c|0x20))
		}
	}
	name := string(initials)
	if reserved.Contains(name) {
		return name + "v"
	}
	return name
}

var reserved = set.New("c", "err", "next", "patch")

// Generated Patch methods call a nested struct's Patch method directly, never through a
// generic helper, so the compiler can keep the document on the stack.
var fileTemplate = template.Must(template.New("go-patch").Parse(
	`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import "github.com/synnaxlabs/x/patch"
{{- range .Types}}
{{- $r := .Receiver}}

// Patch applies the path action at c, which is not done, to the value.
func ({{$r}} *{{.GoName}}) Patch(c patch.Cursor) error {
	switch c.Segment() {
{{- range .Fields}}
	case "{{.JSONName}}":
{{- if eq .Kind "struct"}}
		if c = c.Next(); c.Done() {
			return patch.Value(&{{$r}}.{{.GoName}}, c)
		}
		return {{$r}}.{{.GoName}}.Patch(c)
{{- else if eq .Kind "optionalStruct"}}
		if c = c.Next(); c.Done() || {{$r}}.{{.GoName}} == nil {
			return patch.OptionalValue(&{{$r}}.{{.GoName}}, c)
		}
		next := *{{$r}}.{{.GoName}}
		if err := next.Patch(c); err != nil {
			return err
		}
		{{$r}}.{{.GoName}} = &next
		return nil
{{- else if eq .Kind "optionalValue"}}
		return patch.OptionalValue(&{{$r}}.{{.GoName}}, c.Next())
{{- else}}
		return patch.Value(&{{$r}}.{{.GoName}}, c.Next())
{{- end}}
{{- end}}
	default:
		return c.UnknownField()
	}
}
{{- end}}
`,
))
