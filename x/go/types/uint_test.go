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

var _ = Describe("Uint", func() {
	Describe("BoolToUint8", func() {
		It("Should return 1 if the bool is true", func() {
			Expect(types.BoolToUint8(true)).To(Equal(uint8(1)))
		})
		It("Should return 0 if the bool is false", func() {
			Expect(types.BoolToUint8(false)).To(Equal(uint8(0)))
		})
	})

})
