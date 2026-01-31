// Copyright 2026 Synnax Labs, Inc.
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
	"fmt"
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/status"
)

type Status = status.Status[types.Nil]

type Group []*Calculator

func (g Group) ReadFrom() channel.Keys {
	keys := make(set.Set[channel.Key])
	calcKeys := make(set.Set[channel.Key])
	for _, c := range g {
		keys.Add(c.ReadFrom()...)
		calcKeys.Add(c.Channel().Key(), c.Channel().Index())
	}
	return set.Difference(keys, calcKeys).Keys()
}

func (g Group) WriteTo() channel.Keys {
	keys := make(set.Set[channel.Key])
	for _, c := range g {
		keys.Add(c.WriteTo()...)
	}
	return keys.Keys()
}

func (g Group) Next(
	ctx context.Context,
	input framer.Frame,
) (output framer.Frame, changed bool, statuses []Status) {
	var (
		changedLocal bool
		err          error
	)
	output = input.ShallowCopy()
	for _, c := range g {
		output, changedLocal, err = c.Next(ctx, output, output)
		if err != nil {
			statuses = append(statuses, Status{
				Key:         c.Channel().Key().String(),
				Variant:     status.VariantError,
				Message:     fmt.Sprintf("calculation for %s failed", c.Channel()),
				Description: err.Error(),
			})
			continue
		}
		if changedLocal {
			changed = changedLocal
		}
	}
	return output, changed, statuses
}

func (g Group) Close() error {
	var a errors.Accumulator
	for _, calc := range g {
		a.Exec(calc.Close)
	}
	return a.Error()
}
