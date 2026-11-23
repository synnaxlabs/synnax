// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package actions emits a Python action module for each Oracle struct or union that
// declares actions: a Pydantic payload model per action, an envelope model per action
// that dumps to the JSON form that Go and TypeScript use, an Action union of the
// envelopes, and a builder per action.
package actions

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/samber/lo"

	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/py/keywords"
	pytypes "github.com/synnaxlabs/oracle/plugin/py/types"
	"github.com/synnaxlabs/oracle/resolution"
)

// Options configures the actions plugin output.
type Options struct {
	// FileNamePattern is the basename written for each output package.
	FileNamePattern string
}

// DefaultOptions returns the production defaults: actions_gen.py.
func DefaultOptions() Options {
	return Options{FileNamePattern: "actions_gen.py"}
}

// Plugin emits the generated Python action module.
type Plugin struct{ opts Options }

// New constructs a Plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{opts: opts} }

func (*Plugin) Name() string       { return "py/actions" }
func (*Plugin) Domains() []string  { return []string{"py"} }
func (*Plugin) Requires() []string { return []string{"py/types"} }

// Generate emits one actions_gen.py per output package containing a struct or union
// that declares actions. It skips types without actions and types that Python
// hand-writes or omits.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	req = req.OmitFields("py")
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	types := append(req.Resolutions.StructTypes(), req.Resolutions.UnionTypes()...)
	for _, typ := range types {
		actions := typ.Actions()
		if len(actions) == 0 || omit.IsSkipped(typ, "py") {
			continue
		}
		outputPath := output.GetPath(typ, "py")
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
	fp := pytypes.NewFieldProcessor(
		typ.Namespace,
		typ.FilePath,
		outputPath,
		req.Resolutions,
	)
	fp.AddTyping("Annotated")
	fp.AddTyping("Literal")
	fp.AddPydantic("BaseModel")
	fp.AddPydantic("Field")
	data := &templateData{TargetName: typ.Name}
	for _, action := range actions {
		snake := casing.TypeSnake(action.Name)
		a := actionData{
			Payload:  casing.TypePascal(action.Name) + "Payload",
			Envelope: casing.VariantTypeName("Action", snake),
			Type:     snake,
			Field:    keywords.EscapeField(snake),
			Doc:      doc.Get(action.Domains),
		}
		a.Config = envelopeConfig(a.Field, snake)
		fp.Reserve(a.Payload)
		fp.Reserve(a.Envelope)
		fp.Reserve(a.Field)
		for _, field := range action.Fields {
			fp.ReserveField(field)
		}
		data.Actions = append(data.Actions, a)
	}
	fp.Reserve("Action")
	hasConfig := false
	for i, action := range actions {
		a := &data.Actions[i]
		if action.IsAlias() {
			a.Target = fp.ProcessField(resolution.Field{Type: action.Alias}).PyType
		}
		for _, field := range action.Fields {
			fd := fp.ProcessField(field)
			a.Fields = append(a.Fields, fd)
		}
		a.PayloadConfig = payloadConfig(a.Fields)
		hasConfig = hasConfig || a.PayloadConfig != "" || a.Config != ""
	}
	if hasConfig {
		fp.AddPydantic("ConfigDict")
	}
	imports, err := fp.Imports()
	if err != nil {
		return nil, err
	}
	data.Imports = imports
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// templateData's fields are read by fileTemplate, so they must stay exported.
type templateData struct {
	Imports    string
	TargetName string
	Actions    []actionData
}

// payloadConfig returns the model_config options of a payload with fields, or empty
// for none.
func payloadConfig(fields []pytypes.FieldData) string {
	var opts []string
	if lo.SomeBy(fields, func(f pytypes.FieldData) bool { return f.Alias != "" }) {
		opts = append(opts, "populate_by_name=True")
	}
	if lo.SomeBy(fields, func(f pytypes.FieldData) bool {
		return keywords.Protected(f.Name)
	}) {
		opts = append(opts, "protected_namespaces=()")
	}
	return strings.Join(opts, ", ")
}

// envelopeConfig returns the model_config options of an envelope whose payload field is
// named field and sent as typ, or empty for none.
func envelopeConfig(field, typ string) string {
	var opts []string
	if field != typ {
		opts = append(opts, "populate_by_name=True")
	}
	if keywords.Protected(field) {
		opts = append(opts, "protected_namespaces=()")
	}
	return strings.Join(opts, ", ")
}

type actionData struct {
	// Payload is the name of the payload model.
	Payload string
	// Envelope is the name of the model that carries the payload in an Action.
	Envelope string
	// Type is the action type on the wire.
	Type string
	// Field is the envelope attribute that holds the payload. It is also the name of
	// the builder function.
	Field string
	Doc   string
	// Target is the Python type of the struct that an action alias uses as its payload.
	Target string
	// Config is the options of the envelope's model_config, or empty for none.
	Config string
	// PayloadConfig is the options of the payload's model_config, or empty for none.
	PayloadConfig string
	Fields        []pytypes.FieldData
}

func payloadDocstring(a actionData) string {
	fields := make([]doc.FieldDoc, 0, len(a.Fields))
	for _, f := range a.Fields {
		fields = append(fields, doc.FieldDoc{Name: f.Name, Doc: f.Doc})
	}
	return doc.FormatPyDocstringGoogle(a.Doc, fields)
}

var templateFuncs = template.FuncMap{
	"payloadDocstring": payloadDocstring,
	"formatComment":    doc.FormatPyComment,
	"actionDoc": func(target string) string {
		return "# Action is a discriminated union for all " + target + " mutations."
	},
}

var fileTemplate = template.Must(
	template.New("py-actions").
		Funcs(templateFuncs).
		Parse(`# Code generated by Oracle. DO NOT EDIT.

from __future__ import annotations{{ .Imports }}
{{- range .Actions }}
{{- if .Target }}


{{ with formatComment .Doc }}{{ . }}
{{ end -}}
{{ .Payload }} = {{ .Target }}
{{- else }}


class {{ .Payload }}(BaseModel):
{{- with payloadDocstring . }}
{{ . }}
{{- end }}
{{- with .PayloadConfig }}
    model_config = ConfigDict({{ . }})
{{ end }}
{{- range .Fields }}
    {{ .Name }}: {{ .PyType }}{{ .Default }}
{{- end }}
{{- if and (not .Fields) (not (payloadDocstring .)) }}
    pass
{{- end }}
{{- end }}
{{- end }}
{{- range .Actions }}


class {{ .Envelope }}(BaseModel):
{{- with .Config }}
    model_config = ConfigDict({{ . }})
{{ end }}
    type: Literal["{{ .Type }}"] = "{{ .Type }}"
    {{ .Field }}: {{ .Payload }}{{ if ne .Field .Type }} = Field(alias="{{ .Type }}"){{ end }}
{{- end }}


{{ actionDoc .TargetName }}
Action = Annotated[
    {{ range $i, $a := .Actions }}{{ if $i }} | {{ end }}{{ $a.Envelope }}{{ end }},
    Field(discriminator="type"),
]
{{- range .Actions }}


def {{ .Field }}(payload: {{ .Payload }}) -> {{ .Envelope }}:
    return {{ .Envelope }}({{ .Field }}=payload)
{{- end }}
`),
)
