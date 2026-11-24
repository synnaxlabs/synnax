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
	"encoding/binary"
	"io"
	"math"
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/segment"
	xfs "github.com/synnaxlabs/x/io/fs"
	fstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// clockCfg is the configuration of a version 2 clock segment.
var clockCfg = segment.Config{
	Key:      5,
	DB:       7,
	ID:       9,
	DataType: 8,
	Clock:    5,
	Version:  segment.ProgressionVersion,
}

// stampBytes returns stamps as int64 values, little-endian.
func stampBytes(stamps []int64) []byte {
	b := make([]byte, 0, 8*len(stamps))
	for _, x := range stamps {
		b = binary.LittleEndian.AppendUint64(b, uint64(x))
	}
	return b
}

// encode writes stamps as a version 2 segment to a new file of fs, adding them in
// chunks of chunk stamps, and returns the file and its committed part. It returns false
// when the result takes more than limit bytes or the stamps do not rise.
func encode(
	fs xfs.FS,
	stamps []int64,
	chunk int,
	limit int64,
) (*segment.Handle, segment.Committed, bool, error) {
	budget, err := segment.NewBudget(4, 4, segment.GoExecutor)
	if err != nil {
		return nil, segment.Committed{}, false, err
	}
	h, err := budget.Create(fs, "9.seg")
	if err != nil {
		return nil, segment.Committed{}, false, err
	}
	w := segment.NewWriter(h, clockCfg)
	e := segment.NewEncoder(w, limit)
	b := stampBytes(stamps)
	add := func() error {
		for i := 0; i < len(b); i += 8 * chunk {
			if err := e.Add(b[i:min(i+8*chunk, len(b))]); err != nil {
				return err
			}
		}
		return nil
	}
	if err = add(); err != nil {
		return h, segment.Committed{}, false, err
	}
	fits, err := e.Rewind()
	if err != nil || !fits {
		return h, segment.Committed{}, false, err
	}
	if err = add(); err == nil {
		err = e.Close()
	}
	length, tail := w.Commit()
	return h, segment.Committed{Config: clockCfg, Length: int64(length), TailCRC: tail},
		true, err
}

// footer returns the entry count, the sample count, and the raw stamp count of the
// version 2 segment s in r.
func footer(r io.ReaderAt, s segment.Committed) (entries, samples, raw uint32) {
	GinkgoHelper()
	b := make([]byte, 20)
	Expect(segment.NewReader(nil).Read(r, s, b, s.Length-20)).To(Succeed())
	return binary.LittleEndian.Uint32(b[8:]), binary.LittleEndian.Uint32(b[12:]),
		binary.LittleEndian.Uint32(b[16:])
}

// regular returns n stamps from start with one step.
func regular(start, step int64, n int) []int64 {
	s := make([]int64, n)
	for i := range s {
		s[i] = start + int64(i)*step
	}
	return s
}

// driver returns the stamps of writes of per stamps at one step, with a gap of a varied
// length before each write.
func driver(writes, per int, step int64) []int64 {
	r := rand.New(rand.NewPCG(1, 2))
	s := make([]int64, 0, writes*per)
	x := int64(1_000_000_000)
	for range writes {
		x += step + 1 + r.Int64N(step)
		for range per {
			s = append(s, x)
			x += step
		}
	}
	return s
}

// irregular returns n stamps with random gaps.
func irregular(n int, seed uint64) []int64 {
	r := rand.New(rand.NewPCG(seed, 3))
	s := make([]int64, n)
	x := int64(5)
	for i := range s {
		x += 1 + r.Int64N(1_000)
		s[i] = x
	}
	return s
}

