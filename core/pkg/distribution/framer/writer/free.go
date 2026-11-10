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
	"github.com/synnaxlabs/cesium"
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
	counters map[channel.Key]*cesium.AlignmentCounter
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
			c = cesium.NewAlignmentCounter()
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

func (s *Service) newFree(
	mode Mode,
	sync bool,
	keys []channel.Key,
	freeIndexes map[channel.Key]channel.Key,
	alignments map[channel.Key]telem.Alignment,
	group uint32,
) StreamWriter {
	if freeIndexes == nil {
		freeIndexes = make(map[channel.Key]channel.Key)
	}
	unindexed := lo.Filter(keys, func(k channel.Key, _ int) bool {
		_, indexed := freeIndexes[k]
		return !indexed
	})
	w := &freeWriter{
		freeWrites: s.cfg.FreeWrites,
		mode:       mode,
		sync:       sync,
		group:      group,
		indexes:    freeIndexes,
		alignments: alignments,
		stamper:    newUnindexedStamper(unindexed),
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
	// freeWrites is the inlet for communicating free frames to the relay
	freeWrites confluence.Inlet[relay.Response]
	// mode is the mode of the writer.
	mode Mode
	// sync is true if the writer should receive acknowledgments for all requests,
	// including Write commands.
	sync bool
	// group is the writer group identifier, used for server-side deduplication.
	group uint32
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
	return fr
}

func (w *freeWriter) transform(
	ctx context.Context,
	req Request,
) (res Response, ok bool, err error) {
	if req.Command == CommandWrite && w.mode.Stream() {
		if err = signal.SendUnderContext(
			ctx, w.freeWrites.Inlet(),
			relay.Response{Frame: w.alignFrame(req.Frame), Group: w.group},
		); err != nil || !w.sync {
			return res, ok, err
		}
	}
	return Response{
		Command:    req.Command,
		SeqNum:     req.SeqNum,
		Authorized: true,
	}, true, nil
}
