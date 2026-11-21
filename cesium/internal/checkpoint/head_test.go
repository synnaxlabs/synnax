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
	"encoding/binary"
	"os"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Head", func() {
	DescribeTable(
		"Should decode the state that it encodes",
		func(s *state.Snapshot) {
			b := checkpoint.Encode(db, generation, s)
			h, got := MustSucceed2(decode(b))
			Expect(h.DB).To(Equal(uint64(db)))
			Expect(h.Generation).To(Equal(uint64(generation)))
			Expect(h.Sequence).To(Equal(s.Sequence))
			Expect(h.Limits).To(Equal(s.Limits))
			Expect(h.Length).To(Equal(uint64(len(b))))
			Expect(got.Sequence).To(Equal(s.Sequence))
			Expect(got.Limits).To(Equal(s.Limits))
			Expect(got.Channels).To(HaveLen(len(s.Channels)))
			for i, c := range s.Channels {
				Expect(*got.Channels[i]).To(Equal(*c))
			}
			Expect(got.Intents).To(Equal(s.Intents))
			Expect(checkpoint.Encode(db, generation, got)).To(Equal(b))
		},
		Entry("a small state", sample(2)),
		Entry("a state over three chunks", sample(4000)),
		Entry("an empty state", &state.Snapshot{Sequence: 1}),
	)

	It("Should lay out the header sector, the sections, and both tables", func() {
		s := sample(2)
		b := checkpoint.Encode(db, generation, s)
		Expect(b[:8]).To(Equal([]byte("CSCHKPT\x00")))
		Expect(binary.LittleEndian.Uint16(b[8:])).To(Equal(uint16(1)))
		Expect(b[10]).To(Equal(uint8(4)))
		wants := []uint64{db, generation, 77, 64, 1 << 20, 3, uint64(len(b))}
		for i, want := range wants {
			Expect(binary.LittleEndian.Uint64(b[16+8*i:])).To(Equal(want))
		}
		table := binary.LittleEndian.Uint64(b[72:])
		Expect(binary.LittleEndian.Uint32(b[80:])).To(Equal(uint32(9)))
		chunks := binary.LittleEndian.Uint64(b[88:])
		Expect(chunks).To(Equal(table + 9*24))
		Expect(uint64(len(b))).To(Equal(chunks + 4))
		Expect(b[100:508]).To(Equal(make([]byte, 408)))
		Expect(b[512:4096]).To(Equal(make([]byte, 3584)))
		Expect(binary.LittleEndian.Uint32(b[508:])).
			To(Equal(place.Head(db, generation, 0).Sum(b[:508])))
		Expect(binary.LittleEndian.Uint32(b[96:])).
			To(Equal(place.Head(db, generation, chunks).Sum(b[chunks:])))
		Expect(binary.LittleEndian.Uint32(b[chunks:])).
			To(Equal(place.Head(db, generation, 4096).Sum(b[4096:chunks])))
		sizes := []uint32{48, 1, 24, 20, 32, 36, 96, 4, 4}
		counts := []uint64{0, 0, 0, 3, 5, 5, 0, 2, 2}
		for kind := 1; kind <= 9; kind++ {
			e := b[table+uint64(kind-1)*24:]
			Expect(binary.LittleEndian.Uint16(e)).To(Equal(uint16(kind)))
			Expect(binary.LittleEndian.Uint16(e[2:])).To(Equal(uint16(1)))
			Expect(binary.LittleEndian.Uint32(e[4:])).To(Equal(sizes[kind-1]))
			off, count := section(b, kind)
			Expect(off % 8).To(BeZero())
			Expect(count).To(Equal(counts[kind-1]), "section %d", kind)
		}
	})

	It("Should write each record with the fields of the format in order", func() {
		b := checkpoint.Encode(db, generation, sample(2))
		chans, _ := section(b, 4)
		Expect(b[chans+20 : chans+40]).To(Equal(le32(6, 2, 0, 0xFFFFFFFF, 0xFFFFFFFF)))
		segs, _ := section(b, 5)
		want := binary.LittleEndian.AppendUint64(nil, 12)
		want = append(want, le32(5, 80, 0xABCD, 1)...)
		want = binary.LittleEndian.AppendUint16(want, 1)
		want = append(want, 7, 0, 0, 0, 0, 0)
		Expect(b[segs+32 : segs+64]).To(Equal(want))
		Expect(b[segs+27]).To(Equal(uint8(1)), "the first segment is sealed")
		ptrs, _ := section(b, 6)
		want = binary.LittleEndian.AppendUint64(nil, 1010)
		want = binary.LittleEndian.AppendUint64(want, 1014)
		want = binary.LittleEndian.AppendUint64(want, 11<<32)
		want = append(want, le32(40, 40, 5)...)
		Expect(b[ptrs+72 : ptrs+108]).To(Equal(want))
		clocks, _ := section(b, 8)
		Expect(b[clocks : clocks+8]).To(Equal(le32(5, 7)))
		intents, _ := section(b, 9)
		Expect(b[intents : intents+8]).To(Equal(le32(2, 6)))
	})

	It("Should encode one state to the same bytes every time", func() {
		Expect(checkpoint.Encode(db, generation, sample(300))).
			To(Equal(checkpoint.Encode(db, generation, sample(300))))
	})

	It("Should report a zero header sector as incomplete", func() {
		b := checkpoint.Encode(db, generation, sample(2))
		clear(b[:512])
		Expect(decodeErr(b)).To(MatchError(checkpoint.ErrIncomplete))
	})

	DescribeTable(
		"Should refuse a head whose bytes fail a checksum",
		func(at func(b []byte) int, msg string) {
			b := checkpoint.Encode(db, generation, sample(4000))
			b[at(b)] ^= 0x10
			Expect(decodeErr(b)).To(And(
				MatchError(place.ErrCorrupt),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("header sector", func([]byte) int { return 30 }, "header sector fails"),
		Entry("first chunk", func([]byte) int { return 4096 }, "chunk at 4096 fails"),
		Entry("third chunk", func([]byte) int { return 4096 + 2*65_536 + 5 },
			"chunk at 135168 fails"),
		Entry("section table", func(b []byte) int {
			return int(binary.LittleEndian.Uint64(b[72:])) + 3
		}, "chunk at 135168 fails"),
		Entry("chunk table", func(b []byte) int { return len(b) - 1 },
			"chunk table fails"),
	)
})

// withRecord returns head b with record added at the end of section kind, the later
// sections and tables moved to fit, and every checksum set again.
func withRecord(b []byte, kind int, record []byte) []byte {
	off, count := section(b, kind)
	size := uint64(len(record))
	at := off + count*size
	grow := (size + 7) &^ 7
	grown := slices.Concat(b[:at], record, make([]byte, grow-size), b[at:])
	tab := binary.LittleEndian.Uint64(grown[72:]) + grow
	for k := 1; k <= 9; k++ {
		e := tab + uint64(k-1)*24
		switch {
		case k == kind:
			binary.LittleEndian.PutUint64(grown[e+16:], count+1)
		case k > kind:
			o := binary.LittleEndian.Uint64(grown[e+8:])
			binary.LittleEndian.PutUint64(grown[e+8:], o+grow)
		}
	}
	chunks := tab + 9*24
	body := chunks - checkpoint.BodyStart
	n := (body + checkpoint.ChunkSize - 1) / checkpoint.ChunkSize
	grown = append(grown[:chunks], make([]byte, 4*n)...)
	binary.LittleEndian.PutUint64(grown[64:], uint64(len(grown)))
	binary.LittleEndian.PutUint64(grown[72:], tab)
	binary.LittleEndian.PutUint64(grown[88:], chunks)
	return reseal(grown)
}

// decode writes the head bytes b to a memory file system and reads them back.
func decode(b []byte) (checkpoint.Header, *state.Snapshot, error) {
	fs := xfs.NewMem()
	f, err := fs.Open(checkpoint.Name(generation), os.O_CREATE|os.O_RDWR)
	if err != nil {
		return checkpoint.Header{}, nil, err
	}
	if _, err = f.Write(b); err != nil {
		return checkpoint.Header{}, nil, err
	}
	if err = f.Close(); err != nil {
		return checkpoint.Header{}, nil, err
	}
	return checkpoint.Read(fs, generation)
}

func decodeErr(b []byte) error {
	_, _, err := decode(b)
	return err
}

func le32(vs ...uint32) []byte {
	var b []byte
	for _, v := range vs {
		b = binary.LittleEndian.AppendUint32(b, v)
	}
	return b
}

// refusals mutate a head of sample(2) before reseal. Each refusal names the error it
// wraps and a part of its message.
var _ = Describe("Head checks", func() {
	DescribeTable(
		"Should refuse a head that breaks a rule of the format",
		func(mutate func(b []byte) []byte, wraps error, msg string) {
			b := reseal(mutate(checkpoint.Encode(db, generation, sample(2))))
			Expect(decodeErr(b)).To(And(
				MatchError(wraps),
				MatchError(ContainSubstring(msg)),
			))
		},
		headRefusalEntries(),
	)

	It("Should refuse a head with parts or tombstones as unknown", func() {
		b := withRecord(
			checkpoint.Encode(db, generation, sample(2)),
			1,
			make([]byte, 48),
		)
		Expect(decodeErr(b)).To(And(
			MatchError(checkpoint.ErrUnknown),
			MatchError(ContainSubstring("section 1")),
		))
	})

	It("Should refuse segment records that no channel names", func() {
		empty := checkpoint.Encode(db, generation, &state.Snapshot{
			Limits: state.Limits{Segments: 9},
		})
		record := binary.LittleEndian.AppendUint64(nil, 1)
		record = append(record, make([]byte, 16)...)
		record = binary.LittleEndian.AppendUint16(record, 1)
		record = append(record, make([]byte, 6)...)
		Expect(decodeErr(withRecord(empty, 5, record))).To(And(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring("1 segment and 0 pointer records, but the "+
				"channels name 0 and 0")),
		))
	})

	It("Should refuse a section table inside the first 4 KiB", func() {
		b := checkpoint.Encode(db, generation, &state.Snapshot{})
		tab := binary.LittleEndian.Uint64(b[72:])
		moved := slices.Concat(b[:4088], b[tab:tab+9*24], make([]byte, 4))
		binary.LittleEndian.PutUint64(moved[64:], uint64(len(moved)))
		binary.LittleEndian.PutUint64(moved[72:], 4088)
		binary.LittleEndian.PutUint64(moved[88:], 4088+9*24)
		Expect(decodeErr(reseal(moved))).To(And(
			MatchError(place.ErrCorrupt),
			MatchError(
				ContainSubstring("section table at 4088 and chunk table at 4304"),
			),
		))
	})

	It("Should refuse a gap before the section table", func() {
		b := checkpoint.Encode(db, generation, sample(2))
		tab := binary.LittleEndian.Uint64(b[72:])
		grown := slices.Concat(b[:tab], make([]byte, 8), b[tab:])
		binary.LittleEndian.PutUint64(grown[64:], uint64(len(grown)))
		binary.LittleEndian.PutUint64(grown[72:], tab+8)
		binary.LittleEndian.PutUint64(grown[88:], tab+8+9*24)
		Expect(decodeErr(reseal(grown))).To(And(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring("does not follow the last section")),
		))
	})
})
