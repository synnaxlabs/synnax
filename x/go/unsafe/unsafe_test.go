// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unsafe_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/unsafe"
)

var _ = Describe("Unsafe", func() {
	Describe("ReinterpretSlice", func() {
		type myCustomUint32 uint32
		It("should convert a slice of one type to a slice of another type", func() {
			in := []uint32{1, 2, 3}
			out := unsafe.ReinterpretSlice[uint32, myCustomUint32](in)
			Expect(out).To(Equal([]myCustomUint32{1, 2, 3}))
		})
	})
	Describe("ReinterpretMap", func() {
		type myCustomUint32 uint32
		type myCustomUint64 uint64
		It("should convert a map of one type to a map of another type", func() {
			in := map[uint32]uint64{1: 1, 2: 2, 3: 3}
			out := unsafe.ReinterpretMap[uint32, uint64, myCustomUint32, myCustomUint64](in)
			Expect(out).To(Equal(map[myCustomUint32]myCustomUint64{1: 1, 2: 2, 3: 3}))
		})
	})
})
