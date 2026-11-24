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
	ws, err := s.clockWriters(ctx, cfg, channels)
	if err != nil {
		release()
		return nil, err
	}
	unlock := lockEntries(ws)
	defer unlock()
	g, transfers, err := s.controls.openGates(cfg, channels)
	if err != nil {
		release()
		return nil, err
	}
	if err = join(ctx, s.cfg.TS, ws); err == nil {
		err = s.openOwn(ctx, cfg, ws)
	}
	if err != nil {
		err = errors.Join(err, closeClocks(ctx, ws))
		transfers = append(transfers, g.release()...)
		followAll(ctx, s.cfg.TS, ws)
		s.controls.notify(transfers)
		release()
		named, _ := s.controls.nameHolder(err)
		return nil, named
	}
	s.controls.notify(transfers)
	return s.newSession(ws, cfg, autoIndex, release, g), nil
}

// join adds each sharing handle of ws to the entry of its clock, and moves the writer
// of each clock to its leader. It returns the error of a writer that a handle of ws
// needed and that failed to open. The caller holds the entries' locks.
func join(ctx context.Context, ts *cesium.DB, ws []*clockWriter) error {
	for _, c := range ws {
		if c.shares {
			c.entry.members[c.member.subject] = c.member
		}
	}
	followAll(ctx, ts, ws)
	for _, c := range ws {
		if c.shares && c.member.err != nil {
			return c.member.err
		}
	}
	return nil
}

// clockWriters returns the handle of the session of cfg on each clock of channels,
// which are the stored channels of cfg. A session that persists and streams, and that
// writes a clock, shares the clock's writer with the other sessions on it. The other
// handles hold their own writer, which openOwn opens. A session that persists and
// streams the data of a clock and not the clock also joins the clock's entry, which
// keeps one live claim per key between its writers.
func (s *Service) clockWriters(
	ctx context.Context,
	cfg Config,
	channels []cesium.Channel,
) ([]*clockWriter, error) {
	var (
		clocks  []cesium.ChannelKey
		keys    = make(map[cesium.ChannelKey][]cesium.ChannelKey, 1)
		byKey   = make(map[cesium.ChannelKey]cesium.Channel, len(channels))
		ws      = make([]*clockWriter, 0, 1)
		sharing = cfg.Mode == ModePersistStream
	)
	for _, ch := range channels {
		clock := ch.Clock()
		if _, ok := keys[clock]; !ok {
			clocks = append(clocks, clock)
		}
		keys[clock] = append(keys[clock], ch.Key)
		byKey[ch.Key] = ch
	}
	for _, clock := range clocks {
		c := &clockWriter{
			ts:    s.cfg.TS,
			clock: clock,
			member: &clockMember{
				subject: cfg.ControlSubject.Key,
				keys:    keys[clock],
				start:   cfg.Start,
				end:     cfg.Start,
			},
		}
		ch, writesClock := byKey[clock]
		switch {
		case writesClock && cfg.Mode != ModePersistOnly:
			entry, err := s.controls.clockEntry(ch)
			if err != nil {
				return nil, err
			}
			c.entry, c.shares = entry, sharing
		case !writesClock && sharing:
			clockCh, err := s.cfg.TS.RetrieveChannel(ctx, clock)
			if err != nil {
				return nil, err
			}
			if c.entry, err = s.controls.clockEntry(clockCh); err != nil {
				return nil, err
			}
			c.claims = true
		}
		ws = append(ws, c)
	}
	return ws, nil
}

// openOwn opens the writer of each handle of ws that does not share its clock's writer.
// A handle in a clock's entry claims only the keys that no writer of the entry claims,
// and opens no writer when another writer claims all of them. A failed open returns its
// error, and the caller closes the writers that opened. The caller holds the entries'
// locks.
func (s *Service) openOwn(ctx context.Context, cfg Config, ws []*clockWriter) error {
	for _, c := range ws {
		if c.shares {
			continue
		}
		keys := c.member.keys
		if c.claims {
			if keys = c.entry.unclaimed(keys); len(keys) == 0 {
				continue
			}
		}
		w, err := s.cfg.TS.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    c.clock,
			Start:    cfg.Start,
			Bounded:  cfg.Mode == ModePersistOnly,
			Unstored: cfg.Mode == ModeStreamOnly,
		})
		if err != nil {
			return err
		}
		c.own = w
		if c.claims {
			c.claim(keys, keys)
		}
	}
	return nil
}

// closeClocks closes the own writer of each handle of ws, and leaves each shared clock.
// The caller holds the entries' locks.
func closeClocks(ctx context.Context, ws []*clockWriter) error {
	var err error
	for _, c := range ws {
		switch {
		case c.shares:
			err = errors.Join(err, c.entry.leave(ctx, c.member))
		case c.claims:
			err = errors.Join(err, c.leave())
		case c.own != nil:
			err = errors.Join(err, c.own.Close())
		}
	}
	return err
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
