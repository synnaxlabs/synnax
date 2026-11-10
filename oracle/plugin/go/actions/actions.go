// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package actions generates the reducer of each struct or union that has actions. The
// go/types plugin emits the payloads and the Action union. This plugin emits Reduce,
// one New<Name>Action constructor per action, the Handle method of each payload that
// only the Core emits, the Handle and CheckCoreOnly methods of the update payload, and
// the CheckCoreOnly method of the Action union of a record with standard actions. Every
// other Handle method is hand-written in a sibling file.
package actions

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/presence"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// Options configures the actions plugin output.
type Options struct {
	// FileNamePattern is the basename written for each output package.
	FileNamePattern string
	// RuntimeImportPath is the Go package that supplies ErrCoreOnly, which the Handle
	// method of a payload that only the Core emits returns. Its package name must be
	// actions. Required.
	RuntimeImportPath string
}

// DefaultOptions returns the production defaults: actions.gen.go against the Core's
// actions service package.
func DefaultOptions() Options {
	return Options{
		FileNamePattern:   "actions.gen.go",
		RuntimeImportPath: "github.com/synnaxlabs/synnax/pkg/service/actions",
	}
}

// Plugin emits the generated reducer.
type Plugin struct{ Options Options }

// New constructs a Plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string       { return "go/actions" }
func (p *Plugin) Domains() []string  { return []string{"go"} }
func (p *Plugin) Requires() []string { return []string{"go/types"} }

// Generate emits one actions.gen.go per output package that holds a struct or union
// with actions.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	if p.Options.RuntimeImportPath == "" {
		return nil, errors.New("go/actions requires Options.RuntimeImportPath")
	}
	resp := &plugin.Response{}
	for _, typ := range domain.ActionTargets(req.Resolutions, "go") {
		outputPath := output.GetPath(typ, "go")
		content, err := p.generateFile(typ, outputPath, req.Resolutions)
		if err != nil {
			return nil, err
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    outputPath + "/" + p.Options.FileNamePattern,
			Content: content,
		})
	}
	return resp, nil
}

