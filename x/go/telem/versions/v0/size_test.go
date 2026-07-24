// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/x/telem/versions/v0"
)

var _ = Describe("Size", func() {
	Describe("Gigabytes", func() {
		It("Should return the correct number of gigabytes", func() {
			s := v0.Gigabyte + v0.Megabyte + v0.Kilobyte + 42*v0.Byte
			Expect(s.Gigabytes()).To(Equal(1.001001042))
		})
	})
})
