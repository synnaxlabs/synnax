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
	"sync"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

// freeWriteAlignments tracks a monotonically increasing alignment counter for each free
// index channel and each free channel with no index, shared across every writer opened
// on the service.
type freeWriteAlignments struct {
	counters map[channel.Key]*telem.AlignmentCounter
	mu       sync.Mutex
}

// reserve returns the first alignment of a new domain for the index channel of each key
// in keys, or for the key itself when indexes holds no index for it.
func (f *freeWriteAlignments) reserve(
	keys []channel.Key,
	indexes map[channel.Key]channel.Key,
) (map[channel.Key]telem.Alignment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	alignments := make(map[channel.Key]telem.Alignment, len(keys))
	for _, key := range keys {
		idx, indexed := indexes[key]
		if !indexed {
			idx = key
		}
		if _, ok := alignments[idx]; ok {
			continue
		}
		c, ok := f.counters[idx]
		if !ok {
			c = telem.NewAlignmentCounter(telem.LeadingDomain)
			f.counters[idx] = c
		}
		a, err := c.Next()
		if err != nil {
			return nil, err
		}
		alignments[idx] = a
	}
	return alignments, nil
}

// storedStamps stamps the stored channels of a stream-only writer as Cesium stamps
// them. A series takes the time range of its clock's series in the frame, and a frame
// of
// data without its clock moves the alignment of the clock by the data length. It is not
// safe for concurrent use.
type storedStamps struct {
	// autoIndex is nil when the writer does not stamp its indexes.
	autoIndex *indexStamper
	// clockOf maps each stored channel to its clock.
	clockOf map[channel.Key]channel.Key
	// scans holds the scan of the current frame for each clock.
	scans map[channel.Key]*clockScan
}

// clockScan is the scan of one frame for a clock: the time range of the clock's series,
// whether the frame holds that series, and the length of the first data series.
type clockScan struct {
	timeRange telem.TimeRange
	indexed   bool
	dataLen   int64
}

func newStoredStamps(
	clockOf map[channel.Key]channel.Key,
	autoIndex *indexStamper,
) *storedStamps {
	scans := make(map[channel.Key]*clockScan)
	for _, clock := range clockOf {
		if _, ok := scans[clock]; !ok {
			scans[clock] = &clockScan{}
		}
	}
	return &storedStamps{autoIndex: autoIndex, clockOf: clockOf, scans: scans}
}

// stamp sets the time range of each stored series of fr, and moves the alignment of
// each clock whose data fr holds without the clock's series.
func (st *storedStamps) stamp(
	fr frame.Frame,
	alignments map[channel.Key]telem.Alignment,
) {
	for _, sc := range st.scans {
		*sc = clockScan{}
	}
	for rawI, key := range fr.RawKeys() {
		clock, ok := st.clockOf[key]
		if !ok || fr.ShouldExcludeRaw(rawI) {
			continue
		}
		sc, s := st.scans[clock], fr.RawSeriesAt(rawI)
		switch {
		case key == clock && s.Len() > 0:
			sc.indexed = true
			sc.timeRange = telem.TimeRange{
				Start: s.ValueAt[telem.TimeStamp](0),
				End:   s.ValueAt[telem.TimeStamp](int(s.Len()-1)) + 1,
			}
		case key != clock && sc.dataLen == 0:
			sc.dataLen = s.Len()
		}
	}
	for rawI, key := range fr.RawKeys() {
		clock, ok := st.clockOf[key]
		if !ok || fr.ShouldExcludeRaw(rawI) || !st.scans[clock].indexed {
			continue
		}
		s := fr.RawSeriesAt(rawI)
		s.TimeRange = st.scans[clock].timeRange
		fr.SetRawSeriesAt(rawI, s)
	}
	for clock, sc := range st.scans {
		if !sc.indexed && sc.dataLen > 0 {
			alignments[clock] = alignments[clock].AddSamples(uint32(sc.dataLen))
		}
	}
}

func (s *Service) newFree(
	mode Mode,
	sync bool,
	keys []channel.Key,
	freeIndexes map[channel.Key]channel.Key,
	alignments map[channel.Key]telem.Alignment,
	group uint32,
	g *gates,
	release func(),
	stored *storedStamps,
) StreamWriter {
	if freeIndexes == nil {
		freeIndexes = make(map[channel.Key]channel.Key)
	}
	unindexed := lo.Filter(keys, func(k channel.Key, _ int) bool {
		_, indexed := freeIndexes[k]
		return !indexed
	})
	w := &freeWriter{
		writes:     s.cfg.Writes,
		mode:       mode,
		sync:       sync,
		group:      group,
		indexes:    freeIndexes,
		alignments: alignments,
		stamper:    newUnindexedStamper(unindexed),
		gates:      g,
		controls:   s.controls,
		release:    release,
		stored:     stored,
	}
	w.Transform = w.transform
	return w
}

