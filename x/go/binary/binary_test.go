// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package binary_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary"
)

var _ = Describe("Binary", func() {
	Describe("MakeCopy", func() {
		It("Should return a copy of the given byte slice", func() {
			bytes := []byte("hello")
			copied := binary.MakeCopy(bytes)
			Expect(copied).To(Equal(bytes))
			Expect(copied).ToNot(BeIdenticalTo(bytes))
		})
	})
	Describe("DecodeVarint", func() {
		It("should decode a varint", func() {
			buf := []byte{0x80, 0x01}
			value, bytesRead, err := binary.UVarint(buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(uint64(128)))
			Expect(bytesRead).To(Equal(2))
		})
		It("should decode zero", func() {
			buf := []byte{0x00}
			value, bytesRead, err := binary.UVarint(buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(uint64(0)))
			Expect(bytesRead).To(Equal(1))
		})
		It("should error if the varint is not found", func() {
			buf := []byte{0x80}
			_, _, err := binary.UVarint(buf)
			Expect(err).To(MatchError("buffer too small"))
		})
		It("should error if the varint is too large", func() {
			buf := []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x00}
			_, _, err := binary.UVarint(buf)
			Expect(err).To(MatchError("varint too large"))
		})
	})
	Describe("VarintLength", func() {
		It("should return the number of bytes required to encode the given uint64", func() {
			Expect(binary.VarintLength(0)).To(Equal(1))
			Expect(binary.VarintLength(1<<(7) - 1)).To(Equal(1))
			Expect(binary.VarintLength(1 << (7))).To(Equal(2))
			Expect(binary.VarintLength(1<<(7*2) - 1)).To(Equal(2))
			Expect(binary.VarintLength(1 << (7 * 2))).To(Equal(3))
			Expect(binary.VarintLength(1<<(7*3) - 1)).To(Equal(3))
			Expect(binary.VarintLength(1 << (7 * 3))).To(Equal(4))
			Expect(binary.VarintLength(1<<(7*4) - 1)).To(Equal(4))
			Expect(binary.VarintLength(1 << (7 * 4))).To(Equal(5))
			Expect(binary.VarintLength(1<<(7*5) - 1)).To(Equal(5))
			Expect(binary.VarintLength(1 << (7 * 5))).To(Equal(6))
			Expect(binary.VarintLength(1<<(7*6) - 1)).To(Equal(6))
			Expect(binary.VarintLength(1 << (7 * 6))).To(Equal(7))
			Expect(binary.VarintLength(1<<(7*7) - 1)).To(Equal(7))
			Expect(binary.VarintLength(1 << (7 * 7))).To(Equal(8))
			Expect(binary.VarintLength(1<<(7*8) - 1)).To(Equal(8))
			Expect(binary.VarintLength(1 << (7 * 8))).To(Equal(9))
			Expect(binary.VarintLength(1<<(7*9) - 1)).To(Equal(9))
			Expect(binary.VarintLength(1 << (7 * 9))).To(Equal(10))
			Expect(binary.VarintLength(^uint(0))).To(Equal(10))
		})
	})
})
