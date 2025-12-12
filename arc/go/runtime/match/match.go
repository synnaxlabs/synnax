// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package match provides the Match node for Arc runtime case routing.
// Match nodes route string input values to corresponding u8(1) outputs,
// implementing switch/case behavior for stage transitions.
package match

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

const (
	symName          = "match"
	casesConfigParam = "cases"
)

var (
	sym = symbol.Symbol{
		Name: symName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: ir.DefaultInputParam, Type: types.String()}},
			Config: types.Params{
				// cases is an array of {value: string, output: string} objects
				// This is handled dynamically in the factory
				{Name: casesConfigParam, Type: types.String()}, // Placeholder type
			},
			// Outputs are dynamic based on cases config
		}),
	}
	// SymbolResolver provides the match symbol for the Arc analyzer.
	SymbolResolver = symbol.MapResolver{symName: sym}
)

// CaseMapping stores the output index and output name for a case value.
type CaseMapping struct {
	OutputIndex int
	OutputName  string
}

// Match is a node that routes input values to corresponding outputs.
// It receives a string input and fires a u8(1) signal on the output
// that matches the input value.
type Match struct {
	state   *state.Node
	caseMap map[string]CaseMapping
}

// Init performs one-time initialization (no-op for Match).
func (m *Match) Init(_ node.Context) {}

// Next checks the input value and routes it to the matching output.
func (m *Match) Next(ctx node.Context) {
	// Check if we have new input
	if !m.state.RefreshInputs() {
		return
	}

	input := m.state.Input(0)
	if input.Len() == 0 {
		return
	}

	// Get the input value as string (strings use variable-length encoding)
	strings := telem.UnmarshalStrings(input.Data)
	if len(strings) == 0 {
		return
	}
	inputValue := strings[0]

	// Find matching case
	if mapping, ok := m.caseMap[inputValue]; ok {
		output := m.state.Output(mapping.OutputIndex)
		output.Resize(1)
		telem.SetValueAt[uint8](*output, 0, uint8(1))
		// Mark the output as changed so edges propagate
		ctx.MarkChanged(mapping.OutputName)
	}
}

// Factory creates Match nodes for "match" type nodes in the IR.
type Factory struct{}

// NewFactory creates a new Match factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Create constructs a Match node from the given configuration.
// Returns query.NotFound if the node type is not "match".
func (f *Factory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symName {
		return nil, query.NotFound
	}

	// Build caseMap from config
	caseMap := make(map[string]CaseMapping)

	casesParam, ok := cfg.Node.Config.Get(casesConfigParam)
	if ok && casesParam.Value != nil {
		// Cases is an array of maps with "value" and "output" keys
		cases, ok := casesParam.Value.([]any)
		if ok {
			for _, caseEntry := range cases {
				entryMap, ok := caseEntry.(map[string]any)
				if !ok {
					continue
				}
				value, valueOk := entryMap["value"].(string)
				outputName, outputOk := entryMap["output"].(string)
				if !valueOk || !outputOk {
					continue
				}

				// Find output index by name
				for j, out := range cfg.Node.Outputs {
					if out.Name == outputName {
						caseMap[value] = CaseMapping{
							OutputIndex: j,
							OutputName:  outputName,
						}
						break
					}
				}
			}
		}
	}

	return &Match{
		state:   cfg.State,
		caseMap: caseMap,
	}, nil
}
