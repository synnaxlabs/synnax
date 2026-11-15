// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/color/versions"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Color", func() {
	Describe("Parse", func() {
		It("Should parse an 8 digit hex color", func() {
			Expect(versions.Parse("#12345678")).
				To(Equal(versions.Color{R: 0x12, G: 0x34, B: 0x56, A: 0x78}))
		})
		It("Should reject an invalid hex color", func() {
			Expect(
				versions.Parse("#12345"),
			).Error().
				To(MatchError(validate.ErrValidation))
		})
	})

	Describe("FromUint32", func() {
		It("Should unpack a 0xRRGGBBAA color", func() {
			Expect(versions.FromUint32(0x12345678)).
				To(Equal(versions.Color{R: 0x12, G: 0x34, B: 0x56, A: 0x78}))
		})
	})
})
