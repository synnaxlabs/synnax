// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("Errors", func() {
	DescribeTable(
		"Correctly encodes/decodes a network portable freighter error",
		func(ctx SpecContext, err error) {
			pld := errors.Encode(ctx, err, false)
			Expect(errors.Decode(ctx, pld)).To(MatchError(err))
		},
		Entry("InvalidCredentials", auth.ErrInvalidCredentials),
		Entry("RepeatedUsername", auth.ErrRepeatedUsername),
		Entry("InvalidToken", auth.ErrInvalidToken),
		Entry("ExpiredToken", auth.ErrExpiredToken),
		Entry("AccessDenied", auth.ErrAccessDenied),
		Entry("Auth", auth.ErrAuth),
	)
	It("Should keep a denial distinct from a plain auth error", func(ctx SpecContext) {
		pld := errors.Encode(ctx, auth.ErrAccessDenied, false)
		Expect(errors.Is(errors.Decode(ctx, pld), auth.ErrAccessDenied)).To(BeTrue())
		pld = errors.Encode(ctx, auth.ErrAuth, false)
		Expect(errors.Is(errors.Decode(ctx, pld), auth.ErrAccessDenied)).To(BeFalse())
	})
	It("Should defer non-auth errors to other encoders", func(ctx SpecContext) {
		errTest := errors.New("test error")
		pld := errors.Encode(ctx, errTest, false)
		errDecoded := errors.Decode(ctx, pld)
		Expect(errors.Is(errDecoded, auth.ErrAuth)).To(BeFalse())
		Expect(errors.Is(errDecoded, errTest)).To(BeTrue())
	})
})
