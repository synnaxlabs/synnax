// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/confluence"
)

// clocker sets the clock of each series that an iterator returns. The clock of a
// channel is its index, and zero for a channel with no index.
type clocker struct {
	confluence.LinearTransform[Response, Response]
	clocks map[channel.Key]uint32
}

func newClocker(channels []channel.Channel) responseSegment {
	c := &clocker{clocks: make(map[channel.Key]uint32, len(channels))}
	for _, ch := range channels {
		c.clocks[ch.Key()] = uint32(ch.Index())
	}
	c.Transform = c.transform
	return c
}

func (c *clocker) transform(_ context.Context, in Response) (Response, bool, error) {
	for i, key := range in.Frame.RawKeys() {
		s := in.Frame.RawSeriesAt(i)
		s.Clock = c.clocks[key]
		in.Frame.SetRawSeriesAt(i, s)
	}
	return in, true, nil
}
