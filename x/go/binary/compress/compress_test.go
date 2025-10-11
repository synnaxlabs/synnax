// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compress_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary/compress"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Compress", func() {
	var cd compress.CompressorDecompressor = compress.Bool{}
	Describe("Sub 8-bit tests", func() {
		It("Basic", func() {
			bytes := []byte{0, 1, 0, 1, 0, 1}
			compressed := MustSucceed(cd.Compress(bytes))
			result := MustSucceed(cd.Decompress(compressed))
			Expect(result).To(Equal(bytes))
		})
		It("Basic II", func() {
			bytes := []byte{1, 0, 1, 0, 1, 0}
			compressed := MustSucceed(cd.Compress(bytes))
			result := MustSucceed(cd.Decompress(compressed))
			Expect(result).To(Equal(bytes))
		})
		It("Basic III", func() {
			bytes := []byte{0, 0, 0, 1, 1, 0, 0, 0, 0, 1}
			compressed := MustSucceed(cd.Compress(bytes))
			result := MustSucceed(cd.Decompress(compressed))
			Expect(result).To(Equal(bytes))
		})
		It("Basic IV", func() {
			bytes := []byte{0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0}
			compressed := MustSucceed(cd.Compress(bytes))
			result := MustSucceed(cd.Decompress(compressed))
			Expect(result).To(Equal(bytes))
		})
		It("Longer I", func() {
			bytes := []byte{1, 1, 1, 0, 0, 1, 1, 1, 1, 0}
			compressed := MustSucceed(cd.Compress(bytes))
			result := MustSucceed(cd.Decompress(compressed))
			Expect(result).To(Equal(bytes))
		})
		It("Longer II", func() {
			bytes := []byte{0, 1, 1, 1, 1, 1, 0, 1, 1, 0, 0, 1, 1, 1, 1}
			compressed := MustSucceed(cd.Compress(bytes))
			result := MustSucceed(cd.Decompress(compressed))
			Expect(result).To(Equal(bytes))
		})
		It("Edge", func() {
			bytes := []byte{0}
			compressed := MustSucceed(cd.Compress(bytes))
			result := MustSucceed(cd.Decompress(compressed))
			Expect(result).To(Equal(bytes))
		})
		It("Edge II", func() {
			bytes := []byte{1}
			compressed := MustSucceed(cd.Compress(bytes))
			result := MustSucceed(cd.Decompress(compressed))
			Expect(result).To(Equal(bytes))
		})
		It("All Zero", func() {
			bytes := []byte{0, 0, 0, 0, 0, 0, 0}
			compressed := MustSucceed(cd.Compress(bytes))
			result := MustSucceed(cd.Decompress(compressed))
			Expect(result).To(Equal(bytes))
		})
		It("All One", func() {
			bytes := []byte{1, 1, 1, 1, 1}
			compressed := MustSucceed(cd.Compress(bytes))
			result := MustSucceed(cd.Decompress(compressed))
			Expect(result).To(Equal(bytes))
		})
	})
	Describe("Longer Tests", func() {
		It("Longer I", func() {
			bytes := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 0, 0, 0, 0,
			}
			compressed := MustSucceed(cd.Compress(bytes))
			result := MustSucceed(cd.Decompress(compressed))
			Expect(result).To(Equal(bytes))
		})
	})
})
