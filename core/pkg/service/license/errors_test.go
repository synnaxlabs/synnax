// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package license_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("Errors", func() {
	DescribeTable(
		"should round-trip a license error under its wire type",
		func(ctx SpecContext, err error, wireType string) {
			pld := errors.Encode(ctx, err, false)
			Expect(pld.Type).To(Equal(wireType))
			decoded := errors.Decode(ctx, pld)
			Expect(decoded).To(MatchError(err))
			Expect(decoded).To(MatchError(license.ErrLicense))
		},
		Entry("missing", license.ErrMissing, "sy.license.missing"),
		Entry("expired", license.ErrExpired, "sy.license.expired"),
		Entry("invalid", license.ErrInvalid, "sy.license.invalid"),
		Entry("fingerprint", license.ErrFingerprint, "sy.license.fingerprint"),
		Entry("too many", license.ErrTooMany, "sy.license.too_many"),
		Entry("base", license.ErrLicense, "sy.license"),
	)
	It(
		"should decode an unknown license subtype as a license error",
		func(ctx SpecContext) {
			decoded := errors.Decode(ctx, errors.Payload{
				Type: "sy.license.revoked",
				Data: "revoked",
			})
			Expect(decoded).To(MatchError(license.ErrLicense))
			Expect(decoded).ToNot(MatchError(license.ErrMissing))
		},
	)
	It("should defer other errors to other encoders", func(ctx SpecContext) {
		errTest := errors.New("test error")
		decoded := errors.Decode(ctx, errors.Encode(ctx, errTest, false))
		Expect(decoded).ToNot(MatchError(license.ErrLicense))
		Expect(decoded).To(MatchError("test error"))
	})
})
