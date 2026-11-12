// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package actions emits a TypeScript discriminated-union action codec for any Oracle
// struct that declares one or more actions. Each action becomes a Zod payload schema,
// an Action union dispatches by literal type, and the plugin emits a Handlers interface
// plus a createReduceAll factory that binds a caller-provided Handlers object.
// HandlerResult, ReduceAllResult, and the outer reduceAll loop live in the shared
// actions package; this plugin emits type aliases and a thin wrapper that delegate to
// them. The per-action handler functions are hand-written and supplied by the caller;
// this plugin does not generate them.
package actions

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/ts/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/ts/internal/paths"
	tstypes "github.com/synnaxlabs/oracle/plugin/ts/types"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
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

// Plugin emits the generated TS action codec.
type Plugin struct{ opts Options }

// New constructs a Plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{opts: opts} }

func (*Plugin) Name() string       { return "ts/actions" }
func (*Plugin) Domains() []string  { return []string{"ts"} }
func (*Plugin) Requires() []string { return []string{"ts/types"} }

// Generate emits one actions.gen.ts per output package containing a struct or union
// that declares actions. It skips types without actions and types that TS hand-writes
// or omits.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	req = req.OmitFields("ts")
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	types := append(req.Resolutions.StructTypes(), req.Resolutions.UnionTypes()...)
	for _, typ := range types {
		actions := typ.Actions()
		if len(actions) == 0 || omit.IsSkipped(typ, "ts") {
			continue
		}
		outputPath := output.GetPath(typ, "ts")
		if outputPath == "" {
			continue
		}
		content, err := p.generateFile(typ, actions, outputPath, req)
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
	actions []resolution.Action,
	outputPath string,
	req *plugin.Request,
) ([]byte, error) {
	mgr := imports.NewManager()
	fp := &tstypes.FieldProcessor{
		Imports:    mgr,
		Namespace:  typ.Namespace,
		OutputPath: outputPath,
		Request:    req,
	}

	tsName := domain.GetName(typ, "ts")
	data := &templateData{
		Manager:         mgr,
		TargetTypeName:  tsName,
		TargetProseName: proseName(typ.Name),
	}

	for _, action := range actions {
		a := actionData{
			Name:     casing.TypePascal(action.Name),
			TypeName: casing.TypeSnake(action.Name),
			Doc:      doc.Get(action.Domains),
		}
		if action.IsAlias() {
			target := fp.ProcessField(resolution.Field{Type: action.Alias}, typ)
			a.Target = &target
			fp.CollectTypeImports(&action.Alias)
		}
		for _, field := range action.Fields {
			a.Fields = append(a.Fields, fp.ProcessField(field, typ))
			fp.CollectTypeImports(&field.Type)
		}
		data.Actions = append(data.Actions, a)
	}

	mgr.AddImport("zod", "z")
	mgr.AddImport("immer", "Draft")
	mgr.AddImport("@synnaxlabs/x", "zod")
	sameDir := paths.CalculateImport(outputPath, outputPath)
	mgr.AddImport(sameDir+"/types.gen", tsName)
	if ref, ok := resolution.RetrieveKey(typ); ok {
		data.KeyZ = fp.ProcessField(
			resolution.Field{Name: "key", Type: ref},
			typ,
		).ZodType
	} else {
		data.KeyZ = "keyZ"
		mgr.AddImport(sameDir+"/types.gen", "keyZ")
	}
	mgr.AddImport(paths.CalculateImport(outputPath, "client/ts/src/actions"), "actions")

	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// templateData's fields are read by fileTemplate, so they must stay exported.
type templateData struct {
	*imports.Manager
	TargetTypeName  string
	TargetProseName string
	// KeyZ is the zod schema of the target type's key.
	KeyZ    string
	Actions []actionData
}

var properNouns = set.New("Arc")

func proseName(name string) string {
	if properNouns.Contains(name) {
		return name
	}
	return strings.ReplaceAll(casing.TypeSnake(name), "_", " ")
}

type actionData struct {
	Name     string
	TypeName string
	Doc      string
	// Target holds the zod schema and type of the struct that an action alias uses as
	// its payload. It is nil for an action with its own payload.
	Target *tstypes.FieldData
	Fields []tstypes.FieldData
}

var templateFuncs = template.FuncMap{
	"camelCase":  casing.TypeCamel,
	"pascalCase": casing.TypePascal,
	"formatDoc":  doc.FormatTS,
}

var fileTemplate = template.Must(
	template.New("ts-actions").
		Funcs(templateFuncs).
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
{{range .Actions}}
{{- if .Doc}}
{{formatDoc .Name .Doc}}
{{- end}}
{{- if .Target }}
export const {{ camelCase .Name }}PayloadZ = {{ .Target.ZodType }};

export type {{ .Name }}Payload = {{ .Target.TSType }};
{{- else }}
export const {{ camelCase .Name }}PayloadZ = z.object({
{{- range .Fields}}
  {{ .TSName }}: {{ .ZodType }},
{{- end}}
});

export type {{ .Name }}Payload = z.infer<typeof {{ camelCase .Name }}PayloadZ>;
{{- end }}
{{end}}
export type Action =
{{- range .Actions}}
  | { type: "{{ .TypeName }}"; {{ camelCase .Name }}: {{ .Name }}Payload }
{{- end}};

export const actionZ: z.ZodType<Action> = z.discriminatedUnion("type", [
{{- range .Actions}}
  z.object({ type: z.literal("{{ .TypeName }}"), {{ camelCase .Name }}: {{ camelCase .Name }}PayloadZ }),
{{- end}}
]);
{{range .Actions}}
export const {{ camelCase .Name }} = (payload: z.input<typeof {{ camelCase .Name }}PayloadZ>): Action => ({
  type: "{{ .TypeName }}",
  {{ camelCase .Name }}: zod.parse({{ camelCase .Name }}PayloadZ, payload, {
    label: "{{ $.TargetProseName }} {{ .TypeName }} action payload",
  }),
});
{{end}}
export type HandlerResult = actions.HandlerResult<Action>;

export type ReduceAllResult = actions.ReduceAllResult<{{ .TargetTypeName }}, Action>;

export interface Handlers {
{{- range .Actions}}
  {{ camelCase .Name }}: (state: Draft<{{ $.TargetTypeName }}>, payload: {{ .Name }}Payload) => HandlerResult;
{{- end}}
}

export const createReduceAll = (handlers: Handlers) =>
  actions.createReduceAll<{{ .TargetTypeName }}, Action>((state, action) => {
    switch (action.type) {
{{- range .Actions}}
      case "{{ .TypeName }}":
        return handlers.{{ camelCase .Name }}(state, action.{{ camelCase .Name }});
{{- end}}
    }
  });

export const scopedActionZ = actions.scopedZ({{ .KeyZ }}, actionZ);

export const dispatchReqZ = actions.dispatchReqZ({{ .KeyZ }}, actionZ);

export const dispatchResZ = actions.dispatchResZ(actionZ);
`),
)
