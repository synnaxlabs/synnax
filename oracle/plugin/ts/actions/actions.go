// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package actions emits the TypeScript action helpers of each struct or union that has
// actions. The ts/types plugin emits the payload schemas and the Action union. This
// plugin emits one builder per action that clients dispatch, a Handlers interface, a
// createReduceAll factory that binds a caller-provided Handlers object, and the scoped
// and dispatch schemas. The factory applies the update action itself, so Handlers
// leaves it out. HandlerResult, ReduceAllResult, applyUpdate, and the outer reduceAll
// loop live in the shared actions package.
package actions

import (
	"bytes"
	"text/template"

	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/ts/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/ts/internal/paths"
	tstypes "github.com/synnaxlabs/oracle/plugin/ts/types"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// Options configures the actions plugin output.
type Options struct {
	// FileNamePattern is the basename written for each output package.
	FileNamePattern string
}

// DefaultOptions returns the production defaults: actions.gen.ts.
func DefaultOptions() Options {
	return Options{FileNamePattern: "actions.gen.ts"}
}

// Plugin emits the generated TS action helpers.
type Plugin struct{ opts Options }

// New constructs a Plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{opts: opts} }

func (*Plugin) Name() string       { return "ts/actions" }
func (*Plugin) Domains() []string  { return []string{"ts"} }
func (*Plugin) Requires() []string { return []string{"ts/types"} }

// Generate emits one actions.gen.ts per output package that holds a struct or union
// with actions. It errors when such a type has no key.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	req = req.OmitFields("ts")
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	for _, typ := range domain.ActionTargets(req.Resolutions, "ts") {
		outputPath := output.GetPath(typ, "ts")
		content, err := p.generateFile(typ, outputPath, req)
		if err != nil {
			return nil, err
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    outputPath + "/" + p.opts.FileNamePattern,
			Content: content,
		})
	}
	return resp, nil
}

