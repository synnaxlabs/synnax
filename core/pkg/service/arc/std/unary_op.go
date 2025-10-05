// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package std

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/arc/stage"
	"github.com/synnaxlabs/synnax/pkg/service/arc/value"
	"github.com/synnaxlabs/x/maps"
)

// NOT is a unary operator
var symbolNot = ir.Symbol{
	Name: "not",
	Kind: ir.KindStage,
	Type: ir.Stage{
		Params: maps.Ordered[string, ir.Type]{
			Keys:   []string{"input"},
			Values: []ir.Type{ir.U8{}},
		},
		Return: ir.U8{},
	},
}

type notOperator struct {
	base
	input *value.Value
}

func (n *notOperator) Load(param string, val value.Value) {
	if param == "input" {
		n.input = &val
	}
}

func (n *notOperator) Next(ctx context.Context) {
	if n.input == nil {
		return
	}

	var result uint8
	if n.input.GetUint64() == 0 {
		result = 1
	} else {
		result = 0
	}
	n.outputHandler(ctx, "output", value.Value{Type: ir.U8{}}.PutUint8(result))
}

func NotFactory(_ context.Context, cfg Config) (stage.Node, error) {
	o := &notOperator{}
	o.key = cfg.Node.Key
	return o, nil
}