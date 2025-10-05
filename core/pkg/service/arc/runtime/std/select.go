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

	"github.com/samber/lo"
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
			Keys:   []string{"input", "true", "false"},
			Values: []ir.Type{ir.U8{}, ir.U8{}, ir.U8{}},
		},
		Return: ir.NewTypeVariable("T", nil),
	},
}

type selectStage struct {
	base
	input *value.Value
}

func (s *selectStage) Load(param string, val value.Value) {
	if param == "input" {
		s.input = &val
	}
}

func (s *selectStage) Next(ctx context.Context) {
	if s.input == nil {
		return
	}

	s.outputHandler(ctx, lo.Ternary(s.input.Value == 0, "false", "true"), *s.input)
}

func createSelect(_ context.Context, cfg Config) (stage.Node, error) {
	return &selectStage{base: base{key: cfg.Node.Key}}, nil
}
