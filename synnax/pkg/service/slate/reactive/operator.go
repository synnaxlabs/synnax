// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package reactive

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
	nodeType string
	x        *spec.Value
	y        *spec.Value
	compare  func(a, b float64) bool
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

func newComparison(_ context.Context, cfg factoryConfig) (bool, error) {
	if !strings.HasPrefix(cfg.node.Type, spec.OperatorPrefix) {
		return false, nil
	}
	c := &operator{}
	if strings.HasSuffix(cfg.node.Type, spec.OperatorGTESuffix) {
		c.compare = func(a, b float64) bool { return a >= b }
	} else if strings.HasSuffix(cfg.node.Type, spec.OperatorLTESuffix) {
		c.compare = func(a, b float64) bool { return a <= b }
	} else if strings.HasSuffix(cfg.node.Type, spec.OperatorLTSuffix) {
		c.compare = func(a, b float64) bool { return a < b }
	} else if strings.HasSuffix(cfg.node.Type, spec.OperatorGTSuffix) {
		c.compare = func(a, b float64) bool { return a > b }
	} else if strings.HasSuffix(cfg.node.Type, spec.OperatorEQSuffix) {
		c.compare = func(a, b float64) bool { return a == b }
	} else if strings.HasSuffix(cfg.node.Type, spec.OperatorAndSuffix) {
		c.compare = func(a, b float64) bool { return a == 1 && b == 1 }
	} else if strings.HasSuffix(cfg.node.Type, spec.OperatorOrSuffix) {
		c.compare = func(a, b float64) bool { return a == 1 || b == 1 }
	} else if strings.HasSuffix(cfg.node.Type, spec.OperatorNotSuffix) {
		c.compare = func(a, b float64) bool { return a == 0 }
	}
	plumber.SetSegment[spec.Value, spec.Value](cfg.pipeline, address.Address(cfg.node.Key), c)
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
	if n.x == nil {
		return nil
	}
	var res bool
	if strings.HasSuffix(n.nodeType, spec.OperatorNotSuffix) {
		res = n.compare(toFloat64(n.x), 0)
	} else {
		if n.y == nil {
			return nil
		}
		res = n.compare(toFloat64(n.x.Value), toFloat64(n.y.Value))
	}
	return signal.SendUnderContext(ctx, n.Out.Inlet(), spec.Value{
		DataType: "uint8",
		Value:    types.BoolToUint8(res),
	})
}
