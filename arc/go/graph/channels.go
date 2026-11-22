// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph

import (
	"context"

	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// ChannelInputs returns the names of the inputs that the function of each node in g
// types as a channel, keyed by node key. It resolves a node's function among the
// functions of g, then in root, and analyzes no function body. A node with no function
// type or an undefined function has no entry, since nothing types its inputs. Any
// other failure to resolve a function is returned.
func ChannelInputs(
	ctx context.Context,
	g Graph,
	root *symbol.Symbol,
) (map[string][]string, error) {
	functions := make(map[string]types.Type, len(g.Functions))
	for _, fn := range g.Functions {
		functions[fn.Key] = fn.Type()
	}
	inputs := make(map[string][]string)
	for _, n := range g.Nodes {
		name, ok := g.Inputs[n.Key]["type"].(string)
		if !ok {
			continue
		}
		t, ok := functions[name]
		if !ok {
			sym, err := resolveQualified(ctx, root, name)
			if _, undefined := errors.AsType[*symbol.UndefinedSymbolError](
				err,
			); undefined {
				continue
			}
			if err != nil {
				return nil, err
			}
			t = sym.Type
		}
		for _, p := range t.Inputs {
			if p.Type.Kind == types.KindChan {
				inputs[n.Key] = append(inputs[n.Key], p.Name)
			}
		}
	}
	return inputs, nil
}
