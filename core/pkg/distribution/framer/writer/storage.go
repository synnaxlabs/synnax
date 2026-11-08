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

	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
)

// virtualKeys returns the keys of the virtual channels among keys.
func virtualKeys(
	ctx context.Context,
	db *ts.DB,
	keys []ts.ChannelKey,
) ([]ts.ChannelKey, error) {
	channels, err := db.RetrieveChannels(ctx, keys...)
	if err != nil {
		return nil, err
	}
	var virtual []ts.ChannelKey
	for _, ch := range channels {
		if ch.Virtual {
			virtual = append(virtual, ch.Key)
		}
	}
	return virtual, nil
}

// unindexedStamper times each sample of a channel with no index 1ns after the one
// before it in the frame, from the host's clock. It is not safe for concurrent use.
type unindexedStamper[K types.SizedNumeric] struct {
	slots map[K]*stampSlot
	frame uint64
}

// stampSlot holds where a channel's last series ended, and in which frame.
type stampSlot struct {
	frame uint64
	end   telem.TimeStamp
}

func newUnindexedStamper[K types.SizedNumeric](keys []K) *unindexedStamper[K] {
	slots := make(map[K]*stampSlot, len(keys))
	for _, k := range keys {
		slots[k] = &stampSlot{}
	}
	return &unindexedStamper[K]{slots: slots}
}

func (st *unindexedStamper[K]) stamp(fr telem.Frame[K]) {
	if len(st.slots) == 0 {
		return
	}
	st.frame++
	now := telem.Now()
	for rawI, k := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		slot, ok := st.slots[k]
		if !ok {
			continue
		}
		start := now
		if slot.frame == st.frame {
			start = slot.end
		}
		s := fr.RawSeriesAt(rawI)
		s.TimeRange = start.SpanRange(telem.TimeSpan(s.Len()) * telem.Nanosecond)
		slot.frame, slot.end = st.frame, s.TimeRange.End
		fr.SetRawSeriesAt(rawI, s)
	}
}

func newRequestTranslator(
	virtual []ts.ChannelKey,
) func(ctx context.Context, in Request) (ts.WriterRequest, bool, error) {
	st := newUnindexedStamper(virtual)
	return func(ctx context.Context, in Request) (ts.WriterRequest, bool, error) {
		fr := in.Frame.ToStorage()
		if in.Command == CommandWrite {
			st.stamp(fr)
		}
		return ts.WriterRequest{
			Command: ts.WriterCommand(in.Command),
			Frame:   fr,
			Config:  in.Config.toStorage(),
			SeqNum:  in.SeqNum,
		}, true, nil
	}
}

func newResponseTranslator(
	host node.Key,
) func(ctx context.Context, in ts.WriterResponse) (Response, bool, error) {
	return func(ctx context.Context, in ts.WriterResponse) (Response, bool, error) {
		return Response{
			Command:    Command(in.Command),
			SeqNum:     in.SeqNum,
			NodeKey:    host,
			End:        in.End,
			Authorized: in.Authorized,
			Err:        in.Err,
		}, true, nil
	}
}
