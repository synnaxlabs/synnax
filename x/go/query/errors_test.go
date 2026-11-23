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
		Entry("query", query.ErrQuery, "sy.query"),
	)

	It("Should send a multiple results error as its text and keep its cause", func() {
		cause := errors.New("cause")
		err := query.MultipleResults(cause, "%d channels are named %q", 2, "pt")
		Expect(err).To(MatchError(`2 channels are named "pt"`))
		Expect(errors.Is(err, query.ErrQuery)).To(BeTrue())
		Expect(errors.Is(err, cause)).To(BeTrue())
		codec := errors.NewCodec(query.ErrorProvider())
		encoded := codec.Encode(err, false)
		Expect(encoded.Type).To(Equal("sy.query.multiple_results"))
		Expect(encoded.Data).To(Equal(`2 channels are named "pt"`))
		decoded := codec.Decode(encoded)
		Expect(decoded).To(MatchError(ContainSubstring(`2 channels are named "pt"`)))
		Expect(codec.Encode(decoded, false).Type).To(Equal("sy.query.multiple_results"))
	})

	DescribeTable(
		"Should not repeat the sentinel message on each hop",
		func(base error) {
			codec := errors.NewCodec(query.ErrorProvider())
			err := errors.Wrap(base, "channel 5")
			Expect(codec.Decode(
				codec.Encode(codec.Decode(codec.Encode(err, false)), false),
			)).To(MatchError(err.Error()))
			pld := codec.Encode(base, false)
			Expect(pld.Data).To(Equal(base.Error()))
			Expect(codec.Decode(pld)).To(Equal(base))
		},
		Entry("not found", query.ErrNotFound),
		Entry("unique violation", query.ErrUniqueViolation),
		Entry("invalid parameters", query.ErrInvalidParameters),
		Entry("unsupported", query.ErrUnsupported),
		Entry("query", query.ErrQuery),
	)
})
