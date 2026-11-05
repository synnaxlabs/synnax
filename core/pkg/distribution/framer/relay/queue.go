// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay

import (
	"slices"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/telem"
)

// DropOldest is a confluence.QueuePolicy that holds frames up to Capacity. Past it, it
// drops the oldest responses that hold frames, but never the newest response. Before
// the next response with a frame, it adds a notice: a response with no frame whose
// Dropped holds the channels of the dropped frames. It merges adjacent notices, so a
// stalled consumer holds a bounded number of them.
type DropOldest struct {
	Capacity telem.Size
	size     telem.Size
}

var _ confluence.QueuePolicy[Response] = (*DropOldest)(nil)

// Push implements confluence.QueuePolicy.
func (d *DropOldest) Push(held []Response, res Response) []Response {
	held = append(held, res)
	if d.size += res.Frame.Size(); d.size <= d.Capacity {
		return held
	}
	var (
		kept    = held[:0]
		last    = len(held) - 1
		dropped channel.Keys
	)
	// Each kept notice follows at least one dropped response, so kept never passes the
	// read position of held.
	for i, r := range held {
		s := r.Frame.Size()
		if d.size > d.Capacity && s > 0 && i < last {
			d.size -= s
			dropped = append(dropped, r.Frame.KeysSlice()...)
			continue
		}
		if len(dropped) > 0 && (s > 0 || i == last) {
			if n := len(kept) - 1; n >= 0 && isNotice(kept[n]) {
				kept[n].Dropped = mergeKeys(kept[n].Dropped, dropped)
			} else {
				kept = append(kept, Response{Dropped: mergeKeys(nil, dropped)})
			}
			dropped = nil
		}
		kept = append(kept, r)
	}
	clear(held[len(kept):])
	return kept
}

func isNotice(res Response) bool { return res.Frame.Empty() && len(res.Dropped) > 0 }

func mergeKeys(a, b channel.Keys) channel.Keys {
	merged := slices.Concat(a, b)
	slices.Sort(merged)
	return slices.Compact(merged)
}

// Pop implements confluence.QueuePolicy.
func (d *DropOldest) Pop(res Response) { d.size -= res.Frame.Size() }
