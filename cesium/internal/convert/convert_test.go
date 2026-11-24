// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package convert_test

import (
	"encoding/binary"
	jsonv2 "encoding/json/v2"
	"fmt"
	"os"
	"path"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/convert"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/encoding/json"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var codec = json.NewCodec(jsonv2.MatchCaseInsensitiveNames(true))

// offsetFS records the offset of each read of the file name.
type offsetFS struct {
	xfs.FS
	name string
	offs *[]int64
}

func (f offsetFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || name != f.name {
		return file, err
	}
	return offsetFile{File: file, offs: f.offs}, nil
}

type offsetFile struct {
	xfs.File
	offs *[]int64
}

func (f offsetFile) ReadAt(p []byte, off int64) (int, error) {
	*f.offs = append(*f.offs, off)
	return f.File.ReadAt(p, off)
}

// old is a pointer of an old index.domain file.
type old struct {
	start, end   telem.TimeStamp
	file         uint16
	offset, size uint32
}

// oldRoot writes the files of the old layout.
type oldRoot struct{ fs xfs.FS }

func (r oldRoot) write(name string, b []byte) {
	GinkgoHelper()
	sub := MustSucceed(r.fs.Sub(path.Dir(name)))
	f := MustSucceed(sub.Open(path.Base(name), os.O_CREATE|os.O_RDWR|os.O_TRUNC))
	MustSucceed(f.WriteAt(b, 0))
	Expect(f.Close()).To(Succeed())
}

func (r oldRoot) channel(ch channel.Channel) {
	GinkgoHelper()
	ch.Version = channel.Version3
	r.write(fmt.Sprintf("%d/meta.json", ch.Key), MustSucceed(codec.Encode(ch)))
}

func (r oldRoot) pointers(key uint32, ptrs ...old) {
	GinkgoHelper()
	b := make([]byte, 26*len(ptrs))
	for i, p := range ptrs {
		e := b[26*i:]
		binary.LittleEndian.PutUint64(e, uint64(p.start))
		binary.LittleEndian.PutUint64(e[8:], uint64(p.end))
		binary.LittleEndian.PutUint16(e[16:], p.file)
		binary.LittleEndian.PutUint32(e[18:], p.offset)
		binary.LittleEndian.PutUint32(e[22:], p.size)
	}
	r.write(fmt.Sprintf("%d/index.domain", key), b)
}

func (r oldRoot) file(key uint32, n uint16, b []byte) {
	GinkgoHelper()
	r.write(fmt.Sprintf("%d/%d.domain", key, n), b)
}

func stamps(ts ...telem.TimeStamp) []byte {
	return telem.NewSeriesV(ts...).Data
}

// strs returns ss as variable-type samples: a uint32 length, then the bytes.
func strs(ss ...string) []byte {
	var b []byte
	for _, s := range ss {
		b = binary.LittleEndian.AppendUint32(b, uint32(len(s)))
		b = append(b, s...)
	}
	return b
}

func idx(key uint32) channel.Channel {
	return channel.Channel{
		Key:      key,
		Name:     "idx",
		DataType: telem.TimestampT,
		IsIndex:  true,
	}
}

// data returns data key 2 of type dt, timed by index key 1.
func data(dt telem.DataType) channel.Channel {
	return channel.Channel{Key: 2, Name: "data", DataType: dt, Index: 1}
}

func ptr(
	first, last telem.TimeStamp,
	num, pos, offset, length, samples uint32,
) state.Pointer {
	return state.Pointer{
		First:     first,
		Last:      last,
		Alignment: telem.NewAlignment(num, pos),
		Offset:    offset,
		Length:    length,
		Samples:   samples,
	}
}

var _ = Describe("Read", func() {
	var (
		fs   xfs.FS
		root oldRoot
	)
	BeforeEach(func() {
		fs = xfs.NewMem()
		root = oldRoot{fs: fs}
	})
	var cfg func() convert.Config
	BeforeEach(func() {
		cfg = func() convert.Config { return convert.Config{FS: fs, Codec: codec} }
	})
	read := func() (*state.Snapshot, []channel.Channel) {
		GinkgoHelper()
		p := MustSucceed(convert.Read(cfg()))
		return p.Snapshot, p.Channels
	}

	It(
		"Should give each index pointer one alignment number, keys rising, then time",
		func() {
			root.channel(idx(20))
			root.file(20, 1, stamps(100, 110, 300, 310))
			root.pointers(20, old{300, 311, 1, 16, 16}, old{100, 111, 1, 0, 16})
			root.channel(idx(10))
			root.file(10, 1, stamps(5, 6, 7))
			root.pointers(10, old{5, 8, 1, 0, 24})
			s, chs := read()
			Expect(chs).To(HaveLen(2))
			Expect(s.Limits).To(Equal(state.Limits{Segments: 3, Alignments: 4}))
			Expect(s.Channels).To(Equal([]*state.Channel{
				{Key: 10, Segments: []state.Segment{
					{
						ID:       1,
						Clock:    10,
						File:     1,
						DataType: uint8(telem.DataTypeCodeTimestamp),
						Sealed:   true,
						Length:   24,
						Pointers: []state.Pointer{ptr(5, 7, 1, 0, 0, 24, 3)},
					},
				}},
				{Key: 20, Segments: []state.Segment{
					{
						ID:       2,
						Clock:    20,
						File:     1,
						DataType: uint8(telem.DataTypeCodeTimestamp),
						Sealed:   true,
						Length:   32,
						Pointers: []state.Pointer{
							ptr(100, 110, 2, 0, 0, 16, 2),
							ptr(300, 310, 3, 0, 16, 16, 2),
						},
					},
				}},
			}))
		},
	)

	It(
		"Should take the stamps of an index pointer from its file, not its bounds",
		func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(10, 20, 30))
			root.pointers(1, old{5, 1000, 1, 0, 24})
			s, _ := read()
			Expect(s.Channels[0].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(10, 30, 1, 0, 0, 24, 3)},
			))
		},
	)

	It("Should read the stamps of an old index file in file-offset order", func() {
		root.channel(idx(1))
		root.file(1, 1, stamps(30, 40, 10, 20))
		root.pointers(1, old{30, 41, 1, 0, 16}, old{10, 21, 1, 16, 16})
		var offs []int64
		p := MustSucceed(convert.Read(convert.Config{
			FS:    offsetFS{FS: fs, name: "1/1.domain", offs: &offs},
			Codec: codec,
		}))
		Expect(offs).To(Equal([]int64{0, 0, 8, 8, 16, 16, 24, 24}))
		Expect(p.Snapshot.Channels[0].Segments[0].Pointers).To(Equal([]state.Pointer{
			ptr(30, 40, 2, 0, 0, 16, 2),
			ptr(10, 20, 1, 0, 16, 16, 2),
		}))
	})

	It(
		"Should split a data pointer at the start of each index pointer it crosses",
		func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 3, 4, 5))
			root.file(1, 2, stamps(6, 7, 8))
			// The second index pointer starts at the end of the first, as a rollover
			// does.
			root.pointers(1, old{1, 6, 1, 0, 40}, old{6, 9, 2, 0, 24})
			root.channel(data(telem.Float64T))
			root.file(2, 1, telem.NewSeriesV[float64](1, 2, 3, 4, 5, 6, 7, 8).Data)
			root.pointers(2, old{1, 9, 1, 0, 64})
			s, _ := read()
			Expect(s.Channels[1].Segments).To(Equal([]state.Segment{{
				ID: 3, Clock: 1, File: 1, DataType: uint8(telem.DataTypeCodeFloat64),
				Sealed: true, Length: 64,
				Pointers: []state.Pointer{
					ptr(1, 5, 1, 0, 0, 40, 5),
					ptr(6, 8, 2, 0, 40, 24, 3),
				},
			}}))
		},
	)

	It(
		"Should find the first stamp of a data pointer that starts inside an index pointer",
		func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(10, 20, 30, 40, 50, 60))
			root.pointers(1, old{10, 61, 1, 0, 48})
			root.channel(data(telem.Int32T))
			root.file(2, 1, telem.NewSeriesV[int32](3, 4, 5).Data)
			// A rollover starts the pointer between two stamps.
			root.pointers(2, old{21, 51, 1, 0, 12})
			s, _ := read()
			Expect(s.Channels[1].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(30, 50, 1, 2, 0, 12, 3)},
			))
		},
	)

	It(
		"Should split a variable-type pointer that crosses an index start on whole samples",
		func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 3))
			root.file(1, 2, stamps(4, 5))
			root.pointers(1, old{1, 4, 1, 0, 24}, old{4, 6, 2, 0, 16})
			root.channel(data(telem.StringT))
			b := strs("a", "bcdef", "", "ghi", "jk")
			root.file(2, 1, b)
			root.pointers(2, old{1, 6, 1, 0, uint32(len(b))})
			s, _ := read()
			head := uint32(len(strs("a", "bcdef", "")))
			Expect(s.Channels[1].Segments[0].Pointers).To(Equal([]state.Pointer{
				ptr(1, 3, 1, 0, 0, head, 3),
				ptr(4, 5, 2, 0, head, uint32(len(b))-head, 2),
			}))
		},
	)

	It("Should end a variable-type pointer at its end inside an index pointer", func() {
		root.channel(idx(1))
		root.file(1, 1, stamps(1, 2, 3, 4, 5))
		root.pointers(1, old{1, 6, 1, 0, 40})
		root.channel(data(telem.JSONT))
		b := strs(`{"a":1}`, `{"b":2}`)
		root.file(2, 1, b)
		root.pointers(2, old{2, 4, 1, 0, uint32(len(b))})
		s, _ := read()
		Expect(s.Channels[1].Segments[0].Pointers).To(Equal(
			[]state.Pointer{ptr(2, 3, 1, 1, 0, uint32(len(b)), 2)},
		))
	})

	It(
		"Should make one segment for each old file, with pointers in file order",
		func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 5, 6))
			root.file(1, 2, stamps(3, 4))
			root.pointers(
				1,
				old{1, 3, 1, 0, 16},
				old{3, 5, 2, 0, 16},
				old{5, 7, 1, 16, 16},
			)
			s, _ := read()
			segs := s.Channels[0].Segments
			Expect(segs).To(HaveLen(2))
			Expect(segs[0].File).To(BeEquivalentTo(1))
			Expect(segs[0].Length).To(BeEquivalentTo(32))
			Expect(segs[0].Pointers).To(Equal([]state.Pointer{
				ptr(1, 2, 1, 0, 0, 16, 2),
				ptr(5, 6, 3, 0, 16, 16, 2),
			}))
			Expect(segs[1].File).To(BeEquivalentTo(2))
			Expect(
				segs[1].Pointers,
			).To(Equal([]state.Pointer{ptr(3, 4, 2, 0, 0, 16, 2)}))
		},
	)

	It(
		"Should skip a virtual channel, and a directory with no meta.json or no key",
		func() {
			root.channel(idx(1))
			root.channel(channel.Channel{
				Key:      3,
				Name:     "virtual",
				DataType: telem.Float32T,
				Virtual:  true,
			})
			root.write("4/1.domain", []byte{1})
			root.write("5-DELETE-77/meta.json", []byte("{}"))
			root.write("notes/meta.json", []byte("{}"))
			s, chs := read()
			Expect(chs).To(HaveLen(1))
			Expect(chs[0].Key).To(BeEquivalentTo(1))
			Expect(s.Channels).To(BeEmpty())
			Expect(s.Limits).To(Equal(state.Limits{Segments: 1, Alignments: 1}))
		},
	)

	It("Should carry the delete intents in rising order", func() {
		p := MustSucceed(convert.Read(convert.Config{
			FS:      fs,
			Codec:   codec,
			Intents: []uint32{9, 2, 9, 4},
		}))
		Expect(p.Snapshot.Intents).To(Equal([]uint32{2, 4, 9}))
	})

	Describe("Files of an old compaction", func() {
		It("Should read a missing old file from the file that an old compaction "+
			"renamed it to", func() {
			root.channel(idx(1))
			root.write("1/1.domain_temp", stamps(1, 2, 3))
			root.pointers(1, old{1, 4, 1, 0, 24})
			s, _ := read()
			Expect(s.Channels[0].Segments[0]).To(SatisfyAll(
				HaveField("File", BeEquivalentTo(1)),
				HaveField(
					"Pointers",
					Equal([]state.Pointer{ptr(1, 3, 1, 0, 0, 24, 3)}),
				),
			))
		})

		It("Should read an old file that exists, and not the files of an old "+
			"compaction", func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 3))
			root.write("1/1.domain_temp", stamps(7, 8, 9))
			root.write("1/1.domain_gc", stamps(4, 5, 6))
			root.pointers(1, old{1, 4, 1, 0, 24})
			s, _ := read()
			Expect(s.Channels[0].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 3, 1, 0, 0, 24, 3)},
			))
		})

		It("Should cut a pointer past the end of the file that an old compaction "+
			"renamed its file to", func() {
			root.channel(idx(1))
			root.write("1/1.domain_temp", stamps(1, 2))
			root.pointers(1, old{1, 4, 1, 0, 24})
			s, _ := read()
			Expect(s.Channels[0].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 2, 1, 0, 0, 16, 2)},
			))
		})
	})

	Describe("Cuts", func() {
		var logs *observer.ObservedLogs
		BeforeEach(func() {
			var ins alamos.Instrumentation
			ins, logs = ObservedInstrumentation(zapcore.WarnLevel)
			cfg = func() convert.Config {
				return convert.Config{Instrumentation: ins, FS: fs, Codec: codec}
			}
		})
		// warned returns the fields of each warning with the message msg.
		warned := func(msg string) []map[string]any {
			entries := logs.FilterMessage(msg).All()
			fields := make([]map[string]any, 0, len(entries))
			for _, e := range entries {
				fields = append(fields, e.ContextMap())
			}
			return fields
		}
		const (
			pastFile = "cut a pointer that ends past its file to the whole samples " +
				"inside it"
			noStamps = "cut the samples of a data pointer that its index has no " +
				"stamps for"
			indexCut = "cut a pointer of an index after its last stamp inside its " +
				"time range"
			dropped = "dropped a pointer of an index, whose first stamp is outside " +
				"its time range"
		)

		It("Should cut a pointer that ends past its file to its whole stamps", func() {
			root.channel(idx(1))
			root.file(1, 1, append(stamps(1, 2), 3, 0, 0))
			root.pointers(1, old{1, 4, 1, 0, 24})
			s, _ := read()
			Expect(s.Channels[0].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 2, 1, 0, 0, 16, 2)},
			))
			Expect(warned(pastFile)).To(Equal([]map[string]any{{
				"key":    uint32(1),
				"file":   "1/1.domain",
				"end":    int64(24),
				"length": int64(19),
				"kept":   int64(16),
			}}))
		})

		It("Should drop a pointer to a file that does not exist", func() {
			root.channel(idx(1))
			root.pointers(1, old{1, 4, 2, 0, 24})
			s, chs := read()
			Expect(chs).To(HaveLen(1))
			Expect(s.Channels).To(BeEmpty())
			Expect(warned(pastFile)).To(ConsistOf(HaveKeyWithValue("kept", int64(0))))
		})

		It("Should cut a variable-type pointer past its file at a whole sample, and "+
			"time each sample it keeps", func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 3))
			root.pointers(1, old{1, 4, 1, 0, 24})
			root.channel(data(telem.StringT))
			whole := strs("ab", "cde")
			root.file(2, 1, append(slices.Clone(whole), strs("f")[:4]...))
			root.pointers(2, old{1, 4, 1, 0, uint32(len(strs("ab", "cde", "f")))})
			s, _ := read()
			Expect(s.Channels[1].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 2, 1, 0, 0, uint32(len(whole)), 2)},
			))
			Expect(warned(pastFile)).To(ConsistOf(
				HaveKeyWithValue("kept", int64(len(whole))),
			))
		})

		It("Should cut data samples that have no stamps", func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2))
			root.pointers(1, old{1, 3, 1, 0, 16})
			root.channel(data(telem.Int64T))
			root.file(2, 1, telem.NewSeriesV[int64](1, 2, 3).Data)
			root.pointers(2, old{1, 4, 1, 0, 24})
			s, _ := read()
			Expect(s.Channels[1].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 2, 1, 0, 0, 16, 2)},
			))
			Expect(s.Channels[1].Segments[0].Length).To(BeEquivalentTo(16))
			Expect(warned(noStamps)).To(Equal([]map[string]any{{
				"key":    uint32(2),
				"file":   "2/1.domain",
				"offset": uint32(16),
				"bytes":  uint32(8),
				"index":  uint32(1),
			}}))
		})

		It("Should cut the data samples past the last stamp before the end of their "+
			"pointer", func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 3, 4, 5))
			root.pointers(1, old{1, 6, 1, 0, 40})
			root.channel(data(telem.Int64T))
			root.file(2, 1, telem.NewSeriesV[int64](1, 2, 3, 4, 5).Data)
			root.pointers(2, old{1, 4, 1, 0, 40})
			s, _ := read()
			Expect(s.Channels[1].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 3, 1, 0, 0, 24, 3)},
			))
			Expect(warned(noStamps)).To(ConsistOf(SatisfyAll(
				HaveKeyWithValue("offset", uint32(24)),
				HaveKeyWithValue("bytes", uint32(16)),
			)))
		})

		It("Should walk a variable-type pointer that ends past the newest stamp of "+
			"its index, and cut the samples with no stamps", func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2))
			root.pointers(1, old{1, 3, 1, 0, 16})
			root.channel(data(telem.StringT))
			b := strs("a", "bc", "def")
			root.file(2, 1, b)
			root.pointers(2, old{1, 4, 1, 0, uint32(len(b))})
			s, _ := read()
			kept := uint32(len(strs("a", "bc")))
			Expect(s.Channels[1].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 2, 1, 0, 0, kept, 2)},
			))
			Expect(warned(noStamps)).To(ConsistOf(SatisfyAll(
				HaveKeyWithValue("offset", kept),
				HaveKeyWithValue("bytes", uint32(len(b))-kept),
			)))
		})

		It("Should cut the bytes of a variable-type pointer that hold no whole "+
			"sample", func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 3, 4))
			root.pointers(
				1,
				old{1, 2, 1, 0, 8},
				old{2, 3, 1, 8, 8},
				old{3, 5, 1, 16, 16},
			)
			root.channel(data(telem.StringT))
			b := append(strs("a"), 9, 9)
			root.file(2, 1, b)
			root.pointers(2, old{1, 5, 1, 0, uint32(len(b))})
			s, _ := read()
			Expect(s.Channels[1].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 1, 1, 0, 0, 5, 1)},
			))
			Expect(warned(noStamps)).To(ConsistOf(SatisfyAll(
				HaveKeyWithValue("offset", uint32(5)),
				HaveKeyWithValue("bytes", uint32(2)),
			)))
		})

		It("Should cut the stamps of an index pointer past its file before the data "+
			"it times", func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 3))
			root.pointers(1, old{1, 6, 1, 0, 40})
			root.channel(data(telem.Int64T))
			root.file(2, 1, telem.NewSeriesV[int64](1, 2, 3, 4, 5).Data)
			root.pointers(2, old{1, 6, 1, 0, 40})
			s, _ := read()
			Expect(s.Channels[0].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 3, 1, 0, 0, 24, 3)},
			))
			Expect(s.Channels[1].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 3, 1, 0, 0, 24, 3)},
			))
			Expect(warned(pastFile)).To(HaveLen(1))
			Expect(
				warned(noStamps),
			).To(ConsistOf(HaveKeyWithValue("offset", uint32(24))))
		})

		It("Should cut a pointer of an index after its last stamp inside its time "+
			"range", func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(10, 20, 30, 0, 0))
			root.pointers(1, old{10, 51, 1, 0, 40})
			s, _ := read()
			Expect(s.Channels[0].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(10, 30, 1, 0, 0, 24, 3)},
			))
			Expect(warned(indexCut)).To(Equal([]map[string]any{{
				"key":    uint32(1),
				"file":   "1/1.domain",
				"offset": uint32(0),
				"stamps": uint32(5),
				"kept":   uint32(3),
			}}))
		})

		It("Should drop a pointer of an index whose first stamp is outside its range, "+
			"and cut one whose last stamp is outside", func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 0, 0, 0, 0))
			root.pointers(1, old{1, 4, 1, 0, 24}, old{4, 7, 1, 24, 24})
			s, _ := read()
			Expect(s.Limits.Alignments).To(BeEquivalentTo(2))
			Expect(s.Channels[0].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 2, 1, 0, 0, 16, 2)},
			))
			Expect(
				warned(dropped),
			).To(ConsistOf(HaveKeyWithValue("offset", uint32(24))))
			Expect(warned(indexCut)).To(ConsistOf(HaveKeyWithValue("kept", uint32(2))))
		})

		It("Should cut an older index pointer whose last stamp is outside its range, "+
			"and stop the data that crosses it there", func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 0, 4, 5, 6))
			root.pointers(1, old{1, 4, 1, 0, 24}, old{4, 7, 1, 24, 24})
			root.channel(data(telem.Int64T))
			root.file(2, 1, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6).Data)
			root.pointers(2, old{1, 7, 1, 0, 48})
			s, _ := read()
			Expect(s.Channels[0].Segments[0].Pointers).To(Equal([]state.Pointer{
				ptr(1, 2, 1, 0, 0, 16, 2),
				ptr(4, 6, 2, 0, 24, 24, 3),
			}))
			Expect(s.Channels[1].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 2, 1, 0, 0, 16, 2)},
			))
			Expect(warned(noStamps)).To(ConsistOf(SatisfyAll(
				HaveKeyWithValue("offset", uint32(16)),
				HaveKeyWithValue("bytes", uint32(32)),
			)))
		})

		It("Should stop the data at a dropped index pointer, and never give it the "+
			"stamps of a later pointer", func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 3, 0, 0, 0, 7, 8, 9))
			root.pointers(
				1,
				old{1, 4, 1, 0, 24},
				old{4, 7, 1, 24, 24},
				old{7, 10, 1, 48, 24},
			)
			root.channel(data(telem.Int64T))
			root.file(2, 1, telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7, 8, 9).Data)
			root.pointers(2, old{1, 10, 1, 0, 72})
			s, _ := read()
			Expect(s.Limits.Alignments).To(BeEquivalentTo(3))
			Expect(s.Channels[0].Segments[0].Pointers).To(Equal([]state.Pointer{
				ptr(1, 3, 1, 0, 0, 24, 3),
				ptr(7, 9, 2, 0, 48, 24, 3),
			}))
			Expect(s.Channels[1].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 3, 1, 0, 0, 24, 3)},
			))
			Expect(
				warned(noStamps),
			).To(ConsistOf(HaveKeyWithValue("offset", uint32(24))))
		})

		It("Should cut a data pointer that starts in the range of a dropped index "+
			"pointer", func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 3, 0, 0, 0, 7, 8, 9))
			root.pointers(
				1,
				old{1, 4, 1, 0, 24},
				old{4, 7, 1, 24, 24},
				old{7, 10, 1, 48, 24},
			)
			root.channel(data(telem.Int64T))
			root.file(2, 1, telem.NewSeriesV[int64](5, 6, 7, 8, 9).Data)
			root.pointers(2, old{5, 10, 1, 0, 40})
			s, chs := read()
			Expect(chs).To(HaveLen(2))
			Expect(s.Channels).To(HaveLen(1))
			Expect(warned(noStamps)).To(ConsistOf(SatisfyAll(
				HaveKeyWithValue("offset", uint32(0)),
				HaveKeyWithValue("bytes", uint32(40)),
			)))
		})

		It("Should keep an index pointer whose file is missing as a mark, and cut the "+
			"data that starts in its range", func() {
			root.channel(idx(1))
			root.file(1, 2, stamps(4, 5, 6))
			root.pointers(1, old{1, 4, 1, 0, 24}, old{4, 7, 2, 0, 24})
			root.channel(data(telem.Int64T))
			root.file(2, 1, telem.NewSeriesV[int64](2, 3, 4, 5, 6).Data)
			root.pointers(2, old{2, 7, 1, 0, 40})
			s, chs := read()
			Expect(chs).To(HaveLen(2))
			Expect(s.Channels).To(HaveLen(1))
			Expect(s.Channels[0].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(4, 6, 1, 0, 0, 24, 3)},
			))
			Expect(warned(noStamps)).To(ConsistOf(SatisfyAll(
				HaveKeyWithValue("offset", uint32(0)),
				HaveKeyWithValue("bytes", uint32(40)),
			)))
		})

		It("Should not give the samples after a piece that a bad stamp ended the "+
			"stamps of a later pointer", func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 0, 4, 5, 6, 7, 8, 9))
			root.pointers(1, old{1, 7, 1, 0, 48}, old{7, 10, 1, 48, 24})
			root.channel(data(telem.Int64T))
			root.file(2, 1, telem.NewSeriesV[int64](1, 2, 3).Data)
			// An explicit end reaches into the next index pointer.
			root.pointers(2, old{1, 10, 1, 0, 24})
			s, _ := read()
			Expect(s.Channels[1].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 2, 1, 0, 0, 16, 2)},
			))
			Expect(warned(noStamps)).To(ConsistOf(SatisfyAll(
				HaveKeyWithValue("offset", uint32(16)),
				HaveKeyWithValue("bytes", uint32(8)),
			)))
		})

		It("Should end a data piece at its last stamp inside its index pointer, and "+
			"keep the zero inside the index pointer for the check pass", func() {
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 0, 4, 5, 6))
			root.pointers(1, old{1, 7, 1, 0, 48})
			root.channel(data(telem.Int64T))
			root.file(2, 1, telem.NewSeriesV[int64](1, 2, 3).Data)
			root.pointers(2, old{1, 4, 1, 0, 24})
			s, _ := read()
			Expect(s.Channels[0].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 6, 1, 0, 0, 48, 6)},
			))
			Expect(s.Channels[1].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 2, 1, 0, 0, 16, 2)},
			))
			Expect(warned(noStamps)).To(ConsistOf(SatisfyAll(
				HaveKeyWithValue("offset", uint32(16)),
				HaveKeyWithValue("bytes", uint32(8)),
			)))
		})
	})

	It(
		"Should refuse a data key whose index is not an index of the old layout",
		func() {
			root.channel(data(telem.Int64T))
			Expect(convert.Read(cfg())).Error().To(MatchError(
				"key 2 is timed by key 1, which is no index of the old layout",
			))
		},
	)

	Describe("Reads of an old file", func() {
		faulty := func(repeated bool) {
			GinkgoHelper()
			fs = MustSucceed(fault.New(fault.Config{
				RunKey:             7,
				ReadCorruptionRate: 1,
				ReadFaultsRepeated: repeated,
			}))
			root = oldRoot{fs: fs}
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 3))
			root.pointers(1, old{1, 4, 1, 0, 24})
		}

		It("Should use the bytes of two agreeing reads in three", func() {
			faulty(false)
			s, _ := read()
			Expect(s.Channels[0].Segments[0].Pointers).To(Equal(
				[]state.Pointer{ptr(1, 3, 1, 0, 0, 24, 3)},
			))
		})

		It("Should refuse a file with no two agreeing reads in three", func() {
			faulty(true)
			Expect(convert.Read(cfg())).Error().To(SatisfyAll(
				MatchError(place.ErrCorrupt),
				MatchError(ContainSubstring(
					"of 1/meta.json have no two agreeing reads in three",
				)),
			))
		})
	})
})
