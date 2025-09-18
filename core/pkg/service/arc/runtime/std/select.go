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
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/value"
	"github.com/synnaxlabs/x/maps"
)

var symbolSelect = ir.Symbol{
	Name: "select",
	Kind: ir.KindStage,
	Type: ir.Stage{
		Params: maps.Ordered[string, ir.Type]{
			Keys:   []string{"condition", "false", "true"},
			Values: []ir.Type{
				ir.U8{}, // Boolean condition
				ir.NewTypeVariable("T", nil), // false branch value
				ir.NewTypeVariable("T", nil), // true branch value
			},
		},
		Return: ir.NewTypeVariable("T", nil), // Return type matches branches
	},
}

type selectStage struct{ base }

func (s *selectStage) Next(ctx context.Context, val value.Value) {
	if val.Value == 0 {
		val.Param = "false"
	} else {
		val.Param = "true"
	}
	s.outputHandler(ctx, val)
}

func createSelect(_ context.Context, cfg Config) (stage.Stage, error) {
	return &selectStage{base: base{key: cfg.Node.Key}}, nil
}
