// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package place_test

import (
	"encoding/binary"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/place"
)

// crc32c is a bitwise CRC32C with an initial value and a final XOR of 0xFFFFFFFF, kept
// apart from the table code that place uses.
func crc32c(b []byte) uint32 {
	crc := ^uint32(0)
	for _, c := range b {
		crc ^= uint32(c)
		for range 8 {
			if crc&1 != 0 {
				crc = crc>>1 ^ 0x82F63B78
			} else {
				crc >>= 1
			}
		}
	}
	return ^crc
}

// encoded returns the 32 bytes of a place, built from the field table of the format.
func encoded(
	db, file, offset uint64,
	kind, dataType uint8,
	version uint16,
	clock uint32,
) []byte {
	b := make([]byte, place.Size)
	binary.LittleEndian.PutUint64(b[0:], db)
	binary.LittleEndian.PutUint64(b[8:], file)
	binary.LittleEndian.PutUint64(b[16:], offset)
	b[24] = kind
	b[25] = dataType
	binary.LittleEndian.PutUint16(b[26:], version)
	binary.LittleEndian.PutUint32(b[28:], clock)
	return b
}

var covered = []byte("covered bytes")

var _ = Describe("Place", func() {
	It("Should compute CRC32C with the standard initial value and final XOR", func() {
		Expect(place.Update(0, []byte("123456789"))).To(Equal(uint32(0xE3069283)))
	})

	DescribeTable(
		"Should sum the place and then the covered bytes",
		func(p place.Place, want []byte) {
			Expect(p.Sum(covered)).To(Equal(crc32c(append(want, covered...))))
		},
		Entry(
			"segment",
			place.Segment(7, 9, 65_536, 1, 4, 0x00200001),
			encoded(7, 9, 65_536, 1, 4, 1, 0x00200001),
		),
		Entry("check file", place.Check(8, 9, 16, 0), encoded(8, 9, 16, 2, 0, 0, 0)),
		Entry("log", place.Log(7, 3, 8_192), encoded(7, 3, 8_192, 3, 0, 1, 0)),
		Entry("head", place.Head(5, 3, 0), encoded(5, 3, 0, 4, 0, 1, 0)),
		Entry("part", place.Part(7, 12, 4_096), encoded(7, 12, 4_096, 5, 0, 1, 0)),
	)

	DescribeTable(
		"Should give another sum when one field of a segment place changes",
		func(other place.Place) {
			base := place.Segment(1, 2, 3, 1, 5, 6)
			Expect(other.Sum(covered)).ToNot(Equal(base.Sum(covered)))
		},
		Entry("database", place.Segment(9, 2, 3, 1, 5, 6)),
		Entry("file", place.Segment(1, 9, 3, 1, 5, 6)),
		Entry("offset", place.Segment(1, 2, 9, 1, 5, 6)),
		Entry("version", place.Segment(1, 2, 3, 2, 5, 6)),
		Entry("data type", place.Segment(1, 2, 3, 1, 9, 6)),
		Entry("clock", place.Segment(1, 2, 3, 1, 5, 9)),
		Entry("kind", place.Check(1, 2, 3, 1)),
	)

	It("Should move only the offset with At", func() {
		p := place.Segment(1, 2, 3, 1, 5, 6)
		Expect(p.At(100)).To(Equal(place.Segment(1, 2, 100, 1, 5, 6)))
	})

	It("Should continue a checksum over covered bytes in parts", func() {
		p := place.Log(4, 5, 6)
		crc := place.Update(p.Start(), covered[:4])
		Expect(place.Update(crc, covered[4:])).To(Equal(p.Sum(covered)))
	})
})
