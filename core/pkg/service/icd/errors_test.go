// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("ErrStale", func() {
	It("Should round trip under its wire type", func(ctx SpecContext) {
		err := errors.Wrap(icd.ErrStale, "ICD Engine")
		pld := errors.Encode(ctx, err, false)
		Expect(pld.Type).To(Equal("sy.icd.stale"))
		Expect(pld.Data).To(Equal("ICD Engine: changed since it was read"))
		decoded := errors.Decode(ctx, pld)
		Expect(decoded).To(MatchError(icd.ErrStale))
		Expect(decoded).To(MatchError(ContainSubstring("ICD Engine")))
	})

	It("Should leave other errors to other decoders", func(ctx SpecContext) {
		decoded := errors.Decode(ctx, errors.Encode(ctx, errors.New("other"), false))
		Expect(decoded).ToNot(MatchError(icd.ErrStale))
	})
})
