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
		"Should round trip a link error ahead of the validation provider",
		func(err error, errType, data string) {
			pld := codec.Encode(err, false)
			Expect(pld).To(Equal(errors.Payload{Type: errType, Data: data}))
			decoded := codec.Decode(pld)
			Expect(decoded).To(MatchError(err))
			Expect(decoded).To(MatchError(validate.ErrValidation))
		},
		Entry(
			"provider linked",
			oidc.ErrProviderLinked,
			"sy.validation.provider_linked",
			"user already has an account of the provider linked",
		),
		Entry(
			"account linked",
			oidc.ErrAccountLinked,
			"sy.validation.account_linked",
			"provider account is linked to another user",
		),
	)
	It("Should leave other errors to the next provider", func() {
		Expect(codec.Encode(validate.ErrValidation, false).Type).
			To(Equal("sy.validation"))
	})
})
