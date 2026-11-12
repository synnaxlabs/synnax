// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package actions emits a C++ action codec for each Oracle struct or union that
// declares actions: a payload struct per action, an Action std::variant of the
// payloads, a builder per action, and an x::json codec for the JSON form of an Action
// that Go and TypeScript use.
package actions

import (
	"bytes"
	"text/template"

	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/cpp/internal/includes"
	cppjson "github.com/synnaxlabs/oracle/plugin/cpp/json"
	"github.com/synnaxlabs/oracle/plugin/cpp/keywords"
	cppnaming "github.com/synnaxlabs/oracle/plugin/cpp/naming"
	cpptypes "github.com/synnaxlabs/oracle/plugin/cpp/types"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// Options configures the actions plugin output.
type Options struct {
	// FileNamePattern is the basename written for each output package.
	FileNamePattern string
}

// DefaultOptions returns the production defaults: actions.gen.h.
func DefaultOptions() Options {
	return Options{FileNamePattern: "actions.gen.h"}
}

// Plugin emits the generated C++ action codec.
type Plugin struct{ opts Options }

// New constructs a Plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{opts: opts} }

func (*Plugin) Name() string       { return "cpp/actions" }
func (*Plugin) Domains() []string  { return []string{"cpp"} }
func (*Plugin) Requires() []string { return []string{"cpp/json"} }

// Generate emits one actions.gen.h per output package containing a struct or union
// that declares actions. It skips types without actions and types that C++ hand-writes
// or omits.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	req = req.OmitFields("cpp")
	jsonPaths, err := cppjson.ContentPaths(req)
	if err != nil {
		return nil, err
	}
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	types := append(req.Resolutions.StructTypes(), req.Resolutions.UnionTypes()...)
	for _, typ := range types {
		actions := typ.Actions()
		if len(actions) == 0 || omit.IsSkipped(typ, "cpp") {
			continue
		}
		outputPath := output.GetPath(typ, "cpp")
		if outputPath == "" {
			continue
		}
		content, err := p.generateFile(typ, actions, outputPath, jsonPaths, req)
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
	jsonPaths set.Set[string],
	req *plugin.Request,
) ([]byte, error) {
	mgr := includes.NewManager()
	mgr.AddSystem("string")
	mgr.AddSystem("type_traits")
	mgr.AddSystem("variant")
	mgr.AddInternal(cppjson.IncludeFor(jsonPaths, outputPath))
	mgr.AddInternal("x/cpp/json/json.h")
	types := &cpptypes.FieldProcessor{
		Includes:  mgr,
		Namespace: typ.Namespace,
		Table:     req.Resolutions,
	}
	codecs := &cppjson.FieldProcessor{
		Includes:   mgr,
		Namespace:  typ.Namespace,
		OutputPath: outputPath,
		Table:      req.Resolutions,
		JSONPaths:  jsonPaths,
	}
	data := &templateData{
		Manager:    mgr,
		Namespace:  cppnaming.Namespace(outputPath),
		TargetName: typ.Name,
	}
	for _, action := range actions {
		snake := casing.TypeSnake(action.Name)
		a := actionData{
			Payload: casing.TypePascal(action.Name) + "Payload",
			Type:    snake,
			Builder: keywords.Escape(snake),
			Doc:     doc.Get(action.Domains),
		}
		// The payload is not in the resolution table, so no field refers to it.
		parent := resolution.Type{Name: a.Payload, Namespace: typ.Namespace}
		if action.IsAlias() {
			target := resolution.Field{Type: action.Alias}
			a.Target = types.ProcessField(target, parent).CppType
			codecs.ProcessField(target, parent)
		}
		for _, field := range action.Fields {
			a.Fields = append(a.Fields, fieldData{
				Decl:  types.ProcessField(field, parent),
				Codec: codecs.ProcessField(field, parent),
			})
		}
		data.Actions = append(data.Actions, a)
	}
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// templateData's fields are read by fileTemplate, so they must stay exported.
type templateData struct {
	*includes.Manager
	Namespace  string
	TargetName string
	Actions    []actionData
}

type actionData struct {
	// Payload is the name of the payload struct.
	Payload string
	// Type is the action type on the wire.
	Type string
	// Builder is the name of the function that wraps a payload in an Action.
	Builder string
	Doc     string
	// Target is the C++ type of the struct that an action alias uses as its payload.
	Target string
	Fields []fieldData
}

type fieldData struct {
	Decl  cpptypes.FieldData
	Codec cppjson.FieldData
}

var templateFuncs = template.FuncMap{
	"formatDoc": doc.FormatCpp,
	"actionDoc": func(target string) string {
		return "is a discriminated union for all " + target + " mutations. Its JSON " +
			`form is {"type": t, t: payload}, where t is the type of the action.`
	},
}

var fileTemplate = template.Must(
	template.New("cpp-actions").
		Funcs(templateFuncs).
		Parse(`// Code generated by oracle. DO NOT EDIT.

#pragma once

{{range .SystemIncludes -}}
#include <{{.}}>
{{end}}
{{range .InternalIncludes -}}
#include "{{.}}"
{{end}}
namespace {{.Namespace}} {
{{- range .Actions}}

{{if .Doc}}{{formatDoc .Payload .Doc}}
{{end -}}
{{if .Target -}}
using {{.Payload}} = {{.Target}};
{{- else -}}
struct {{.Payload}} {
{{- range .Fields}}
{{- if .Decl.Doc}}
    {{formatDoc .Decl.Name .Decl.Doc 4}}
{{- end}}
    {{.Decl.CppType}} {{.Decl.Name}}{{if .Decl.DefaultValue}} = {{.Decl.DefaultValue}}{{end}};
{{- end}}

    static {{.Payload}} parse(x::json::Parser parser);
    [[nodiscard]] x::json::json to_json() const;
};
{{- end}}
{{- end}}

{{formatDoc "Action" (actionDoc .TargetName)}}
using Action = std::variant<{{range $i, $a := .Actions}}{{if $i}}, {{end}}{{$a.Payload}}{{end}}>;

/// @brief parses an Action from its JSON form.
Action parse_action(x::json::Parser parser);
/// @brief returns the JSON form of an Action.
[[nodiscard]] x::json::json to_json(const Action &value);
{{- range .Actions}}

/// @brief returns an Action that carries the given payload.
inline Action {{.Builder}}({{.Payload}} payload) {
    return payload;
}
{{- end}}
{{- range .Actions}}
{{- if not .Target}}

inline {{.Payload}} {{.Payload}}::parse(x::json::Parser{{if .Fields}} parser{{end}}) {
    return {{.Payload}}{
{{- range .Fields}}
        .{{.Codec.Name}} = {{.Codec.ParseExpr}},
{{- end}}
    };
}

inline x::json::json {{.Payload}}::to_json() const {
    x::json::json j = x::json::json::object();
{{- range .Fields}}
    {{.Codec.ToJSONExpr}}
{{- end}}
    return j;
}
{{- end}}
{{- end}}

inline Action parse_action(x::json::Parser parser) {
    const auto type = parser.field<std::string>("type");
{{- range .Actions}}
    if (type == "{{.Type}}") return {{.Payload}}::parse(parser.child("{{.Type}}"));
{{- end}}
    parser.field_err("type", "unknown Action type: " + type);
    return {};
}

inline x::json::json to_json(const Action &value) {
    return std::visit(
        [](const auto &payload) {
            using Payload = std::decay_t<decltype(payload)>;
            std::string type;
{{- range $i, $a := .Actions}}
            {{if $i}}else {{end}}if constexpr (std::is_same_v<Payload, {{$a.Payload}}>) type = "{{$a.Type}}";
{{- end}}
            x::json::json j = x::json::json::object();
            j["type"] = type;
            j[type] = payload.to_json();
            return j;
        },
        value
    );
}

}
`),
)
