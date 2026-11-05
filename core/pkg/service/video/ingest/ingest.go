// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package ingest holds what the tasks that read video streams share.
package ingest

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Channels returns the channels that a task writes the frames of the video channel key
// to: its index and the channel when the channel persists, or the channel alone when it
// is virtual. It returns validate.ErrValidation for a channel that does not exist or is
// not video.
func Channels(
	ctx context.Context,
	svc *channel.Service,
	key channel.Key,
) (channel.Keys, error) {
	var ch channel.Channel
	err := svc.NewRetrieve().Where(channel.MatchKeys(key)).Entry(&ch).Exec(ctx, nil)
	if errors.Is(err, query.ErrNotFound) {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"channel: %s does not exist",
			key,
		)
	}
	if err != nil {
		return nil, err
	}
	if ch.DataType != telem.VideoT {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"channel: %s has data type %s, not video", ch, ch.DataType,
		)
	}
	if ch.Virtual {
		return channel.Keys{key}, nil
	}
	return channel.Keys{ch.Index(), key}, nil
}

// Frame returns the frame that writes frames to the channels from Channels. stamps
// holds the time of each frame, and is ignored for a virtual channel.
func Frame(
	keys channel.Keys,
	frames []telem.VideoFrame,
	stamps []telem.TimeStamp,
) framer.Frame {
	series := telem.NewVideoSeries(frames)
	if len(keys) == 1 {
		return frame.NewUnary(keys[0], series)
	}
	return frame.NewMulti(keys, []telem.Series{telem.NewSeriesV(stamps...), series})
}
