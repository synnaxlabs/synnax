// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// LastStored returns the last timestamp stored in index, or zero when index holds no
// data. It reads index alone, because an iterator over channels on several Cores
// reports the seek result of only one of them.
func LastStored(
	ctx context.Context,
	frm *framer.Service,
	index channel.Key,
) (telem.TimeStamp, error) {
	it, err := frm.OpenIterator(ctx, framer.IteratorConfig{
		Keys:      channel.Keys{index},
		Bounds:    telem.TimeRangeMax,
		ChunkSize: 1,
	})
	if err != nil {
		return 0, err
	}
	var last telem.TimeStamp
	if it.SeekLast() && it.Prev(framer.IteratorAutoSpan) {
		for _, s := range it.Value().Get(index).Series {
			if s.Len() > 0 {
				last = max(last, s.ValueAt[telem.TimeStamp](-1))
			}
		}
	}
	if err = errors.Combine(it.Error(), it.Close()); err != nil {
		return 0, err
	}
	return last, nil
}
