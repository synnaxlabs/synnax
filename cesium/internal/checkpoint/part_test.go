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
	// parted returns the live segments of the parts that h names.
	parted := func(h checkpoint.Header) []state.Segment {
		GinkgoHelper()
		s := &state.Snapshot{}
		MustSucceed(checkpoint.LoadParts(fs, db, h)).Merge(s)
		var out []state.Segment
		for _, ch := range s.Channels {
			out = append(out, ch.Segments...)
		}
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
	})

	It("Should give back the whole state from a head and its parts", func() {
		whole := withSealed(sealed(1, 10), sealed(2, 20), sealed(3, 30))
		switchTo(2, whole)
		h, s := MustSucceed2(checkpoint.Read(fs, 2))
		x := MustSucceed(checkpoint.LoadParts(fs, db, h))
		x.Merge(s)
		Expect(s.Channels).To(Equal(whole.Channels))
	})

	It("Should set the bit of a dead part segment", func() {
		switchTo(2, withSealed(sealed(1, 10), sealed(2, 20), sealed(3, 30)))
		h := switchTo(3, withSealed(sealed(1, 10), sealed(2, 20)))
		Expect(ids(h)).To(Equal([]uint64{1}))
		Expect(h.Parts[0].Dead).To(Equal([]byte{0b100}))
		_, s := MustSucceed2(checkpoint.Read(fs, 3))
		x := MustSucceed(checkpoint.LoadParts(fs, db, h))
		x.Merge(s)
		Expect(s.Channels[0].Segments).To(HaveLen(3))
	})

	It("Should write a part again under a new identity when half of it is dead",
		func() {
			switchTo(2, withSealed(sealed(1, 10), sealed(2, 20), sealed(3, 30)))
			h := switchTo(3, withSealed(sealed(1, 10)))
			Expect(ids(h)).To(Equal([]uint64{2}))
			Expect(h.Parts[0].Dead).To(BeNil())
			Expect(partedIDs(h)).To(Equal([]uint64{1}))
		})

	It("Should write a part again under a new identity when a segment in it is cut",
		func() {
			switchTo(2, withSealed(sealed(1, 10), sealed(2, 20), sealed(3, 30)))
			cut := sealed(2, 20)
			cut.Pointers = nil
			h := switchTo(3, withSealed(sealed(1, 10), cut, sealed(3, 30)))
			Expect(ids(h)).To(Equal([]uint64{2}))
			Expect(parted(h)[1].Pointers).To(BeEmpty())
		})

	It("Should drop a part whose segments are all dead", func() {
		switchTo(2, withSealed(sealed(1, 10)))
		h := switchTo(3, withSealed())
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
