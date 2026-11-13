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
	"github.com/synnaxlabs/x/telem"
)

func convert[To, From payload.Number](v From) (any, bool) {
	return payload.Convert[To](v)
}

var _ = Describe("Number", func() {
	DescribeTable("Convert",
		func(converted func() (any, bool), expected any, ok bool) {
			v, gotOk := converted()
			Expect(gotOk).To(Equal(ok))
			if ok {
				Expect(v).To(Equal(expected))
			}
		},
		Entry("a whole float to an integer",
			func() (any, bool) { return convert[payload.Integer](payload.Float(-3)) },
			payload.Integer(-3), true),
		Entry("a float with a fraction to an integer",
			func() (any, bool) { return convert[payload.Integer](payload.Float(0.5)) },
			nil, false),
		Entry("NaN to an integer",
			func() (any, bool) {
				return convert[payload.Integer](payload.Float(math.NaN()))
			},
			nil, false),
		Entry(
			"2^63 to an integer",
			func() (any, bool) { return convert[payload.Integer](payload.Float(1 << 63)) },
			nil,
			false,
		),
		Entry("2^63 to an unsigned",
			func() (any, bool) {
				return convert[payload.Unsigned](payload.Float(1 << 63))
			},
			payload.Unsigned(1<<63), true),
		Entry("a negative float to an unsigned",
			func() (any, bool) { return convert[payload.Unsigned](payload.Float(-1)) },
			nil, false),
		Entry("a float to a float",
			func() (any, bool) { return convert[payload.Float](payload.Float(0.5)) },
			payload.Float(0.5), true),
		Entry("an integer to a float",
			func() (any, bool) { return convert[payload.Float](payload.Integer(-2)) },
			payload.Float(-2), true),
		Entry(
			"a negative integer to an unsigned",
			func() (any, bool) { return convert[payload.Unsigned](payload.Integer(-2)) },
			nil,
			false,
		),
		Entry("an integer to an integer",
			func() (any, bool) { return convert[payload.Integer](payload.Integer(-2)) },
			payload.Integer(-2), true),
		Entry("an unsigned past int64 to an integer",
			func() (any, bool) {
				return convert[payload.Integer](payload.Unsigned(math.MaxUint64))
			},
			nil, false),
		Entry("an unsigned to a float",
			func() (any, bool) { return convert[payload.Float](payload.Unsigned(7)) },
			payload.Float(7), true),
		Entry("an integer past 2^53 with an equal float to a float",
			func() (any, bool) {
				return convert[payload.Float](payload.Integer(1 << 62))
			},
			payload.Float(1<<62), true),
		Entry("the minimum integer to a float",
			func() (any, bool) {
				return convert[payload.Float](payload.Integer(math.MinInt64))
			},
			payload.Float(math.MinInt64), true),
		Entry("an integer past 2^53 with no equal float to a float",
			func() (any, bool) {
				return convert[payload.Float](payload.Integer(1<<53 + 1))
			},
			nil, false),
		Entry("the maximum integer to a float",
			func() (any, bool) {
				return convert[payload.Float](payload.Integer(math.MaxInt64))
			},
			nil, false),
		Entry("an unsigned past 2^53 with no equal float to a float",
			func() (any, bool) {
				return convert[payload.Float](payload.Unsigned(1<<53 + 1))
			},
			nil, false),
		Entry("the maximum unsigned to a float",
			func() (any, bool) {
				return convert[payload.Float](payload.Unsigned(math.MaxUint64))
			},
			nil, false),
	)

	DescribeTable("Fits",
		func(fits, expected bool) { Expect(fits).To(Equal(expected)) },
		Entry("a float past float32",
			payload.Fits(payload.Float(1e39), telem.Float32T), false),
		Entry("infinity in float32",
			payload.Fits(payload.Float(math.Inf(1)), telem.Float32T), true),
		Entry("a float in float64",
			payload.Fits(payload.Float(1e300), telem.Float64T), true),
		Entry("a float in an integer channel",
			payload.Fits(payload.Float(1), telem.Int8T), false),
		Entry("the int8 minimum",
			payload.Fits(payload.Integer(math.MinInt8), telem.Int8T), true),
		Entry("past the int8 maximum",
			payload.Fits(payload.Integer(math.MaxInt8+1), telem.Int8T), false),
		Entry("past the int32 minimum",
			payload.Fits(payload.Integer(math.MinInt32-1), telem.Int32T), false),
		Entry("the uint16 maximum",
			payload.Fits(payload.Unsigned(math.MaxUint16), telem.Uint16T), true),
		Entry("past the uint32 maximum",
			payload.Fits(payload.Unsigned(math.MaxUint32+1), telem.Uint32T), false),
		Entry("an unsigned in an integer channel",
			payload.Fits(payload.Unsigned(1), telem.Int64T), false),
	)
})
