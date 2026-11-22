// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package query_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Errors", func() {
	DescribeTable(
		"Should decode an encoded error as the error it wraps",
		func(base error, errType string) {
			codec := errors.NewCodec(query.ErrorProvider())
			encoded := codec.Encode(errors.Wrap(base, "channel 5"), false)
			Expect(encoded.Type).To(Equal(errType))
			decoded := codec.Decode(encoded)
			Expect(decoded).To(MatchError(base))
			Expect(decoded).To(MatchError(ContainSubstring("channel 5")))
		},
		Entry("not found", query.ErrNotFound, "sy.query.not_found"),
		Entry(
			"unique violation",
			query.ErrUniqueViolation,
			"sy.query.unique_violation",
		),
		Entry(
			"invalid parameters",
			query.ErrInvalidParameters,
			"sy.query.invalid_parameters",
		),
		Entry("unsupported", query.ErrUnsupported, "sy.query.unsupported"),
		Entry(
			"multiple results",
			query.ErrMultipleResults,
			"sy.query.multiple_results",
		),
		Entry("query", query.ErrQuery, "sy.query"),
	)

	It("Should send a multiple results error as its text and keep its cause", func() {
		cause := errors.New("cause")
		err := query.MultipleResults(cause, "%d channels are named %q", 2, "pt")
		Expect(err).To(MatchError(`2 channels are named "pt"`))
		Expect(errors.Is(err, query.ErrMultipleResults)).To(BeTrue())
		Expect(errors.Is(err, query.ErrQuery)).To(BeTrue())
		Expect(errors.Is(err, cause)).To(BeTrue())
		encoded := errors.NewCodec(query.ErrorProvider()).Encode(err, false)
		Expect(encoded.Type).To(Equal("sy.query.multiple_results"))
		Expect(encoded.Data).To(Equal(`2 channels are named "pt"`))
	})
})
