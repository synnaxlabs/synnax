// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"encoding/binary"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
)

// decodeAll decodes the entry at the start of b and returns copies of its records.
func decodeAll(b []byte) ([]any, error) {
	var d log.Decoder
	if _, err := d.Reset(b); err != nil {
		return nil, err
	}
	var out []any
	for {
		r, ok, err := d.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			return out, nil
		}
		switch v := r.(type) {
		case *log.Reserve:
			out = append(out, *v)
		case *log.Create:
			out = append(out, *v)
		case *log.Commit:
			c := *v
			c.Slots = append([]log.Slot(nil), v.Slots...)
			out = append(out, c)
		case *log.Seal:
			out = append(out, *v)
		case *log.Drop:
			out = append(out, *v)
		case *log.Restart:
			out = append(out, *v)
		case *log.Intent:
			out = append(out, *v)
		}
	}
}

// entryOf returns the bytes of an entry that holds records.
func entryOf(records ...log.Record) []byte {
	var e log.Entry
	for _, r := range records {
		e.Add(r)
	}
	return e.Bytes()
}

// raw is a record given as its type and body, with no checks.
type raw struct {
	t    uint16
	body []byte
}

// rawEntry returns an entry of records.
func rawEntry(records ...raw) []byte {
	var body []byte
	for _, r := range records {
		body = binary.LittleEndian.AppendUint16(body, r.t)
		body = binary.LittleEndian.AppendUint16(body, 0)
		body = binary.LittleEndian.AppendUint32(body, uint32(len(r.body)))
		body = append(body, r.body...)
	}
	out := binary.LittleEndian.AppendUint32(nil, uint32(len(body)))
	out = binary.LittleEndian.AppendUint32(out, uint32(len(records)))
	return append(out, body...)
}

func le32(vs ...uint32) []byte {
	var b []byte
	for _, v := range vs {
		b = binary.LittleEndian.AppendUint32(b, v)
	}
	return b
}

func le64(vs ...uint64) []byte {
	var b []byte
	for _, v := range vs {
		b = binary.LittleEndian.AppendUint64(b, v)
	}
	return b
}

func concat(parts ...[]byte) []byte {
	var b []byte
	for _, p := range parts {
		b = append(b, p...)
	}
	return b
}

var commit = log.Commit{
	Samples:   3,
	Alignment: 5<<32 | 7,
	First:     100,
	Last:      300,
	Slots: []log.Slot{
		{Segment: 2, Length: 24, TailCRC: 0xAABBCCDD},
		{Segment: 9, Length: 12, TailCRC: 0x11223344},
	},
}

