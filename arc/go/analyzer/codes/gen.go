// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build ignore

// Writes the table of kinds to codes.gen.json, which the docs site reads. It compiles
// the example of each kind and checks that the example reports the code of the kind.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"go.lsp.dev/protocol"
)

type kind struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Problem  string `json:"problem"`
	Example  string `json:"example"`
}

var (
	severities = map[protocol.DiagnosticSeverity]string{
		protocol.DiagnosticSeverityError:   "error",
		protocol.DiagnosticSeverityWarning: "warning",
	}
	// channels are the channels that an example can read and write.
	channels = []symbol.Symbol{
		channel("sensor", types.F64(), 1),
		channel("valve_cmd", types.U8(), 2),
		channel("count", types.I64(), 3),
		channel("label", types.String(), 4),
		channel("sensor_avg", types.F64(), 5),
	}
	// modules are the modules that an example can import.
	modules = map[string]string{"valves": "func open() i64 {\n    return 1\n}\n"}
	cfg     = parser.Config{AllowDashedNames: true}
)

func channel(name string, t types.Type, id int) symbol.Symbol {
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindChannel,
		Type: types.Chan(t),
		ID:   id,
	}
}

func newRoot() *symbol.Symbol {
	loader := text.NewLoader(text.LoaderConfig{
		Source: func(_ context.Context, path string) (text.Text, error) {
			src, ok := modules[path]
			if !ok {
				return text.Text{}, errors.Wrapf(symbol.ErrModuleNotFound, "%q", path)
			}
			return text.Text{Raw: src}, nil
		},
		Ambient: stl.NewSymbols,
	})
	root := symbol.NewRoot(nil, stl.NewSymbols(), symbol.WithModules(loader))
	for i := range channels {
		ch := channels[i]
		root.Parent.AddChild(&ch)
	}
	return root
}

// report returns each diagnostic that the compiler reports for source as a program or
// as a module.
func report(source string, module bool) diagnostics.Diagnostics {
	parsed, diag := text.Parse(text.Text{Raw: source}, cfg)
	if diag != nil && !diag.Ok() {
		return *diag
	}
	analyze := text.Analyze
	if module {
		analyze = text.AnalyzeModule
	}
	_, diag = analyze(context.Background(), parsed, newRoot(), cfg)
	if diag == nil {
		return nil
	}
	return *diag
}

func generate(k diagnostics.Kind) (kind, error) {
	severity, ok := severities[k.Severity]
	if !ok {
		return kind{}, errors.Newf(
			"%s has a severity that the docs do not show",
			k.Code,
		)
	}
	if k.Message == "" || k.Problem == "" {
		return kind{}, errors.Newf("%s has no message or no problem text", k.Code)
	}
	out := kind{
		Severity: severity,
		Message:  k.Message,
		Problem:  k.Problem,
		Example:  k.Example,
	}
	if k.Example == "" {
		return out, nil
	}
	for _, module := range []bool{false, true} {
		for _, d := range report(k.Example, module) {
			if d.Code == k.Code {
				if d.Message != k.Message &&
					!strings.HasPrefix(d.Message, k.Message+": ") {
					return kind{}, errors.Newf(
						"the message of %s does not start with its headline %q: %s",
						k.Code, k.Message, d.Message,
					)
				}
				return out, nil
			}
		}
	}
	reported := report(k.Example, false)
	return kind{}, errors.Newf(
		"the example of %s does not report it:\n%s",
		k.Code,
		reported.String(),
	)
}

func main() {
	var (
		kinds  = make(map[diagnostics.ErrorCode]kind, len(codes.All))
		failed bool
	)
	for _, k := range codes.All {
		if _, ok := kinds[k.Code]; ok {
			fmt.Fprintf(os.Stderr, "two kinds have the code %s\n", k.Code)
			failed = true
			continue
		}
		out, err := generate(k)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed = true
		}
		kinds[k.Code] = out
	}
	if failed {
		os.Exit(1)
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(kinds)
	if err == nil {
		err = os.WriteFile("codes.gen.json", b.Bytes(), 0o644)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write the kinds: %v\n", err)
		os.Exit(1)
	}
}