// freeWriter is used to write data for free channels into the distribution relay.
type freeWriter struct {
	// indexes maps each free channel with an index to that index channel's key.
	indexes map[channel.Key]channel.Key
	// alignments tracks the current alignment for each free index channel and each free
	// channel with no index.
	alignments map[channel.Key]telem.Alignment
	stamper    *unindexedStamper[channel.Key]
	confluence.LinearTransform[Request, Response]
	// writes is where the writer sends its frames for the relay to stream.
	writes confluence.Inlet[relay.Response]
	// mode is the mode of the writer.
	mode Mode
	// sync is true if the writer should receive acknowledgments for all requests,
	// including Write commands.
	sync bool
	// gates gate the writes of the writer on its control of each leased channel. Free
	// channels have no control, so gates is nil when the writer holds no leased
	// channel.
	gates    *gates
	controls *controls
	// release releases the writer's hold on its leased channels. It is nil for a writer
	// that holds none.
	release func()
	// stored is nil when the writer streams no stored channels.
	stored *storedStamps
	// group is the writer group identifier, used for server-side deduplication.
	group uint32
	// writeRefused is true when a write was refused for no control since the last
	// commit.
	writeRefused bool
}

// Flow implements confluence.Flow. A writer that holds leased channels releases its
// gates and its hold when it exits.
func (w *freeWriter) Flow(ctx signal.Context, opts ...confluence.Option) {
	if w.release == nil {
		w.LinearTransform.Flow(ctx, opts...)
		return
	}
	o := confluence.NewOptions(opts)
	o.AttachClosables(w.Out)
	ctx.Go(func(ctx context.Context) error {
		err := w.run(ctx)
		w.controls.notify(w.gates.release())
		w.release()
		return err
	}, o.Signal...)
}

// run serves requests until the inlet closes or ctx ends.
func (w *freeWriter) run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case req, ok := <-w.In.Outlet():
			if !ok {
				return nil
			}
			res, send, err := w.transform(ctx, req)
			if err != nil {
				return err
			}
			if !send {
				continue
			}
			if err = signal.SendUnderContext(ctx, w.Out.Inlet(), res); err != nil {
				return err
			}
		}
	}
}

// exclude returns fr without the leased channels that the writer does not control, and
// true when it left one out.
func (w *freeWriter) exclude(fr frame.Frame) (frame.Frame, bool) {
	if w.gates == nil {
		return fr, false
	}
	refused := w.gates.refused()
	if len(refused) == 0 {
		return fr, false
	}
	var keys channel.Keys
	for _, key := range fr.KeysSlice() {
		if refused.Contains(uint32(key)) {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return fr, false
	}
	return fr.ExcludeKeys(keys), true
}

func (w *freeWriter) alignFrame(fr frame.Frame) frame.Frame {
	w.stamper.stamp(fr.Frame)
	var (
		idx channel.Key
		ok  bool
	)
	for rawI, s := range fr.RawSeries() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		key := fr.RawKeyAt(rawI)
		if idx, ok = w.indexes[key]; ok {
			s.Alignment = w.alignments[idx]
		} else {
			s.Alignment = w.alignments[key]
			w.alignments[key] = s.Alignment.AddSamples(uint32(s.Len()))
		}
		fr.SetRawSeriesAt(rawI, s)
	}

	for rawI, rawKey := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		idx, ok = w.indexes[rawKey]
		if !ok || rawKey != idx {
			continue
		}
		s := fr.RawSeriesAt(rawI)
		w.alignments[idx] = w.alignments[idx].AddSamples(uint32(s.Len()))
	}
	if w.stored != nil {
		w.stored.stamp(fr, w.alignments)
	}
	return fr
}

func (w *freeWriter) transform(
	ctx context.Context,
	req Request,
) (res Response, ok bool, err error) {
	res = Response{Command: req.Command, SeqNum: req.SeqNum, Authorized: true}
	switch req.Command {
	case CommandWrite:
		fr := req.Frame
		if w.stored != nil && w.stored.autoIndex != nil {
			fr = frame.NewFromStorage(w.stored.autoIndex.stamp(fr.ToStorage()))
		}
		fr, refused := w.exclude(fr)
		w.writeRefused = w.writeRefused || refused
		if w.mode.Stream() {
			if err = signal.SendUnderContext(
				ctx, w.writes.Inlet(),
				relay.Response{Frame: w.alignFrame(fr), Group: w.group},
			); err != nil {
				return res, false, err
			}
		}
		res.Authorized = !refused
		return res, w.sync, nil
	case CommandCommit:
		res.Authorized = !w.writeRefused
		w.writeRefused = false
	case CommandSetAuthority:
		if w.gates == nil {
			res.Authorized = true
			break
		}
		var transfers []transfer
		transfers, res.Err = w.gates.setAuthority(req.Config)
		res.Authorized = res.Err == nil
		w.controls.notify(transfers)
	case CommandOpen, CommandEndDomain:
	}
	return res, true, nil
}