var _ = Describe("Progressions", func() {
	for fsName, makeFS := range fstestutil.FileSystems {
		Context(fsName, func() {
			var rd *segment.Reader
			BeforeEach(func() { rd = segment.NewReader(segment.NewCache(1 << 20)) })

			// mustEncode encodes stamps that fit.
			mustEncode := func(stamps []int64) (*segment.Handle, segment.Committed) {
				GinkgoHelper()
				h, s, fits, err := encode(makeFS(), stamps, 1000, math.MaxInt64)
				DeferCleanup(func() { Expect(h.Close()).To(Succeed()) })
				Expect(err).ToNot(HaveOccurred())
				Expect(fits).To(BeTrue())
				return h, s
			}

			DescribeTable(
				"Should read every stamp back and find each one",
				func(stamps []int64) {
					h, s := mustEncode(stamps)
					Expect(s.Length % segment.BlockSize).To(BeZero())
					_, samples, _ := footer(h, s)
					Expect(samples).To(BeEquivalentTo(len(stamps)))
					got := make([]byte, 8*len(stamps))
					Expect(rd.Stamps(h, s, got, 0)).To(Succeed())
					Expect(got).To(Equal(stampBytes(stamps)))
					for _, at := range [][2]int{{0, 1}, {3, 17}, {len(stamps) - 9, 9}} {
						got = make([]byte, 8*at[1])
						Expect(rd.Stamps(h, s, got, int64(at[0]))).To(Succeed())
						Expect(got).To(Equal(stampBytes(stamps[at[0] : at[0]+at[1]])))
					}
					for i, x := range stamps {
						pos, exact := MustSucceed2(rd.Search(h, s, x))
						Expect(pos).To(BeEquivalentTo(i))
						Expect(exact).To(BeTrue())
						if i > 0 && stamps[i-1] < x-1 {
							pos, exact = MustSucceed2(rd.Search(h, s, x-1))
							Expect(pos).To(BeEquivalentTo(i))
							Expect(exact).To(BeFalse())
						}
					}
					pos, exact := MustSucceed2(rd.Search(h, s, stamps[0]-1))
					Expect(pos).To(BeZero())
					Expect(exact).To(BeFalse())
					pos, exact = MustSucceed2(rd.Search(h, s, stamps[len(stamps)-1]+1))
					Expect(pos).To(BeEquivalentTo(len(stamps)))
					Expect(exact).To(BeFalse())
				},
				Entry("one step", regular(100, 40_000, 20_000)),
				Entry("writes of a Driver clock", driver(400, 125, 40_000)),
				Entry("random gaps", irregular(9_000, 1)),
				Entry("random gaps and runs", append(irregular(20, 2),
					regular(1_000_000, 3, 50)...)),
				Entry("entries in more than one table block", driver(9_000, 5, 1_000)),
			)

			It("Should find a time with the last block and one block of the table",
				func() {
					stamps := driver(9_000, 5, 1_000)
					h, s := mustEncode(stamps)
					r := &countingReader{ReaderAt: h}
					cold := segment.NewReader(nil)
					pos, exact := MustSucceed2(cold.Search(r, s, stamps[20_000]))
					Expect(pos).To(BeEquivalentTo(20_000))
					Expect(exact).To(BeTrue())
					Expect(r.reads.Load()).To(BeEquivalentTo(2),
						"the last block, with the footer and the fences, and one table block")
					MustSucceed2(cold.Search(r, s, stamps[24_000]))
					Expect(r.reads.Load()).To(BeEquivalentTo(3))
				})

			It("Should store a Driver clock in 16 bytes per write", func() {
				h, s := mustEncode(driver(10_000, 125, 40_000))
				entries, _, raw := footer(h, s)
				Expect(entries).To(BeEquivalentTo(10_000))
				Expect(raw).To(BeZero())
				Expect(s.Length).To(BeEquivalentTo(3 * segment.BlockSize))
			})

			DescribeTable(
				"Should split the stamps into entries at the format's limits",
				func(stamps []int64, entries, raw int) {
					h, s := mustEncode(stamps)
					gotEntries, _, gotRaw := footer(h, s)
					Expect(gotEntries).To(BeEquivalentTo(entries))
					Expect(gotRaw).To(BeEquivalentTo(raw))
					got := make([]byte, 8*len(stamps))
					Expect(rd.Stamps(h, s, got, 0)).To(Succeed())
					Expect(got).To(Equal(stampBytes(stamps)))
				},
				Entry("a run of 4 is raw", regular(10, 3, 4), 1, 4),
				Entry("a run of 5 is a progression", regular(10, 3, 5), 1, 0),
				Entry("a raw entry holds 8,192 stamps", irregular(8_192, 4), 1, 8_192),
				Entry(
					"a raw entry of 8,193 stamps splits",
					irregular(8_193, 4),
					2,
					8_193,
				),
				Entry("a step of 2^31 - 1", regular(0, math.MaxInt32, 5), 1, 0),
				Entry("a step of 2^31 is raw", regular(0, math.MaxInt32+1, 5), 1, 5),
				Entry(
					"a raw stamp, then a run",
					append([]int64{1}, regular(10, 2, 6)...),
					2,
					1,
				),
				Entry(
					"a run, then raw stamps",
					append(regular(10, 2, 6), 30, 33),
					2,
					2,
				),
			)

			It("Should not fit stamps that do not rise", func() {
				for _, stamps := range [][]int64{{1, 2, 2, 3}, {5, 4}} {
					h, _, fits, err := encode(makeFS(), stamps, 1, math.MaxInt64)
					Expect(err).ToNot(HaveOccurred())
					Expect(fits).To(BeFalse())
					Expect(h.Close()).To(Succeed())
				}
			})

			It("Should not fit a result past its limit", func() {
				stamps := irregular(20_000, 5)
				h, _, fits, err := encode(makeFS(), stamps, 1000, 2*segment.BlockSize)
				Expect(err).ToNot(HaveOccurred())
				Expect(fits).To(BeFalse())
				Expect(h.Close()).To(Succeed())
				h, _, fits, err = encode(makeFS(), stamps, 1000, 3*segment.BlockSize)
				Expect(err).ToNot(HaveOccurred())
				Expect(fits).To(BeTrue())
				Expect(h.Close()).To(Succeed())
			})

			It("Should fail a second pass that adds other stamps", func() {
				h := MustSucceed(
					MustSucceed(segment.NewBudget(4, 4, segment.GoExecutor)).
						Create(makeFS(), "9.seg"),
				)
				DeferCleanup(func() { Expect(h.Close()).To(Succeed()) })
				e := segment.NewEncoder(segment.NewWriter(h, clockCfg), math.MaxInt64)
				Expect(e.Add(stampBytes(regular(0, 5, 10)))).To(Succeed())
				Expect(e.Rewind()).To(BeTrue())
				Expect(e.Add(stampBytes(regular(0, 5, 9)))).To(Succeed())
				Expect(e.Close()).To(MatchError(ContainSubstring(
					"second pass of 9 stamps and 1 entries, not 10 and 1",
				)))
			})

			DescribeTable(
				"Should refuse a table byte that fails its check",
				func(at func(s segment.Committed) int64) {
					h, s := mustEncode(append(irregular(10, 6),
						regular(1_000_000, 7, 30)...))
					f := &flipReader{ReaderAt: h, at: s.FileOffset(at(s)), left: -1}
					got := make([]byte, 8*40)
					Expect(rd.Stamps(f, s, got, 0)).
						To(MatchError(place.ErrCorrupt))
					Expect(rd.Search(f, s, 1_000_000)).Error().
						To(MatchError(place.ErrCorrupt))
				},
				Entry("a raw stamp", func(segment.Committed) int64 { return 0 }),
				Entry("an entry", func(segment.Committed) int64 { return 80 + 17 }),
				Entry("a fence", func(segment.Committed) int64 { return 80 + 32 + 3 }),
				Entry(
					"the footer",
					func(s segment.Committed) int64 { return s.Length - 9 },
				),
			)

			It("Should refuse a footer whose counts do not fit the length", func() {
				h := MustSucceed(
					MustSucceed(segment.NewBudget(4, 4, segment.GoExecutor)).
						Create(makeFS(), "9.seg"),
				)
				DeferCleanup(func() { Expect(h.Close()).To(Succeed()) })
				w := segment.NewWriter(h, clockCfg)
				b := make([]byte, segment.BlockSize)
				binary.LittleEndian.PutUint32(b[len(b)-12:], 1<<30)
				binary.LittleEndian.PutUint32(b[len(b)-8:], 1<<30)
				Expect(w.Write(b)).To(Succeed())
				length, tail := w.Commit()
				s := segment.Committed{
					Config:  clockCfg,
					Length:  int64(length),
					TailCRC: tail,
				}
				_, _, err := rd.Search(h, s, 5)
				Expect(err).To(MatchError(place.ErrCorrupt))
				Expect(err).To(MatchError(ContainSubstring(
					"footer of 1073741824 entries",
				)))
			})
		})
	}
})

