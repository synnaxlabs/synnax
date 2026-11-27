// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package oidc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ErrorProvider", func() {
	codec := errors.NewCodec(oidc.ErrorProvider(), validate.ErrorProvider())
	DescribeTable(
		"Should round trip a link refusal ahead of the validation provider",
		func(pld errors.Payload) {
			err := codec.Decode(pld)
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(codec.Encode(err, false)).To(Equal(pld))
		},
		Entry("provider linked", errors.Payload{
			Type: "sy.validation.provider_linked",
			Data: "user already has an account of the provider linked",
		}),
		Entry("account linked", errors.Payload{
			Type: "sy.validation.account_linked",
			Data: "provider account is linked to another user",
		}),
	)
	It("Should leave other errors to the next provider", func() {
		Expect(codec.Encode(validate.ErrValidation, false).Type).
			To(Equal("sy.validation"))
	})
})
