// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package streamer

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/gop"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/telem"
)

// keyframeGate passes the frames of a video channel only from a point the viewer can
// decode from: a keyframe, or the cached frames that lead up to the channel's first
// live frame. It finds the start of a channel and each gap in it by alignment, and by
// drop notices.
type keyframeGate struct {
	confluence.LinearTransform[Response, Response]
	gop *gop.Cache
	// next is the alignment after the last frame passed for each video channel.
	next map[channel.Key]telem.Alignment
}

func newKeyframeGate(cache *gop.Cache) responseSegment {
	g := &keyframeGate{gop: cache, next: make(map[channel.Key]telem.Alignment)}
	g.Transform = g.transform
	return g
}

func (g *keyframeGate) transform(
	_ context.Context,
	res Response,
) (Response, bool, error) {
	for _, key := range res.Dropped {
		delete(g.next, key)
	}
	if !hasVideo(res.Frame) {
		return res, true, nil
	}
	out := frame.Alloc(res.Frame.Count())
	for key, s := range res.Frame.Entries() {
		if s.DataType != telem.VideoT {
			out = out.Append(key, s)
			continue
		}
		for _, passed := range g.pass(key, s) {
			out = out.Append(key, passed)
		}
	}
	res.Frame = out
	return res, !out.Empty(), nil
}

func (g *keyframeGate) pass(key channel.Key, s telem.Series) []telem.Series {
	if next, ok := g.next[key]; ok && gop.Continues(next, s.Alignment) {
		g.next[key] = s.AlignmentBounds().Upper
		return []telem.Series{s}
	}
	if bridge, ok := g.gop.Bridge(key, s.Alignment); ok {
		g.next[key] = s.AlignmentBounds().Upper
		return append(bridge, s)
	}
	for i := range s.Keyframes() {
		s = s.Slice(i, int(s.Len()))
		g.next[key] = s.AlignmentBounds().Upper
		return []telem.Series{s}
	}
	delete(g.next, key)
	return nil
}

func hasVideo(fr frame.Frame) bool {
	for s := range fr.Series() {
		if s.DataType == telem.VideoT {
			return true
		}
	}
	return false
}
