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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	db         = 0x1122334455667788
	generation = 3
)

// sealedFrame sets the crc field of the frame in b to the checksum of the bytes it
// covers at offset, so a spec can change a field and keep the check.
func sealedFrame(b []byte, offset uint64) []byte {
	crc := place.Log(db, generation, offset).Sum(b[8:])
	binary.LittleEndian.PutUint32(b[4:], crc)
	return b
}

// sealedHeader sets the checksum of the header copy in b at offset.
func sealedHeader(b []byte) []byte {
	crc := place.Log(db, generation, 0).Sum(b[:60])
	binary.LittleEndian.PutUint32(b[60:], crc)
	return b
}

var header = log.Header{
	DB:         db,
	Generation: generation,
	Sequence:   41,
	Size:       log.FileSize,
	PrevCRC:    0xCAFEF00D,
}

var _ = Describe("Frame", func() {
	entries := entryOf(&log.Drop{Key: 3})

	It("Should encode a frame in the layout of the format", func() {
		f := log.Frame{Count: 1, Sequence: 9, PrevCRC: 0xABCD, Entries: entries}
		b := f.Append(nil, db, generation, 8_192)
		Expect(b).To(HaveLen(f.Size()))
		Expect(b[:32]).To(Equal(concat(
			le32(0x464C5043, f.CRC, uint32(len(entries)), 1),
			le64(9),
			le32(0xABCD, 0),
		)))
		Expect(f.CRC).To(Equal(place.Log(db, generation, 8_192).Sum(b[8:])))
		Expect(b[32:]).To(Equal(entries))
	})

	It("Should decode a frame that it encodes", func() {
		f := log.Frame{Count: 1, Sequence: 9, PrevCRC: 7, Entries: entries}
		b := f.Append([]byte("prefix"), db, generation, 12_288)
		Expect(log.DecodeFrame(b[6:], db, generation, 12_288)).To(Equal(f))
	})

	It("Should decode a clean mark", func() {
		f := log.Frame{Sequence: 10, PrevCRC: 7, Clean: true}
		b := f.Append(nil, db, generation, 8_192)
		Expect(binary.LittleEndian.Uint32(b[28:])).To(Equal(uint32(1)))
		decoded := MustSucceed(log.DecodeFrame(b, db, generation, 8_192))
		Expect(decoded.Clean).To(BeTrue())
		Expect(decoded.Count).To(BeZero())
	})

	It("Should ignore bytes after the frame", func() {
		f := log.Frame{Count: 1, Sequence: 9, Entries: entries}
		b := append(f.Append(nil, db, generation, 8_192), 1, 2, 3)
		Expect(MustSucceed(log.DecodeFrame(b, db, generation, 8_192)).Entries).
			To(Equal(entries))
	})

	DescribeTable(
		"Should refuse a frame that fails its check at its place",
		func(dbOf, generationOf, offset uint64) {
			f := log.Frame{Count: 1, Sequence: 9, Entries: entries}
			b := f.Append(nil, db, generation, 8_192)
			Expect(
				log.DecodeFrame(b, dbOf, generationOf, offset),
			).Error().
				To(SatisfyAll(
					MatchError(place.ErrCorrupt),
					MatchError(ContainSubstring("fails its check")),
				))
		},
		Entry("another database", uint64(1), uint64(generation), uint64(8_192)),
		Entry("another generation", uint64(db), uint64(4), uint64(8_192)),
		Entry("another offset", uint64(db), uint64(generation), uint64(12_288)),
	)

	DescribeTable(
		"Should refuse a frame that breaks the format",
		func(b []byte, message string) {
			Expect(log.DecodeFrame(b, db, generation, 8_192)).Error().To(SatisfyAll(
				MatchError(place.ErrCorrupt),
				MatchError(ContainSubstring(message)),
			))
		},
		refusalEntries(frameRefusals),
	)
})

var _ = Describe("Header", func() {
	DescribeTable(
		"Should decode a copy that it encodes",
		func(offset uint64) {
			Expect(
				log.DecodeHeader(header.Encode(offset), db, generation, offset),
			).To(Equal(header))
		},
		Entry("first copy", uint64(0)),
		Entry("second copy", uint64(log.HeaderCopy)),
	)

	It("Should encode a header in the layout of the format", func() {
		b := header.Encode(0)
		Expect(b).To(Equal(concat(
			[]byte("CSPTRLOG"),
			[]byte{1, 0, 0, 0, 0, 0, 0, 0},
			le64(db, generation, 41, log.FileSize),
			le32(0xCAFEF00D, 0, 0),
			le32(place.Log(db, generation, 0).Sum(b[:60])),
		)))
	})

	It("Should refuse a copy decoded at the offset of the other copy", func() {
		Expect(
			log.DecodeHeader(header.Encode(0), db, generation, log.HeaderCopy),
		).Error().
			To(SatisfyAll(
				MatchError(place.ErrCorrupt),
				MatchError(ContainSubstring("log header at 4096 fails its check")),
			))
	})

	DescribeTable(
		"Should refuse a copy of another log",
		func(dbOf, generationOf uint64) {
			Expect(log.DecodeHeader(header.Encode(0), dbOf, generationOf, 0)).Error().
				To(SatisfyAll(
					MatchError(place.ErrCorrupt),
					MatchError(ContainSubstring("log header at 0 fails its check")),
				))
		},
		Entry("another database", uint64(1), uint64(generation)),
		Entry("another generation", uint64(db), uint64(4)),
	)

	It("Should refuse a copy without the magic", func() {
		b := header.Encode(0)
		b[0] = 'X'
		Expect(log.DecodeHeader(sealedHeader(b), db, generation, 0)).Error().
			To(MatchError(ContainSubstring("fails its check")))
	})

	It("Should refuse a log format version that this binary does not know", func() {
		b := header.Encode(0)
		b[8] = 2
		Expect(
			log.DecodeHeader(sealedHeader(b), db, generation, 0),
		).Error().
			To(SatisfyAll(
				MatchError(log.ErrUnknown),
				MatchError(ContainSubstring("log format version 2")),
			))
	})

	DescribeTable(
		"Should refuse a copy with a set reserved byte",
		func(at int) {
			b := header.Encode(0)
			b[at] = 1
			Expect(
				log.DecodeHeader(sealedHeader(b), db, generation, 0),
			).Error().
				To(SatisfyAll(
					MatchError(place.ErrCorrupt),
					MatchError(ContainSubstring("set reserved byte")),
				))
		},
		Entry("after the version", 12),
		Entry("after the link", 55),
	)

	It("Should refuse a short copy", func() {
		Expect(log.DecodeHeader(make([]byte, 63), db, generation, 0)).Error().
			To(MatchError(ContainSubstring("needs 64 bytes, but 63 remain")))
	})
})
