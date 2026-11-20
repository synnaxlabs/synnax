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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
)

// newStorage opens the session that writes the stored channels of cfg, which are
// channels.
func (s *Service) newStorage(
	ctx context.Context,
	cfg Config,
	channels []cesium.Channel,
) (StreamWriter, error) {
	indexes := make(map[cesium.ChannelKey]cesium.ChannelKey, len(channels))
	for _, ch := range channels {
		if !ch.IsIndex && ch.Index != 0 {
			indexes[ch.Key] = ch.Index
		}
	}
	var err error
	var autoIndex *indexStamper
	if *cfg.AutoIndex {
		cfg = withIndexes(cfg, indexes)
		autoIndex = newIndexStamper(cfg.Keys.Uint32(), indexes, cfg.Start)
		// The session also gates the indexes that withIndexes adds.
		if channels, err = s.cfg.TS.RetrieveChannels(
			ctx,
			cfg.Keys.Uint32()...,
		); err != nil {
			return nil, err
		}
	}
	release, err := s.claims.hold(cfg.Keys)
	if err != nil {
		return nil, err
	}
	g, transfers, err := s.controls.openGates(cfg, channels)
	if err != nil {
		release()
		return nil, err
	}
	ws, err := s.openClocks(ctx, cfg, channels)
	if err != nil {
		g.release()
		release()
		named, _ := s.controls.nameHolder(err)
		return nil, named
	}
	s.controls.notify(transfers)
	return s.newSession(ws, cfg, autoIndex, release, g), nil
}

// openClocks opens one Cesium writer for each clock of channels, which are the stored
// channels of cfg. On an error it closes the writers it opened.
func (s *Service) openClocks(
	ctx context.Context,
	cfg Config,
	channels []cesium.Channel,
) (ws []*cesium.Writer, err error) {
	var (
		clocks []cesium.ChannelKey
		keys   = make(map[cesium.ChannelKey][]cesium.ChannelKey, 1)
	)
	for _, ch := range channels {
		clock := ch.Clock()
		if _, ok := keys[clock]; !ok {
			clocks = append(clocks, clock)
		}
		keys[clock] = append(keys[clock], ch.Key)
	}
	defer func() {
		if err != nil {
			for _, w := range ws {
				err = errors.Join(err, w.Close())
			}
			ws = nil
		}
	}()
	ws = make([]*cesium.Writer, 0, len(clocks))
	for _, clock := range clocks {
		var w *cesium.Writer
		if w, err = s.cfg.TS.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys[clock],
			Clock:    clock,
			Start:    cfg.Start,
			Bounded:  cfg.Mode == ModePersistOnly,
			Unstored: cfg.Mode == ModeStreamOnly,
		}); err != nil {
			return ws, err
		}
		ws = append(ws, w)
	}
	return ws, nil
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
