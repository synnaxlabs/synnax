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

	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

type Status = calculation.Status

// Report saves st for the calculation of the channel with the given key and returns
// true when it saved st.
type Report = func(ctx context.Context, key channel.Key, st Status) bool

type Group []*Calculator

func (g Group) ReadFrom() channel.Keys {
	keys := make(set.Set[channel.Key])
	calcKeys := make(set.Set[channel.Key])
	for _, c := range g {
		keys.Add(c.ReadFrom()...)
		calcKeys.Add(c.Channel().Key(), c.Channel().Index())
	}
	return keys.Difference(calcKeys).Slice()
}

func (g Group) WriteTo() channel.Keys {
	keys := make(set.Set[channel.Key])
	for _, c := range g {
		keys.Add(c.WriteTo()...)
	}
	return keys.Slice()
}

// Next runs every calculator in the group on input. It reports an error status for each
// calculator whose step fails. After a failed step, it reports a success status on each
// clean step with output until report saves the success.
func (g Group) Next(
	ctx context.Context,
	input framer.Frame,
	report Report,
) (output framer.Frame, changed bool) {
	output = input.ShallowCopy()
	for _, c := range g {
		next, changedLocal, err := c.Next(ctx, output, output)
		if err != nil {
			c.failed = true
			report(ctx, c.Channel().Key(), Status{
				Name:        c.Channel().Name,
				Variant:     status.VariantError,
				Message:     "Failed to run calculation",
				Description: err.Error(),
			})
			continue
		}
		output = next
		if !changedLocal {
			continue
		}
		changed = true
		if c.failed {
			c.failed = !report(ctx, c.Channel().Key(), Status{
				Name:    c.Channel().Name,
				Variant: status.VariantSuccess,
				Message: "Calculation running",
			})
		}
	}
	return output, changed
}

func (g Group) Close() error {
	var err error
	for _, calc := range g {
		err = errors.Join(err, calc.Close())
	}
	return err
}
