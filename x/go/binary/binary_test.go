// Copyright 2026 Synnax Labs, Inc.
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

})
