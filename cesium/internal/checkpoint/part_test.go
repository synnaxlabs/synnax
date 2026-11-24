// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint_test

import (
	"cmp"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// partFile is the file name of part 1.
const partFile = "0000000000000001.part"

// sealed returns a sealed segment of clock 5 with one pointer at stamp.
func sealed(id uint64, stamp telem.TimeStamp) state.Segment {
	return state.Segment{
		ID: id, Clock: 5, Version: 1, DataType: 7, Sealed: true, Length: 8,
		Pointers: []state.Pointer{{
			First: stamp, Last: stamp, Alignment: telem.NewAlignment(uint32(id), 0),
			Length: 8, Samples: 1,
		}},
	}
}

// withSealed returns a snapshot of key 5 that holds segs and one open segment.
func withSealed(segs ...state.Segment) *state.Snapshot {
	open := state.Segment{ID: 100, Clock: 5, Version: 1, DataType: 7}
	return &state.Snapshot{
		Sequence: 9,
		Limits:   state.Limits{Segments: 101, Alignments: 101},
		Channels: []*state.Channel{{Key: 5, Segments: append(segs, open)}},
	}
}

// killed returns s with a kill of generation for each segment of key 5 in ids.
func killed(s *state.Snapshot, generation uint64, ids ...uint64) *state.Snapshot {
	for _, id := range ids {
		s.Kills = append(
			s.Kills,
			state.Kill{Key: 5, Segment: id, Generation: generation},
		)
	}
	return s
}

var _ = Describe("Parts", func() {
	var fs xfs.FS
	// switchTo writes the head of generation with a part size of one byte, so every
	// sealed segment that no part holds goes to a part, and returns the head.
	var c *checkpoint.Checkpointer
	BeforeEach(func() {
		fs = xfs.NewMem()
		c = MustSucceed(checkpoint.New(checkpoint.Config{
			FS:       fs,
			DB:       db,
			LogSize:  logSize,
			Prepared: func(uint64, error) {},
			Executor: goExec,
			PartSize: 1,
		}))
		start(c)
	})
	switchTo := func(generation uint64, s *state.Snapshot) checkpoint.Header {
		GinkgoHelper()
		c.Head(generation, s)
		Eventually(func() error { return readErr(fs, generation) }).Should(Succeed())
		h, _ := MustSucceed2(checkpoint.Read(fs, generation))
		return h
	}
	ids := func(h checkpoint.Header) []uint64 {
		out := make([]uint64, len(h.Parts))
		for i, p := range h.Parts {
			out[i] = p.ID
		}
		return out
	}
	// parted returns the live records of key 5 in the parts that h names, in rising
	// order of identity.
	parted := func(h checkpoint.Header) []state.Segment {
		GinkgoHelper()
		cache := checkpoint.NewCache(fs, db, 64<<20)
		var out []state.Segment
		for _, p := range h.Parts {
			r := MustSucceed(cache.Records(p, 5))
			for i, seg := range r.Segments {
				if p.Live(r.First + i) {
					out = append(out, seg)
				}
			}
		}
		slices.SortFunc(
			out,
			func(a, b state.Segment) int { return cmp.Compare(a.ID, b.ID) },
		)
		return out
	}
	partedIDs := func(h checkpoint.Header) []uint64 {
		GinkgoHelper()
		segs := parted(h)
		out := make([]uint64, 0, len(segs))
		for _, seg := range segs {
			out = append(out, seg.ID)
		}
		return out
	}

	It("Should move sealed segments past the part size to a part, and keep the rest "+
		"in the head", func() {
		h := switchTo(2, withSealed(sealed(1, 10), sealed(2, 20), sealed(3, 30)))
		Expect(ids(h)).To(Equal([]uint64{1}))
		Expect(h.Parts[0].First).To(Equal(telem.TimeStamp(10)))
		Expect(h.Parts[0].Last).To(Equal(telem.TimeStamp(30)))
		Expect(h.Limits.Parts).To(Equal(uint64(2)))
		Expect(partedIDs(h)).To(Equal([]uint64{1, 2, 3}))
		_, s := MustSucceed2(checkpoint.Read(fs, 2))
		Expect(s.Channels).To(HaveLen(1))
		Expect(s.Channels[0].Segments).To(HaveLen(1))
		Expect(s.Channels[0].Segments[0].ID).To(Equal(uint64(100)))
		Expect(*s.Channels[0].Parts).To(Equal(state.Parts{Oldest: 1, Newest: 1}))
	})

	It("Should give a key that only parts hold a channel with its part range", func() {
		s := withSealed(sealed(1, 10))
		s.Channels = append(
			s.Channels,
			&state.Channel{Key: 6, Segments: []state.Segment{
				{
					ID:       2,
					Clock:    6,
					Version:  1,
					DataType: 7,
					Sealed:   true,
					Length:   8,
					Pointers: []state.Pointer{
						{First: 5, Last: 5, Length: 8, Samples: 1},
					},
				},
			}},
		)
		h := switchTo(2, s)
		_, got := MustSucceed2(checkpoint.Read(fs, 2))
		Expect(got.Channels).To(HaveLen(2))
		Expect(got.Channels[1].Key).To(Equal(uint32(6)))
		Expect(got.Channels[1].Segments).To(BeEmpty())
		Expect(*got.Channels[1].Parts).To(Equal(state.Parts{
			Oldest: h.Parts[0].ID,
			Newest: h.Parts[0].ID,
		}))
	})

	It("Should give back the whole state from a head and the live records of its "+
		"parts", func() {
		whole := withSealed(sealed(1, 10), sealed(2, 20), sealed(3, 30))
		switchTo(2, whole)
		h, s := MustSucceed2(checkpoint.Read(fs, 2))
		Expect(s.Channels).To(HaveLen(1))
		got := append(parted(h), s.Channels[0].Segments...)
		Expect(got).To(Equal(whole.Channels[0].Segments))
	})

	It("Should set the bit of a killed part segment", func() {
		switchTo(2, withSealed(sealed(1, 10), sealed(2, 20), sealed(3, 30)))
		h := switchTo(3, killed(withSealed(sealed(1, 10), sealed(2, 20)), 2, 3))
		Expect(ids(h)).To(Equal([]uint64{1}))
		Expect(h.Parts[0].Dead).To(Equal([]byte{0b100}))
		Expect(partedIDs(h)).To(Equal([]uint64{1, 2}))
	})

	It("Should write a part again under a new identity when half of it is dead",
		func() {
			switchTo(2, withSealed(sealed(1, 10), sealed(2, 20), sealed(3, 30)))
			h := switchTo(3, killed(withSealed(sealed(1, 10)), 2, 2, 3))
			Expect(ids(h)).To(Equal([]uint64{2}))
			Expect(h.Parts[0].Dead).To(BeNil())
			Expect(partedIDs(h)).To(Equal([]uint64{1}))
		})

	It("Should keep the record of a cut segment dead in its old part and live in its "+
		"new one across two switches", func() {
		switchTo(2, withSealed(sealed(1, 10), sealed(2, 20), sealed(3, 30)))
		cut := sealed(2, 20)
		cut.Pointers = nil
		h := switchTo(3, killed(withSealed(sealed(1, 10), cut, sealed(3, 30)), 2, 2))
		Expect(ids(h)).To(Equal([]uint64{1, 2}))
		Expect(h.Parts[0].Dead).To(Equal([]byte{0b010}))
		Expect(partedIDs(h)).To(Equal([]uint64{1, 2, 3}))
		Expect(parted(h)[1].Pointers).To(BeEmpty())
		// The kill of generation 2 reaches part 1 alone, not part 2 of generation 3.
		h = switchTo(4, killed(withSealed(sealed(1, 10), cut, sealed(3, 30)), 2, 2))
		Expect(ids(h)).To(Equal([]uint64{1, 2}))
		Expect(h.Parts[1].Dead).To(BeNil())
		Expect(parted(h)[1].Pointers).To(BeEmpty())
		// A kill of generation 3 reaches part 2 too, whose only record it kills.
		h = switchTo(5, killed(withSealed(sealed(1, 10), sealed(3, 30)), 3, 2))
		Expect(ids(h)).To(Equal([]uint64{1}))
		Expect(partedIDs(h)).To(Equal([]uint64{1, 3}))
	})

	It("Should take the part ranges of a head from its switch alone, not from the "+
		"channels of its state", func() {
		switchTo(2, withSealed(sealed(1, 10)))
		// The state still carries the range of head 2, whose only record is dead.
		s := killed(withSealed(), 2, 1)
		s.Channels[0].Parts = &state.Parts{Oldest: 1, Newest: 1}
		h := switchTo(3, s)
		Expect(h.Parts).To(BeEmpty())
		_, got := MustSucceed2(checkpoint.Read(fs, 3))
		Expect(got.Channels[0].Parts).To(BeNil())
	})

	It("Should drop a part whose key a kill of every segment names", func() {
		switchTo(2, withSealed(sealed(1, 10)))
		s := withSealed()
		s.Kills = []state.Kill{{Key: 5, All: true, Generation: 2}}
		h := switchTo(3, s)
		Expect(h.Parts).To(BeEmpty())
	})

	It("Should refuse to load a part that the head names and the root lacks", func() {
		h := switchTo(2, withSealed(sealed(1, 10)))
		Expect(fs.Remove(partFile)).To(Succeed())
		Expect(checkpoint.LoadParts(fs, db, h)).Error().To(And(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring("part 1 that head 2 names is missing")),
		))
	})

	It("Should refuse a part that is not the one that the head names", func() {
		h := switchTo(2, withSealed(sealed(1, 10)))
		h.Parts[0].CRC++
		Expect(checkpoint.LoadParts(fs, db, h)).Error().To(And(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring("not the part that its head names")),
		))
	})
})

