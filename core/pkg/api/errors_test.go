// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api_test

import (
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("NewErrorCodec", func() {
	codec := api.NewErrorCodec()
	DescribeTable("Should encode the errors of every provider",
		func(err error, errType string) {
			pld := codec.Encode(err, false)
			Expect(pld.Type).To(Equal(errType))
			Expect(codec.Decode(pld)).To(MatchError(err))
		},
		Entry("freighter", freighter.ErrStreamClosed, "freighter.stream_closed"),
		Entry("query", query.ErrNotFound, "sy.query.not_found"),
		Entry("validate", validate.ErrValidation, "sy.validation"),
		Entry("control", control.ErrUnauthorized, "sy.control.unauthorized"),
		Entry("auth", auth.ErrInvalidToken, "sy.auth.invalid_token"),
		Entry("license", license.ErrMissing, "sy.license.missing"),
	)
	It("Should encode an Arc compile error", func() {
		Expect(codec.Encode(arc.CompileError{}, false).Type).To(Equal("arc"))
	})
	It("Should report a provider's errors as expected", func() {
		Expect(codec.Expected(io.EOF)).To(BeTrue())
		Expect(codec.Expected(errors.New("cat"))).To(BeFalse())
	})
})