func (p *Plugin) generateFile(
	typ resolution.Type,
	outputPath string,
	req *plugin.Request,
) ([]byte, error) {
	table := req.Resolutions
	mgr := imports.NewManager()
	fp := &tstypes.FieldProcessor{
		Imports:    mgr,
		Namespace:  typ.Namespace,
		OutputPath: outputPath,
		Request:    req,
	}
	union, ok := table.Get(resolution.ActionUnion(typ))
	if !ok {
		return nil, errors.Newf("%s: no Action union", typ.QualifiedName)
	}
	unionName := domain.GetName(union, "ts")
	typesPath := paths.CalculateImport(outputPath, outputPath) + "/types.gen"
	data := &templateData{
		Manager:        mgr,
		TargetTypeName: domain.GetName(typ, "ts"),
		ActionType:     unionName,
		ActionZ:        casing.CamelAcronym(unionName) + "Z",
	}
	mgr.AddImport(typesPath, data.TargetTypeName)
	mgr.AddImport(typesPath, data.ActionType)
	mgr.AddImport(typesPath, data.ActionZ)
	var dispatched, dispatchedTypes []string
	for _, action := range typ.Actions() {
		payload, ok := action.Payload.Resolve(table)
		if !ok {
			return nil, errors.Newf(
				"%s: unresolved payload %s", typ.QualifiedName, action.Payload.Name,
			)
		}
		wire := casing.TypeSnake(action.Name)
		payloadName := domain.GetName(payload, "ts")
		a := actionData{
			Builder:     casing.TypeCamel(action.Name),
			Type:        wire,
			Member:      tstypes.FieldName(wire),
			PayloadType: payloadName,
			PayloadZ:    casing.CamelAcronym(payloadName) + "Z",
			Label:       payload.QualifiedName,
			CoreOnly:    action.CoreOnly,
		}
		if action.Update {
			a.UpdateZ = schemaOf(fp, typ, resolution.TypeRef{
				Name: typ.Namespace + "." + action.Name,
			})
			a.RecordZ = schemaOf(fp, typ, resolution.TypeRef{Name: typ.QualifiedName})
		}
		mgr.AddImport(typesPath, a.PayloadType)
		if !action.CoreOnly {
			mgr.AddImport(typesPath, a.PayloadZ)
			variant := tstypes.VariantSchemaName(unionName, wire)
			variantType := casing.VariantTypeName(unionName, wire)
			mgr.AddImport(typesPath, variant)
			mgr.AddImport(typesPath, variantType)
			dispatched = append(dispatched, variant)
			dispatchedTypes = append(dispatchedTypes, variantType)
		}
		data.Actions = append(data.Actions, a)
	}
	if len(dispatched) < len(data.Actions) {
		data.Dispatched = dispatched
		data.DispatchedTypes = dispatchedTypes
	}
	keyRef, ok := resolution.RetrieveKey(typ)
	if !ok {
		f, found := key.Find(typ, table)
		if !found {
			return nil, errors.Newf("%s: actions need a @key field", typ.QualifiedName)
		}
		keyRef = f.Type
	}
	data.KeyZ = fp.ProcessField(
		resolution.Field{Name: "key", Type: keyRef},
		typ,
	).ZodType
	fp.CollectTypeImports(&keyRef)
	if len(dispatched) > 0 {
		mgr.AddImport("zod", "z")
		mgr.AddImport("@synnaxlabs/x", "zod")
	}
	mgr.AddImport("immer", "Draft")
	mgr.AddImport(paths.CalculateImport(outputPath, "client/ts/src/actions"), "actions")

	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// schemaOf returns the zod schema of ref in the file of typ, and imports it.
func schemaOf(
	fp *tstypes.FieldProcessor,
	typ resolution.Type,
	ref resolution.TypeRef,
) string {
	schema := fp.ProcessField(resolution.Field{Name: "value", Type: ref}, typ).ZodType
	fp.CollectTypeImports(&ref)
	return schema
}

// templateData's fields are read by fileTemplate, so they must stay exported.
type templateData struct {
	*imports.Manager
	TargetTypeName string
	// ActionType and ActionZ are the type and schema of the Action union.
	ActionType string
	ActionZ    string
	// Dispatched and DispatchedTypes hold the schemas and types of the variants that
	// clients dispatch, set only when the Core alone emits some actions.
	Dispatched      []string
	DispatchedTypes []string
	// KeyZ is the zod schema of the target type's key.
	KeyZ    string
	Actions []actionData
}

type actionData struct {
	Builder     string
	Type        string
	Member      string
	PayloadType string
	PayloadZ    string
	Label       string
	CoreOnly    bool
	// UpdateZ and RecordZ are the schemas of the update and of the record, set only for
	// the update action.
	UpdateZ string
	RecordZ string
}

var fileTemplate = template.Must(
	template.New("ts-actions").
		Parse(`// Code generated by Oracle. DO NOT EDIT.
{{- range .SynnaxImports }}
import { {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} } from "{{ .Path }}";
{{- end }}
{{- range .ExternalNamedImports }}
import { {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} } from "{{ .Path }}";
{{- end }}
{{- range .InternalNamedImports }}
import { {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} } from "{{ .Path }}";
{{- end }}
{{- range .Actions }}
{{- if not .CoreOnly }}

export const {{ .Builder }} = (payload: z.input<typeof {{ .PayloadZ }}>): {{ $.ActionType }} => ({
  type: "{{ .Type }}",
  {{ .Member }}: zod.parse({{ .PayloadZ }}, payload, { label: "{{ .Label }}" }),
});
{{- end }}
{{- end }}

export type HandlerResult = actions.HandlerResult<{{ .ActionType }}>;

export type ReduceAllResult = actions.ReduceAllResult<{{ .TargetTypeName }}, {{ .ActionType }}>;

export interface Handlers {
{{- range .Actions }}
{{- if not .UpdateZ }}
  {{ .Member }}: (state: Draft<{{ $.TargetTypeName }}>, payload: {{ .PayloadType }}) => HandlerResult;
{{- end }}
{{- end }}
}

{{- range .Actions }}
{{- if .UpdateZ }}

const updateSchemas = { u: {{ .UpdateZ }}, t: {{ .RecordZ }} };
{{- end }}
{{- end }}

export const createReduceAll = (handlers: Handlers) =>
  actions.createReduceAll<{{ .TargetTypeName }}, {{ .ActionType }}>((state, action) => {
    switch (action.type) {
{{- range .Actions }}
      case "{{ .Type }}":
{{- if .UpdateZ }}
        return actions.applyUpdate(state, action.{{ .Member }}, updateSchemas);
{{- else }}
        return handlers.{{ .Member }}(state, action.{{ .Member }});
{{- end }}
{{- end }}
    }
  });

export const scopedActionZ = actions.scopedZ({{ .KeyZ }}, {{ .ActionZ }});
{{- if .Dispatched }}

const dispatchedActionZ: z.ZodType<{{ range $i, $t := .DispatchedTypes }}{{ if $i }} | {{ end }}{{ $t }}{{ end }}> =
  z.discriminatedUnion("type", [{{ range $i, $v := .Dispatched }}{{ if $i }}, {{ end }}{{ $v }}{{ end }}]);

export const dispatchReqZ = actions.dispatchReqZ({{ .KeyZ }}, dispatchedActionZ);
{{- else }}

export const dispatchReqZ = actions.dispatchReqZ({{ .KeyZ }}, {{ .ActionZ }});
{{- end }}

export const dispatchResZ = actions.dispatchResZ({{ .ActionZ }});
`),
)