var _ = Describe("Pad", func() {
	var (
		fs xfs.FS
		h  *segment.Handle
	)
	BeforeEach(func() {
		fs = xfs.NewMem()
		h = MustSucceed(MustSucceed(segment.NewBudget(4, 4, segment.GoExecutor)).
			Create(fs, "3.seg"))
		DeferCleanup(func() { Expect(h.Close()).To(Succeed()) })
	})

	It("Should fill a fixed-density block with zeros and its trailer", func() {
		w := segment.NewWriter(h, cfg)
		Expect(w.Write(pattern(100))).To(Succeed())
		Expect(w.Pad()).To(Succeed())
		Expect(w.Len()).To(BeEquivalentTo(segment.BlockSize))
		Expect(w.Pad()).To(Succeed())
		Expect(w.Len()).To(BeEquivalentTo(segment.BlockSize))
		length, tail := w.Commit()
		got := make([]byte, segment.BlockSize)
		s := segment.Committed{Config: cfg, Length: int64(length), TailCRC: tail}
		Expect(segment.NewReader(nil).Read(h, s, got, 0)).To(Succeed())
		Expect(got[:100]).To(Equal(pattern(100)))
		Expect(got[100:]).To(Equal(make([]byte, segment.BlockSize-100)))
	})

	It("Should count no variable-density sample in its zeros", func() {
		w := segment.NewWriter(h, varCfg)
		first, _ := samples(10, 20)
		Expect(w.Write(first)).To(Succeed())
		Expect(w.Pad()).To(Succeed())
		next, _ := samples(segment.BlockSize - 4)
		Expect(w.Write(next)).To(Succeed())
		trailer := make([]byte, 8)
		at := varCfg.FileOffset(2*segment.BlockSize) - segment.VariableTrailer
		MustSucceed(h.ReadAt(trailer, at))
		Expect(binary.LittleEndian.Uint32(trailer)).To(BeEquivalentTo(2))
		Expect(binary.LittleEndian.Uint32(trailer[4:])).To(BeZero())
	})

	It("Should refuse to pad inside a variable-density sample", func() {
		w := segment.NewWriter(h, varCfg)
		stream, _ := samples(10)
		Expect(w.Copy(stream[:6])).To(Succeed())
		Expect(w.Pad()).To(MatchError(ContainSubstring("inside a sample")))
	})
})