var _ = Describe("Record", func() {
	It("Should decode every record that an entry holds, in order", func() {
		records := []log.Record{
			&log.Reserve{Segments: 17, Alignments: 1 << 20},
			&log.Create{Segment: 2, Key: 4, Clock: 4, Version: 1, DataType: 6},
			&commit,
			&log.Seal{Segment: 2, Length: 24, TailCRC: 0xAABBCCDD},
			&log.Drop{Key: 8},
			&log.Restart{},
			&log.Intent{Key: 8},
			&log.Intent{Key: 8, Clear: true},
		}
		Expect(decodeAll(entryOf(records...))).To(Equal([]any{
			log.Reserve{Segments: 17, Alignments: 1 << 20},
			log.Create{Segment: 2, Key: 4, Clock: 4, Version: 1, DataType: 6},
			commit,
			log.Seal{Segment: 2, Length: 24, TailCRC: 0xAABBCCDD},
			log.Drop{Key: 8},
			log.Restart{},
			log.Intent{Key: 8},
			log.Intent{Key: 8, Clear: true},
		}))
	})

	DescribeTable(
		"Should encode each record in the layout of the format",
		func(r log.Record, want []byte) {
			Expect(entryOf(r)).To(Equal(concat(le32(uint32(len(want)), 1), want)))
		},
		Entry(
			"reserve",
			&log.Reserve{Segments: 17, Alignments: 9},
			concat([]byte{1, 0, 0, 0}, le32(16), le64(17, 9)),
		),
		Entry(
			"create",
			&log.Create{Segment: 2, Key: 4, Clock: 5, Version: 1, DataType: 6},
			concat(
				[]byte{2, 0, 0, 0},
				le32(24),
				le64(2),
				le32(4, 5),
				[]byte{1, 0, 6, 0, 0, 0, 0, 0},
			),
		),
		Entry(
			"commit",
			&commit,
			concat(
				[]byte{3, 0, 0, 0},
				le32(64),
				le32(2, 3),
				le64(5<<32|7, 100, 300),
				le64(2),
				le32(24, 0xAABBCCDD),
				le64(9),
				le32(12, 0x11223344),
			),
		),
		Entry(
			"seal",
			&log.Seal{Segment: 2, Length: 24, TailCRC: 7},
			concat([]byte{4, 0, 0, 0}, le32(16), le64(2), le32(24, 7)),
		),
		Entry(
			"drop",
			&log.Drop{Key: 8},
			concat([]byte{7, 0, 0, 0}, le32(8), le32(8, 0)),
		),
		Entry("restart", &log.Restart{}, concat([]byte{8, 0, 0, 0}, le32(0))),
		Entry(
			"intent",
			&log.Intent{Key: 8},
			concat([]byte{10, 0, 0, 0}, le32(8), le32(8, 0)),
		),
		Entry(
			"intent clear",
			&log.Intent{Key: 8, Clear: true},
			concat([]byte{10, 0, 0, 0}, le32(8), le32(8, 1)),
		),
	)

	It(
		"Should return the length of the entry, so entries can follow one another",
		func() {
			first := entryOf(&log.Drop{Key: 1})
			b := append(append([]byte(nil), first...), entryOf(&log.Drop{Key: 2})...)
			var d log.Decoder
			Expect(d.Reset(b)).To(Equal(len(first)))
			Expect(decodeAll(b[len(first):])).To(Equal([]any{log.Drop{Key: 2}}))
		},
	)

	It("Should decode an entry with no record", func() {
		var e log.Entry
		Expect(e.Bytes()).To(Equal(le32(0, 0)))
		Expect(decodeAll(e.Bytes())).To(BeEmpty())
	})

	It("Should count and reset the records of an entry", func() {
		var e log.Entry
		e.Add(&log.Drop{Key: 1})
		e.Add(&log.Restart{})
		Expect(e.Len()).To(Equal(2))
		e.Reset()
		Expect(e.Len()).To(BeZero())
		e.Add(&log.Drop{Key: 3})
		Expect(decodeAll(e.Bytes())).To(Equal([]any{log.Drop{Key: 3}}))
	})

	It("Should skip a record of an unknown type that an older binary may skip", func() {
		b := rawEntry(
			raw{1<<15 | 40, []byte{1, 2, 3}},
			raw{7, le32(5, 0)},
		)
		Expect(decodeAll(b)).To(Equal([]any{log.Drop{Key: 5}}))
	})

	It(
		"Should refuse a record of an unknown type that an older binary may not skip",
		func() {
			Expect(decodeAll(rawEntry(raw{5, make([]byte, 72)}))).Error().
				To(SatisfyAll(
					MatchError(log.ErrUnknown),
					MatchError(ContainSubstring("record type 5")),
				))
		},
	)

	It("Should refuse a segment format version that this binary does not know", func() {
		body := concat(le64(2), le32(4, 4), []byte{2, 0, 6, 0, 0, 0, 0, 0})
		Expect(decodeAll(rawEntry(raw{2, body}))).Error().
			To(SatisfyAll(
				MatchError(log.ErrUnknown),
				MatchError(ContainSubstring("format version 2")),
			))
	})

	DescribeTable(
		"Should refuse an entry that breaks the format",
		func(b []byte, message string) {
			Expect(decodeAll(b)).Error().To(SatisfyAll(
				MatchError(place.ErrCorrupt),
				MatchError(ContainSubstring(message)),
			))
		},
		Entry("a short entry header", []byte{1, 2, 3}, "entry header needs 8 bytes"),
		Entry(
			"records past the end",
			concat(le32(100, 1), make([]byte, 8)),
			"entry holds 100 bytes of records",
		),
		Entry(
			"a count that its bytes cannot hold",
			concat(le32(8, 2), []byte{8, 0, 0, 0}, le32(0)),
			"cannot hold 2 records",
		),
		Entry(
			"bytes after the last record",
			concat(le32(16, 1), []byte{8, 0, 0, 0}, le32(0), make([]byte, 8)),
			"8 bytes after its last record",
		),
		Entry(
			"fewer records than its count",
			concat(le32(12, 1), []byte{8, 0, 0, 0}, le32(4), make([]byte, 4)),
			"restart record has 4 bytes, not 0",
		),
		Entry(
			"a set reserved field in a record header",
			concat(le32(8, 1), []byte{8, 0, 1, 0}, le32(0)),
			"set reserved field",
		),
		Entry(
			"a record past its entry",
			concat(le32(8, 1), []byte{7, 0, 0, 0}, le32(8)),
			"holds 8 bytes, but 0 remain",
		),
		Entry(
			"a reserve above the alignment limit",
			rawEntry(raw{1, le64(1, 1<<31+1)}),
			"alignment limit 2147483649",
		),
		Entry(
			"a create of segment 0",
			rawEntry(raw{2, concat(le64(0), le32(1, 1), le64(1))}),
			"create record has segment 0",
		),
		Entry(
			"a set reserved byte in a create",
			rawEntry(raw{
				2,
				concat(le64(1), le32(1, 1), []byte{1, 0, 3, 0, 0, 9, 0, 0}),
			}),
			"create record has a set reserved byte",
		),
		Entry(
			"a commit with no slot",
			rawEntry(raw{3, concat(le32(0, 1), le64(1, 1, 1))}),
			"commit record has 32 bytes",
		),
		Entry(
			"a commit whose slot count differs from its size",
			rawEntry(
				raw{3, concat(le32(2, 1), le64(1, 1, 1, 1), le32(1, 1))},
			),
			"names 2 slots",
		),
		Entry(
			"a commit that gains no samples",
			rawEntry(
				raw{3, concat(le32(1, 0), le64(1, 1, 1, 1), le32(1, 1))},
			),
			"gains no samples",
		),
		Entry(
			"a commit that ends before it starts",
			rawEntry(
				raw{3, concat(le32(1, 1), le64(1, 5, 4, 1), le32(1, 1))},
			),
			"starts at 5, after its end 4",
		),
		Entry(
			"commit slots out of order",
			rawEntry(raw{
				3,
				concat(le32(2, 1), le64(1, 1, 1, 4), le32(1, 1), le64(4), le32(1, 1)),
			}),
			"slot 1 names segment 4, not above segment 4",
		),
		Entry(
			"a commit of segment 0",
			rawEntry(
				raw{3, concat(le32(1, 1), le64(1, 1, 1, 0), le32(1, 1))},
			),
			"commit record names segment 0",
		),
		Entry(
			"a seal of segment 0",
			rawEntry(raw{4, concat(le64(0), le32(1, 1))}),
			"seal record has segment 0",
		),
		Entry(
			"a set reserved field in a drop",
			rawEntry(raw{7, le32(1, 1)}),
			"drop record has a set reserved byte",
		),
		Entry(
			"unknown intent flags",
			rawEntry(raw{10, le32(1, 2)}),
			"unknown flags 0x2",
		),
		Entry(
			"a short seal",
			rawEntry(raw{4, le64(1)}),
			"seal record has 8 bytes, not 16",
		),
	)

	It("Should decode a commit with no allocation", func() {
		b := entryOf(&commit, &commit)
		var (
			d   log.Decoder
			n   int
			err error
		)
		allocs := testing.AllocsPerRun(100, func() {
			if _, err = d.Reset(b); err != nil {
				return
			}
			for ok := true; ok && err == nil; {
				if _, ok, err = d.Next(); ok {
					n++
				}
			}
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(202))
		Expect(allocs).To(BeZero())
	})
})
