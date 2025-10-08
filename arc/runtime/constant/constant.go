// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constant

import (
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/maps"
)

var (
	symbol = ir.Symbol{
		Name: "constant",
		Kind: ir.KindStage,
		Type: ir.Stage{
			Config: maps.Ordered[string, ir.Type]{
				Keys:   []string{"value"},
				Values: []ir.Type{ir.NewTypeVariable("T", ir.NumericConstraint{})},
			},
			Outputs: maps.Ordered[string, ir.Type]{
				Keys:   []string{ir.DefaultOutput},
				Values: []ir.Type{ir.NewTypeVariable("T", ir.NumericConstraint{})},
			},
		},
	}
	Resolver = ir.MapResolver{"constant": symbol}
)

type constant struct {
	value       uint64
	initialized bool
}

func (c constant) Next() {}
