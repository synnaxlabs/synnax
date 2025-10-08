// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	arctelem "github.com/synnaxlabs/arc/runtime/util"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

type State struct {
	Outputs map[ir.Handle]telem.Series
}

func NewState(ctx context.Context, program ir.IR) (*State, error) {
	state := &State{Outputs: map[ir.Handle]telem.Series{}}
	for _, node := range program.Nodes {
		symbol, err := program.Symbols.Resolve(ctx, node.Type)
		if err != nil {
			return nil, err
		}
		stage, ok := symbol.Type.(*ir.Stage)
		if !ok {
			return nil, errors.Newf("stage %s is not a stage", symbol.Type)
		}
		for key, t := range stage.Outputs.Iter() {
			state.Outputs[ir.Handle{Node: node.Key, Param: key}] = telem.Series{
				DataType: arctelem.IRTypeToDataType(t),
			}
		}
	}
	return state, nil
}
