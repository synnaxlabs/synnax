// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/pb"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Narrow", func() {
	It("Should convert a value that fits", func() {
		Expect(pb.Narrow[uint8](uint32(255))).To(BeEquivalentTo(255))
		Expect(pb.Narrow[int16](int32(-32768))).To(BeEquivalentTo(-32768))
	})

	It("Should reject a value above an unsigned maximum", func() {
		Expect(pb.Narrow[uint8](uint32(256))).Error().
			To(MatchError(validate.ErrValidation))
	})

	It("Should reject a value above a signed maximum", func() {
		Expect(pb.Narrow[int8](int32(128))).Error().
			To(MatchError(validate.ErrValidation))
	})

	It("Should reject a value below a signed minimum", func() {
		Expect(pb.Narrow[int16](int32(-32769))).Error().
			To(MatchError(validate.ErrValidation))
	})
})
