// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package event

import (
	"context"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/types"
)

type operator struct {
	confluence.MultiSink[spec.Value]
	confluence.AbstractUnarySource[spec.Value]
	x       *spec.Value
	y       *spec.Value
	compare func(a, b float64) bool
}

func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	case int16:
		return float64(n)
	case int8:
		return float64(n)
	case uint64:
		return float64(n)
	case uint32:
		return float64(n)
	case uint16:
		return float64(n)
	case uint8:
		return float64(n)
	case float64:
		return n
	case float32:
		return float64(n)
	case uint:
		return float64(n)
	default:
		panic("unsupported conversion")
	}
}

func newComparison(
	_ context.Context,
	p *plumber.Pipeline,
	_ spec.Config,
	node spec.Node,
) (bool, error) {
	if !strings.HasPrefix(node.Type, spec.ComparisonPrefix) {
		return false, nil
	}
	c := &operator{}
	if strings.HasSuffix(node.Type, spec.ComparisonGTESuffix) {
		c.compare = func(a, b float64) bool { return a >= b }
	} else if strings.HasSuffix(node.Type, spec.ComparisonLTESuffix) {
		c.compare = func(a, b float64) bool { return a <= b }
	} else if strings.HasSuffix(node.Type, spec.ComparisonLTSuffix) {
		c.compare = func(a, b float64) bool { return a < b }
	} else if strings.HasSuffix(node.Type, spec.ComparisonGTSuffix) {
		c.compare = func(a, b float64) bool { return a > b }
	} else if strings.HasSuffix(node.Type, spec.ComparisonEQSuffix) {
		c.compare = func(a, b float64) bool { return a == b }
	}
	plumber.SetSegment[spec.Value, spec.Value](p, address.Address(node.Key), c)
	c.Sink = c.sink
	return true, nil
}

func (n *operator) sink(ctx context.Context, origin address.Address, value spec.Value) error {
	if origin == "x" {
		n.x = &value
	}
	if origin == "y" {
		n.y = &value
	}
	if n.y == nil || n.x == nil {
		return nil
	}
	res := n.compare(toFloat64(n.x.Value), toFloat64(n.y.Value))
	return signal.SendUnderContext(ctx, n.Out.Inlet(), spec.Value{
		DataType: "uint8",
		Value:    types.BoolToUint8(res),
	})
}
