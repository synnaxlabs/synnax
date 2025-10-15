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
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

type Output struct {
	Data telem.Series
	Time telem.Series
}

type State struct {
	Outputs map[ir.Handle]Output
}

func NewState(_ context.Context, program ir.IR) (*State, error) {
	state := &State{Outputs: map[ir.Handle]Output{}}
	for _, node := range program.Nodes {
		for key, t := range node.Outputs.Iter() {
			state.Outputs[ir.Handle{Node: node.Key, Param: key}] = Output{
				Data: telem.Series{DataType: types.ToTelem(t)},
				Time: telem.Series{DataType: telem.TimeStampT},
			}
		}
	}
	return state, nil
}
