// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/set"
)

// newSwitch returns the switch that sends the series of the stored channels to the
// storage writer and every other series to the free writer.
func newSwitch(
	stored set.Set[channel.Key],
) func(context.Context, Request, map[address.Address]Request) error {
	return func(_ context.Context, r Request, oReqs map[address.Address]Request) error {
		var s, f frame.Frame
		for key, series := range r.Frame.Entries() {
			if stored.Contains(key) {
				s = s.Append(key, series)
			} else {
				f = f.Append(key, series)
			}
		}
		sr, fr := r, r
		sr.Frame, fr.Frame = s, f
		oReqs[storageWriterAddr] = sr
		oReqs[freeWriterAddr] = fr
		return nil
	}
}
