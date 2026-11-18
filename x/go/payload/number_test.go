// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/payload"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Number", func() {
	integer := func(v int64) payload.Number {
		r := payload.FromInteger(payload.IntegerRead{Fill: new(payload.Integer(v))})
		return *r.Number.Fill
	}
	unsigned := func(v uint64) payload.Number {
		r := payload.FromUnsigned(payload.UnsignedRead{Fill: new(payload.Unsigned(v))})
		return *r.Number.Fill
	}
	nan := payload.FloatNumber(payload.Float(math.NaN()))

	DescribeTable(
		"should convert a number that the kind holds exactly",
		func(n payload.Number, k payload.Kind, expected payload.Number) {
			Expect(MustBeOk(n.To(k))).To(Equal(expected))
		},
		Entry("a whole float to an integer",
			payload.FloatNumber(-3), payload.KindInteger, integer(-3)),
		Entry("a whole float to an unsigned integer",
			payload.FloatNumber(7), payload.KindUnsigned, unsigned(7)),
		Entry("an integer to a float",
			integer(1<<53), payload.KindFloat, payload.FloatNumber(1<<53)),
		Entry("an unsigned integer to an integer",
			unsigned(math.MaxInt64), payload.KindInteger, integer(math.MaxInt64)),
	)

	It("Should convert a NaN to a float", func() {
		Expect(MustBeOk(nan.To(payload.KindFloat)).String()).To(Equal("NaN"))
	})

	DescribeTable(
		"should not convert a number that the kind does not hold exactly",
		func(n payload.Number, k payload.Kind) {
			_, ok := n.To(k)
			Expect(ok).To(BeFalse())
		},
		Entry(
			"a fraction to an integer",
			payload.FloatNumber(1.5),
			payload.KindInteger,
		),
		Entry("a negative to an unsigned integer", integer(-1), payload.KindUnsigned),
		Entry("a NaN to an integer", nan, payload.KindInteger),
		Entry("an infinity to an unsigned integer",
			payload.FloatNumber(payload.Float(math.Inf(1))), payload.KindUnsigned),
		Entry("an integer past 2^53 to a float", integer(1<<53+1), payload.KindFloat),
		Entry("an unsigned integer past int64 to an integer",
			unsigned(math.MaxUint64), payload.KindInteger),
		Entry("a number to a string", payload.FloatNumber(1), payload.KindString),
	)
})
