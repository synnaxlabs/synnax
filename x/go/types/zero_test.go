// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/types"
)

type zeroStruct struct {
	Name  string
	Count int
}

var _ = Describe("Zero", func() {
	It("Should return 0 for a numeric type", func() {
		Expect(types.Zero[int]()).To(Equal(0))
		Expect(types.Zero[float64]()).To(Equal(float64(0)))
	})

	It("Should return an empty string for a string type", func() {
		Expect(types.Zero[string]()).To(Equal(""))
	})

	It("Should return false for a bool type", func() {
		Expect(types.Zero[bool]()).To(BeFalse())
	})

	It("Should return the zero value for a struct type", func() {
		Expect(types.Zero[zeroStruct]()).To(Equal(zeroStruct{}))
	})

	It("Should return nil for pointer, slice, and map types", func() {
		Expect(types.Zero[*zeroStruct]()).To(BeNil())
		Expect(types.Zero[[]int]()).To(BeNil())
		Expect(types.Zero[map[string]int]()).To(BeNil())
	})
})