func (p *Plugin) generateFile(
	typ resolution.Type,
	outputPath string,
	table *resolution.Table,
) ([]byte, error) {
	mgr := imports.NewManager()
	mgr.AddExternal("github.com/synnaxlabs/x/errors")
	mgr.AddExternal("github.com/synnaxlabs/x/union")
	data := templateData{
		Package:    naming.DerivePackageName(outputPath),
		TargetType: targetType(typ),
		Manager:    mgr,
	}
	for _, action := range typ.Actions() {
		payload, ok := action.Payload.Resolve(table)
		if !ok {
			return nil, errors.Newf(
				"%s: unresolved payload %s", typ.QualifiedName, action.Payload.Name,
			)
		}
		ad := actionData{
			Name:     casing.TypePascal(action.Name),
			Payload:  naming.GetGoName(payload),
			CoreOnly: action.CoreOnly,
		}
		if action.Update {
			if err := p.update(&ad, payload, action, &data, table); err != nil {
				return nil, errors.Wrap(err, typ.QualifiedName)
			}
			data.CoreFields = action.CoreFields
			data.BindsVariant = true
		}
		if form, ok := payload.Form.(resolution.DistinctForm); ok {
			ad.Path = pathOf[form.Base.Name]
		}
		data.Actions = append(data.Actions, ad)
		data.ChecksCoreOnly = data.ChecksCoreOnly || action.CoreOnly || action.Update
		if action.CoreOnly {
			mgr.AddExternal(p.Options.RuntimeImportPath)
		}
	}
	if len(data.CoreFields) > 0 && lo.SomeBy(data.Actions, func(a actionData) bool {
		return a.Path != ""
	}) {
		data.ChecksPaths = true
		mgr.AddExternal("slices")
	}
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, &data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// update fills the data of the update action, whose payload holds the update and its
// guards.
func (p *Plugin) update(
	ad *actionData,
	payload resolution.Type,
	action resolution.Action,
	data *templateData,
	table *resolution.Table,
) error {
	form, ok := payload.Form.(resolution.DistinctForm)
	if !ok || len(form.Base.TypeArgs) != 1 {
		return errors.Newf(
			"the update payload %s holds no update",
			payload.QualifiedName,
		)
	}
	update, ok := form.Base.TypeArgs[0].Resolve(table)
	if !ok {
		return errors.Newf("unresolved update %s", form.Base.TypeArgs[0].Name)
	}
	data.AddExternal(patchImportPath)
	ad.Update = naming.GetGoName(update)
	if len(action.CoreFields) == 0 {
		return nil
	}
	fields, ok := update.Form.(resolution.StructForm)
	if !ok {
		return errors.Newf(
			"%s has Core-only fields but is no struct",
			update.QualifiedName,
		)
	}
	data.AddExternal(p.Options.RuntimeImportPath)
	for _, name := range action.CoreFields {
		f, ok := lo.Find(fields.Fields, func(f resolution.Field) bool {
			return f.Name == name
		})
		if !ok {
			return errors.Newf("%s holds no field %s", update.QualifiedName, name)
		}
		sel := "p.Value." + naming.GetFieldName(f)
		changed := sel + " != nil"
		if presence.Field(f, table) {
			changed = "!" + sel + ".IsZero()"
		}
		ad.CoreFields = append(
			ad.CoreFields,
			coreFieldData{Name: name, Changed: changed},
		)
	}
	return nil
}

// targetType returns the Go type that Reduce takes. Go keeps only the type parameters
// that have no default, and Reduce instantiates each one with any.
func targetType(typ resolution.Type) string {
	name := naming.GetGoName(typ)
	params := resolution.NonDefaultedTypeParams(resolution.TypeParams(typ.Form))
	if len(params) == 0 {
		return name
	}
	return name + "[" + strings.Repeat("any, ", len(params)-1) + "any]"
}

type templateData struct {
	Package    string
	TargetType string
	Actions    []actionData
	// CoreFields names the fields that only the Core sets.
	CoreFields []string
	// ChecksCoreOnly reports that the Action union gets a CheckCoreOnly method, which a
	// record with a create or an update action needs.
	ChecksCoreOnly bool
	// ChecksPaths reports that CheckCoreOnly refuses the path actions on CoreFields.
	ChecksPaths bool
	// BindsVariant reports that a case of CheckCoreOnly reads the payload, which an
	// update does.
	BindsVariant bool
	*imports.Manager
}

type actionData struct {
	Name     string
	Payload  string
	CoreOnly bool
	// Update is the update type that the payload of the update action holds, or empty
	// for any other action.
	Update     string
	CoreFields []coreFieldData
	// Path is the expression for the path of a path action payload v, or empty for any
	// other action.
	Path string
}

// coreFieldData is a field that only the Core sets, with the condition under which an
// update changes it.
type coreFieldData struct{ Name, Changed string }

const patchImportPath = "github.com/synnaxlabs/x/patch"

// pathOf maps the patch type that a path action payload derives from to the expression
// for its path. A remove names the field it removes as its key.
var pathOf = map[string]string{
	"patch.Set":    "v.Path",
	"patch.Remove": "slices.Concat(v.Path, []string{v.Key})",
	"patch.Insert": "v.Path",
	"patch.Move":   "v.Path",
}

var fileTemplate = template.Must(
	template.New("go-actions").
		Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import (
{{- range .StdImports}}
	"{{.}}"
{{- end}}
{{- if and .StdImports .NonStdImports}}
{{end}}
{{- range .NonStdImports}}
{{- if .NeedsAlias}}
	{{.Alias}} "{{.Path}}"
{{- else}}
	"{{.Path}}"
{{- end}}
{{- end}}
)
{{- range .Actions}}
{{- if .CoreOnly}}

// Handle returns actions.ErrCoreOnly, as only the Core emits a {{.Payload}}.
func ({{.Payload}}) Handle({{$.TargetType}}) ({{$.TargetType}}, error) {
	return {{$.TargetType}}{}, actions.ErrCoreOnly
}
{{- end}}
{{- if .Update}}

// Handle applies the update in p to r when its guards hold. It returns r and an error
// wrapping patch.ErrStale when a guard does not hold, and r for any other error.
func (p {{.Payload}}) Handle(r {{$.TargetType}}) ({{$.TargetType}}, error) {
	u := patch.Update[{{.Update}}](p)
	if err := u.Check(r); err != nil {
		return r, err
	}
	next := r
	if err := u.Value.ApplyTo(&next); err != nil {
		return r, err
	}
	return next, nil
}

// CheckCoreOnly returns an error wrapping actions.ErrCoreOnly when p changes a field
// that only the Core sets.
func (p {{.Payload}}) CheckCoreOnly() error {
{{- range .CoreFields}}
	if {{.Changed}} {
		return errors.Wrap(actions.ErrCoreOnly, "{{.Name}}")
	}
{{- end}}
	return nil
}
{{- end}}
{{- end}}
{{- if .ChecksCoreOnly}}

// CheckCoreOnly returns an error wrapping actions.ErrCoreOnly when a is an action that
// only the Core dispatches, or when a changes a field that only the Core sets.
func (a Action) CheckCoreOnly() error {
{{- if .ChecksPaths}}
	var path []string
{{- end}}
	switch {{if .BindsVariant}}v := {{end}}a.Variant.(type) {
{{- range .Actions}}
{{- if .CoreOnly}}
	case {{.Payload}}:
		return actions.ErrCoreOnly
{{- else if .Update}}
	case {{.Payload}}:
		return v.CheckCoreOnly()
{{- else if and $.ChecksPaths .Path}}
	case {{.Payload}}:
		path = {{.Path}}
{{- end}}
{{- end}}
	}
{{- if .ChecksPaths}}
	if len(path) == 0 {
		return nil
	}
	switch path[0] {
	case {{range $i, $f := .CoreFields}}{{if $i}}, {{end}}"{{$f}}"{{end}}:
		return errors.Wrap(actions.ErrCoreOnly, path[0])
	}
{{- end}}
	return nil
}
{{- end}}

// Reduce applies acts to state in order through the Handle method of each payload. It
// returns the first error with the state reduced up to it. An Action with no Variant
// returns union.ErrMissingPayload.
func Reduce(state {{.TargetType}}, acts ...Action) ({{.TargetType}}, error) {
	var err error
	for _, a := range acts {
		switch v := a.Variant.(type) {
{{- range .Actions}}
		case {{.Payload}}:
			state, err = v.Handle(state)
{{- end}}
		case nil:
			return state, union.ErrMissingPayload
		default:
			return state, errors.Newf("Action: unknown variant %T", v)
		}
		if err != nil {
			return state, err
		}
	}
	return state, nil
}
{{range .Actions}}
// New{{.Name}}Action returns an Action that carries p.
func New{{.Name}}Action(p {{.Payload}}) Action { return Action{Variant: p} }
{{end -}}
`),
)
