// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment_test

import (
	"bytes"
	"encoding/binary"
	"math"
	"sort"
	"testing"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
)

// fuzzStamps returns rising stamps from b: each byte is a gap, and a byte above 0xF0
// repeats the gap before it 5 to 20 times, so that runs form.
func fuzzStamps(b []byte) []int64 {
	var (
		s   []int64
		x   int64
		gap int64 = 1
	)
	for _, c := range b {
		if c > 0xF0 && len(s) > 0 {
			for range int(c-0xF0) + 4 {
				x += gap
				s = append(s, x)
			}
			continue
		}
		gap = int64(c) + 1
		x += gap
		s = append(s, x)
	}
	return s
}

// FuzzProgressions checks that rising stamps encode and read back equal, and that a
// search finds the position that a linear search finds.
func FuzzProgressions(f *testing.F) {
	f.Add([]byte{1, 2, 3, 0xFF, 7, 0xF5, 0xF5, 9})
	f.Add(bytes.Repeat([]byte{4, 0xF8}, 300))
	f.Add([]byte{0, 0, 0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, b []byte) {
		stamps := fuzzStamps(b)
		if len(stamps) == 0 || len(stamps) > 1<<16 {
			return
		}
		h, s, fits, err := encode(xfs.NewMem(), stamps, 7, math.MaxInt64)
		if err != nil || !fits {
			t.Fatalf("encode of %d stamps: fits %v, %v", len(stamps), fits, err)
		}
		defer func() {
			if err := h.Close(); err != nil {
				t.Error(err)
			}
		}()
		rd := segment.NewReader(segment.NewCache(1 << 20))
		got := make([]byte, 8*len(stamps))
		if err = rd.Stamps(h, s, got, 0); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, stampBytes(stamps)) {
			t.Fatalf("stamps read back differ")
		}
		for _, x := range []int64{
			stamps[0] - 1, stamps[len(stamps)/2],
			stamps[len(stamps)/2] + 1, stamps[len(stamps)-1], stamps[len(stamps)-1] + 1,
		} {
			want := sort.Search(len(stamps), func(i int) bool { return stamps[i] >= x })
			lower, upper, err := rd.Search(h, s, x, 0, int64(len(stamps)))
			if err != nil {
				t.Fatal(err)
			}
			exact := want < len(stamps) && stamps[want] == x
			if upper != int64(want) || (lower == upper) != exact {
				t.Fatalf("search of %d gives %d to %d, not %d", x, lower, upper, want)
			}
		}
	})
}

// FuzzTable checks that the reader of a version 2 segment never panics on bytes whose
// blocks pass their checks, and that it returns positions inside the segment or an
// error that wraps place.ErrCorrupt.
func FuzzTable(f *testing.F) {
	h, s, _, err := encode(xfs.NewMem(), fuzzStamps([]byte{1, 2, 0xF9, 3, 9}), 7,
		math.MaxInt64)
	if err != nil {
		f.Fatal(err)
	}
	seed := make([]byte, s.Length)
	if err = segment.NewReader(nil).Read(h, s, seed, 0); err != nil {
		f.Fatal(err)
	}
	f.Add(append(seed[s.Length-20:], seed[:200]...), uint64(5))
	f.Add(make([]byte, 40), uint64(0))
	f.Fuzz(func(t *testing.T, b []byte, ts uint64) {
		if len(b) < 20 || len(b) > segment.BlockSize {
			return
		}
		stream := make([]byte, segment.BlockSize)
		copy(stream, b[20:])
		copy(stream[segment.BlockSize-20:], b[:20])
		budget, err := segment.NewBudget(4, 4, segment.GoExecutor)
		if err != nil {
			t.Fatal(err)
		}
		h, err := budget.Create(xfs.NewMem(), "9.seg")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := h.Close(); err != nil {
				t.Error(err)
			}
		}()
		w := segment.NewWriter(h, clockCfg)
		if err = w.Write(stream); err != nil {
			t.Fatal(err)
		}
		length, tail := w.Commit()
		s := segment.Committed{Config: clockCfg, Length: int64(length), TailCRC: tail}
		rd := segment.NewReader(nil)
		samples := int64(binary.LittleEndian.Uint32(stream[segment.BlockSize-8:]))
		_, pos, err := rd.Search(h, s, int64(ts), 0, samples)
		if err != nil {
			if !errors.Is(err, place.ErrCorrupt) {
				t.Fatalf("search fails with %v", err)
			}
			return
		}
		if pos < 0 || pos > samples {
			t.Fatalf("search gives position %d of %d samples", pos, samples)
		}
		got := make([]byte, 8*min(samples, 64))
		if err = rd.Stamps(
			h,
			s,
			got,
			0,
		); err != nil &&
			!errors.Is(err, place.ErrCorrupt) {
			t.Fatalf("stamps fail with %v", err)
		}
	})
}