var _ = Describe("Part determinism", func() {
	It("Should write the same part bytes for the same state", func() {
		write := func() []byte {
			fs := xfs.NewMem()
			c := MustSucceed(checkpoint.New(checkpoint.Config{
				FS:       fs,
				DB:       db,
				LogSize:  logSize,
				Prepared: func(uint64, error) {},
				Executor: goExec,
				PartSize: 1,
			}))
			start(c)
			c.Head(2, withSealed(sealed(1, 10), sealed(2, 20)))
			Eventually(func() error { return readErr(fs, 2) }).Should(Succeed())
			return fileBytes(fs, partFile)
		}
		Expect(write()).To(Equal(write()))
	})
})

var _ = Describe("Part hand-off", func() {
	It("Should hand each committed head's parts and the tags of the records it wrote "+
		"to the Parted hook", func() {
		type call struct {
			generation uint64
			parts      []checkpoint.Part
			tags       []state.Tag
			ranges     []state.Range
		}
		calls := make(chan call, 4)
		fs := xfs.NewMem()
		c := MustSucceed(checkpoint.New(checkpoint.Config{
			FS:       fs,
			DB:       db,
			LogSize:  logSize,
			Prepared: func(uint64, error) {},
			Executor: goExec,
			PartSize: 1,
			Parted: func(
				generation uint64,
				parts []checkpoint.Part,
				tags []state.Tag,
				ranges []state.Range,
			) {
				calls <- call{generation, parts, tags, ranges}
			},
		}))
		start(c)
		c.Head(2, withSealed(sealed(1, 10), sealed(2, 20), sealed(3, 30)))
		var first call
		Eventually(calls).Should(Receive(&first))
		Expect(first.generation).To(Equal(uint64(2)))
		Expect(first.parts).To(HaveLen(1))
		Expect(first.parts[0].ID).To(Equal(uint64(1)))
		Expect(first.ranges).To(Equal([]state.Range{{Key: 5, Oldest: 1, Newest: 1}}))
		Expect(first.tags).To(Equal([]state.Tag{
			{Part: 1, Segment: sealed(1, 10)},
			{Part: 1, Segment: sealed(2, 20)},
			{Part: 1, Segment: sealed(3, 30)},
		}))
		c.Head(3, killed(withSealed(sealed(1, 10)), 2, 2, 3))
		var second call
		Eventually(calls).Should(Receive(&second))
		Expect(second.generation).To(Equal(uint64(3)))
		Expect(second.ranges).To(Equal([]state.Range{{Key: 5, Oldest: 2, Newest: 2}}))
		Expect(second.parts).To(HaveLen(1))
		Expect(second.parts[0].ID).To(Equal(uint64(2)))
		Expect(second.tags).To(Equal([]state.Tag{{Part: 2, Segment: sealed(1, 10)}}))
	})
})
