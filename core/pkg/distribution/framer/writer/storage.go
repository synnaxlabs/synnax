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

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
)

// newStorage opens the session that writes the stored channels of cfg.
func (s *Service) newStorage(ctx context.Context, cfg Config) (StreamWriter, error) {
	channels, err := s.cfg.TS.RetrieveChannels(ctx, cfg.Keys.Uint32()...)
	if err != nil {
		return nil, err
	}
	var (
		virtual []cesium.ChannelKey
		indexes = make(map[cesium.ChannelKey]cesium.ChannelKey, len(channels))
	)
	for _, ch := range channels {
		if ch.Virtual {
			virtual = append(virtual, ch.Key)
		} else if !ch.IsIndex && ch.Index != 0 {
			indexes[ch.Key] = ch.Index
		}
	}
	var autoIndex *indexStamper
	if *cfg.AutoIndex {
		cfg = withIndexes(cfg, indexes)
		autoIndex = newIndexStamper(cfg.Keys.Uint32(), indexes, cfg.Start)
	}
	release, err := s.claims.hold(cfg.Keys)
	if err != nil {
		return nil, err
	}
	w, err := s.cfg.TS.OpenWriter(ctx, cfg.toStorage())
	if err != nil {
		release()
		return nil, err
	}
	return s.newSession(w, cfg, virtual, autoIndex, release), nil
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
