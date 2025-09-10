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
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/x/query"
)

type Constructor = func(
	ctx context.Context,
	node arc.Node,
) (stage.Stage, error)

var factories = map[string]Constructor{
	"ge": createBinaryOpFactory(func(a, b uint64) uint64 {
		return lo.Ternary[uint64](a >= b, 1, 0)
	}),
	"gt": createBinaryOpFactory(func(a, b uint64) uint64 {
		return lo.Ternary[uint64](a > b, 1, 0)
	}),
	"le": createBinaryOpFactory(func(a, b uint64) uint64 {
		return lo.Ternary[uint64](a <= b, 1, 0)
	}),
	"lt": createBinaryOpFactory(func(a, b uint64) uint64 {
		return lo.Ternary[uint64](a <= b, 1, 0)
	}),
	"eq": createBinaryOpFactory(func(a, b uint64) uint64 {
		return lo.Ternary[uint64](a == b, 1, 0)
	}),
	"on":         createChannelSource,
	"stable_for": createStableFor,
}

var resolver = ir.MapResolver{
	"ge":         symbolGE,
	"gt":         symbolGT,
	"le":         symbolLE,
	"lt":         symbolLT,
	"eq":         symbolEQ,
	"ne":         symbolNE,
	"on":         symbolChannelSource,
	"select":     symbolSelect,
	"stable_for": symbolStableFor,
}

func Create(ctx context.Context, node ir.Node) (stage.Stage, error) {
	v, ok := factories[node.Key]
	if !ok {
		return nil, query.NotFound
	}
	return v(ctx, node)
}
