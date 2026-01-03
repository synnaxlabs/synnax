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
	"sync/atomic"

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type freeWriteAlignments struct {
	mu         sync.Mutex
	alignments map[channel.Key]*atomic.Uint32
}

func (f *freeWriteAlignments) increment(key channel.Key) telem.Alignment {
	a := f.alignments[key]
	if a == nil {
		a = new(atomic.Uint32)
		a.Store(cesium.ZeroLeadingAlignment)
		f.alignments[key] = a
	}
	return telem.NewAlignment(a.Add(1), 0)
}

func (s *Service) newFree(
	mode Mode,
	sync bool,
	channels []channel.Channel,
) StreamWriter {
	w := &freeWriter{
		freeWrites: s.cfg.FreeWrites,
		mode:       mode,
		sync:       sync,
		indexes:    make(map[channel.Key]channel.Key),
		alignments: make(map[channel.Key]telem.Alignment),
	}
	for _, ch := range channels {
		if ch.Free() && ch.Index() != 0 {
			w.indexes[ch.Key()] = ch.Index()
		}
	}
	s.freeWriteAlignments.mu.Lock()
	defer s.freeWriteAlignments.mu.Unlock()
	for _, idx := range w.indexes {
		if _, ok := w.alignments[idx]; !ok {
			w.alignments[idx] = s.freeWriteAlignments.increment(idx)
		}
	}
	w.Transform = w.transform
	return w
}

// freeWriter is used to write data for free channels into the distribution relay.
type freeWriter struct {
	indexes    map[channel.Key]channel.Key
	alignments map[channel.Key]telem.Alignment
	confluence.LinearTransform[Request, Response]
	// freeWrites is the inlet for communicating free frames to the relay
	freeWrites confluence.Inlet[relay.Response]
	// mode is the mode of the writer.
	mode Mode
	// sync is true if the writer should receive acknowledgements for all requires,
	// including Write commands.
	sync bool
}

func (w *freeWriter) alignFrame(fr core.Frame) core.Frame {
	var (
		idx channel.Key
		ok  bool
	)
	for rawI, s := range fr.RawSeries() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		s.Alignment = w.alignments[w.indexes[fr.RawKeyAt(rawI)]]
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
	return fr
}

func (w *freeWriter) transform(ctx context.Context, req Request) (res Response, ok bool, err error) {
	if req.Command == Write && w.mode.Stream() {
		if err = signal.SendUnderContext(
			ctx, w.freeWrites.Inlet(),
			relay.Response{Frame: w.alignFrame(req.Frame)},
		); err != nil || !w.sync {
			return
		}
	}
	return Response{Command: req.Command, SeqNum: req.SeqNum, Authorized: true}, true, nil
}
