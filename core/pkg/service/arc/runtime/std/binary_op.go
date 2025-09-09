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
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
)

type operator struct {
	base
	compare func(a, b uint64) uint64
	a       *uint64
	b       *uint64
}

func (n *operator) Next(ctx context.Context, value stage.Value) {
	if value.Param == "a" {
		n.a = &value.Value
	} else {
		n.b = &value.Value
	}
	if n.a != nil && n.b != nil {
		n.outputHandler(ctx, stage.Value{
			Param: "output",
			Type:  types.U8{},
			Value: n.compare(*n.a, *n.b),
		})
	}
}

func createBinaryOpFactory(compare func(a, b uint64) uint64) Constructor {
	return func(ctx context.Context, node ir.Node) (stage.Stage, error) {
		return &operator{compare: compare}, nil
	}
}
