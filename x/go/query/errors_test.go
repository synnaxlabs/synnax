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
		func(ctx SpecContext, base error, errType string) {
			encoded := errors.Encode(ctx, errors.Wrap(base, "channel 5"), false)
			Expect(encoded.Type).To(Equal(errType))
			decoded := errors.Decode(ctx, encoded)
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
})
