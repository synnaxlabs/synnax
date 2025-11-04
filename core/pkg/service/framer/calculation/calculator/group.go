// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculator

import (
	"context"
	"go/types"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/status"
)

type Status = status.Status[types.Nil]

type Group struct {
	calculators []*Calculator
}

func (g *Group) Add(c *Calculator) {
	g.calculators = append(g.calculators, c)
}

func (g *Group) ReadFrom() channel.Keys {
	keys := make(set.Set[channel.Key])
	for _, c := range g.calculators {
		keys.Add(c.ReadFrom()...)
	}
	return keys.Keys()
}

func (g *Group) Remove(c *Calculator) {
	slices.Delete(g.calculators, slices.Index(g.calculators, c), 1)
}

func (g *Group) Next(
	ctx context.Context,
	input framer.Frame,
) (output framer.Frame, changed bool, err error) {
	var (
		changedLocal bool
		localErr     error
	)
	for _, c := range g.calculators {
		output, changedLocal, localErr = c.Next(ctx, input, output)
		if localErr != nil {
			err = localErr
			continue
		}
		if changedLocal {
			changed = changedLocal
		}
	}
	return output, changed, err
}

func (g *Group) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	for _, calc := range g.calculators {
		c.Exec(calc.Close)
	}
	return c.Error()
}
