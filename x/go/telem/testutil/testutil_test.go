// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Test Util Test", func() {
	Describe("EqualUnmarshall", func() {
		It("Positive success", func() {
			marshalled := telem.MarshalSlice([]uint32{1, 2, 3, 4}, telem.Uint32T)
			Expect(marshalled).To(EqualUnmarshal([]uint32{1, 2, 3, 4}))
		})
		It("Negative success - different type", func() {
			marshalled := telem.MarshalSlice([]uint32{1, 2, 3, 4}, telem.Uint32T)
			Expect(marshalled).ToNot(EqualUnmarshal([]uint64{1, 2, 3, 4}))
		})
		It("Negative success - different value", func() {
			marshalled := telem.MarshalSlice([]uint32{1, 2, 3, 4}, telem.Uint32T)
			Expect(marshalled).ToNot(EqualUnmarshal([]uint32{1, 2, 0, 3}))
		})
	})
})
