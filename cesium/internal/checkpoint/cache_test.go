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
	"context"
	"os"
	"slices"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// wide returns a snapshot of keys keys, each with perKey sealed segments, so that its
// part spans more than one chunk.
func wide(keys, perKey int) *state.Snapshot {
	s := &state.Snapshot{}
	id := uint64(0)
	for k := range keys {
		ch := &state.Channel{Key: uint32(k + 1)}
		for range perKey {
			id++
			seg := sealed(id, telem.TimeStamp(id))
			seg.Clock = ch.Key
			ch.Segments = append(ch.Segments, seg)
		}
		s.Channels = append(s.Channels, ch)
	}
	s.Limits = state.Limits{Segments: id + 1, Alignments: id + 1}
	return s
}

// flip changes one byte at off of the file name in fs.
func flip(fs xfs.FS, name string, off int64) {
	GinkgoHelper()
	f := MustSucceed(fs.Open(name, os.O_RDWR))
	b := make([]byte, 1)
	MustSucceed(f.ReadAt(b, off))
	b[0] ^= 0xFF
	MustSucceed(f.WriteAt(b, off))
	Expect(f.Close()).To(Succeed())
}

var _ = Describe("Part cache", func() {
	var (
		fs xfs.FS
		c  *checkpoint.Checkpointer
	)
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

	It("Should read the records of each key of a part that spans chunks", func() {
		s := wide(300, 10)
		h := switchTo(2, s)
		cache := checkpoint.NewCache(fs, db, 64<<20)
		for i, ch := range s.Channels {
			r := MustSucceed(cache.Records(h.Parts[0], ch.Key))
			Expect(r.First).To(Equal(10 * i))
			Expect(r.Segments).To(Equal(ch.Segments))
		}
	})

	It("Should return no records for a key that the part does not hold", func() {
		h := switchTo(2, wide(3, 2))
		cache := checkpoint.NewCache(fs, db, 64<<20)
		Expect(MustSucceed(cache.Records(h.Parts[0], 99)).Segments).To(BeEmpty())
	})

	It("Should find, in order of identity, every record that reaches a range",
		func() {
			// Backfills leave stamps out of the order of identity, and segment 2, the
			// lowest, spans two pointers far apart.
			stamps := []telem.TimeStamp{50, 10, 90, 30, 70, 20, 60}
			ch := &state.Channel{Key: 5}
			for i, stamp := range stamps {
				ch.Segments = append(ch.Segments, sealed(uint64(i+1), stamp))
			}
			wide := &ch.Segments[1]
			wide.Pointers = append(wide.Pointers, state.Pointer{
				First: 200, Last: 210, Alignment: telem.NewAlignment(2, 1),
				Offset: 8, Length: 8, Samples: 1,
			})
			wide.Length = 16
			s := &state.Snapshot{
				Channels: []*state.Channel{ch},
				Limits:   state.Limits{Segments: 8, Alignments: 8},
			}
			h := switchTo(2, s)
			r := MustSucceed(checkpoint.NewCache(fs, db, 64<<20).Records(h.Parts[0], 5))
			for first := telem.TimeStamp(0); first <= 220; first += 5 {
				for _, last := range []telem.TimeStamp{first, first + 15, first + 100} {
					var want []int
					for i, seg := range r.Segments {
						for _, ptr := range seg.Pointers {
							if ptr.First <= last && first <= ptr.Last {
								want = append(want, i)
								break
							}
						}
					}
					got := r.Reaching(first, last, nil)
					Expect(got).To(ContainElements(want), "range %d to %d", first, last)
					Expect(
						slices.IsSorted(got),
					).To(BeTrue(), "range %d to %d", first, last)
				}
			}
		})

	It("Should give the same records from the cache after the file is gone", func() {
		s := wide(3, 2)
		h := switchTo(2, s)
		cache := checkpoint.NewCache(fs, db, 64<<20)
		first := MustSucceed(cache.Records(h.Parts[0], 2))
		Expect(fs.Remove(partFile)).To(Succeed())
		Expect(MustSucceed(cache.Records(h.Parts[0], 2))).To(Equal(first))
	})

	It("Should drop the least recently used records past its budget", func() {
		h := switchTo(2, wide(3, 2))
		cache := checkpoint.NewCache(fs, db, 1)
		MustSucceed(cache.Records(h.Parts[0], 1))
		MustSucceed(cache.Records(h.Parts[0], 2))
		Expect(fs.Remove(partFile)).To(Succeed())
		Expect(cache.Records(h.Parts[0], 1)).Error().To(MatchError(os.ErrNotExist))
	})

	It("Should refuse the records in a chunk that fails its check twice", func() {
		s := wide(300, 10)
		h := switchTo(2, s)
		// Chunk 2 holds pointer records of later keys, and none of key 1.
		flip(fs, partFile, checkpoint.BodyStart+2*checkpoint.ChunkSize+100)
		cache := checkpoint.NewCache(fs, db, 64<<20)
		Expect(MustSucceed(cache.Records(h.Parts[0], 1)).Segments).To(HaveLen(10))
		var failed error
		for _, ch := range s.Channels {
			if _, err := cache.Records(h.Parts[0], ch.Key); err != nil {
				failed = err
				break
			}
		}
		Expect(failed).To(And(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring("of part 1 fails its check")),
		))
	})

	It("Should serve concurrent readers while it drops entries", func() {
		s := wide(50, 4)
		h := switchTo(2, s)
		cache := checkpoint.NewCache(fs, db, 4<<10)
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				defer GinkgoRecover()
				for range 20 {
					for _, ch := range s.Channels {
						r := MustSucceed(cache.Records(h.Parts[0], ch.Key))
						Expect(r.Segments).To(Equal(ch.Segments))
					}
				}
			})
		}
		wg.Wait()
	})

	It("Should refuse a part that is not the one that the head names", func() {
		h := switchTo(2, wide(3, 2))
		p := h.Parts[0]
		p.CRC++
		cache := checkpoint.NewCache(fs, db, 64<<20)
		Expect(cache.Records(p, 1)).Error().To(And(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring("not the part that its head names")),
		))
	})

	It("Should keep a part with its dead marks when a read for its rewrite fails",
		func() {
			// d is a checkpointer that the spec closes, since Close returns the error
			// of
			// the last head.
			d := MustSucceed(checkpoint.New(checkpoint.Config{
				FS:       fs,
				DB:       db,
				LogSize:  logSize,
				Prepared: func(uint64, error) {},
				Executor: goExec,
				PartSize: 1,
			}))
			ran := make(chan error, 1)
			go func() { ran <- d.Run(context.Background()) }()
			d.Head(2, withSealed(sealed(1, 10), sealed(2, 20), sealed(3, 30)))
			Eventually(func() error { return readErr(fs, 2) }).Should(Succeed())
			flip(fs, partFile, checkpoint.BodyStart+10)
			d.Head(3, killed(withSealed(sealed(1, 10)), 2, 2, 3))
			Eventually(func() error { return readErr(fs, 3) }).Should(Succeed())
			Expect(d.Close()).To(And(
				MatchError(place.ErrCorrupt),
				MatchError(ContainSubstring("of part 1 fails its check")),
			))
			Eventually(ran).Should(Receive(BeNil()))
			h, _ := MustSucceed2(checkpoint.Read(fs, 3))
			Expect(h.Parts).To(HaveLen(1))
			Expect(h.Parts[0].ID).To(Equal(uint64(1)))
			Expect(h.Parts[0].Dead).To(Equal([]byte{0b110}))
		})
})
