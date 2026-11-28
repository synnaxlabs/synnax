// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("TimeFormat", func() {
	DescribeTable("Should refuse a timestamp field with no valid time format",
		func(format payload.TimeFormat) {
			f := xjson.Field{DataType: telem.TimestampT, TimeFormat: format}
			err := SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`unknown time format %q`, format)),
			)
			Expect(xjson.NewDecoder(f)).Error().To(err)
			Expect(xjson.NewEncoder(f)).Error().To(err)
		},
		Entry("an unknown format", payload.TimeFormat("rfc822")),
		Entry("no format", payload.TimeFormat("")),
	)

	It("Should need no time format for a field of another data type", func() {
		f := xjson.Field{DataType: telem.Int64T}
		Expect(xjson.NewDecoder(f)).Error().ToNot(HaveOccurred())
		Expect(xjson.NewEncoder(f)).Error().ToNot(HaveOccurred())
	})

	Describe("Encoder", func() {
		encode := func(ts telem.TimeStamp, format payload.TimeFormat) string {
			enc := MustSucceed(xjson.NewEncoder(
				xjson.Field{DataType: telem.TimestampT, TimeFormat: format},
			))
			return string(MustSucceed(enc.Append([]byte("x"), sample(ts))))
		}
		DescribeTable(
			"Should append every digit of a timestamp in ISO 8601 or seconds",
			func(ts telem.TimeStamp, format payload.TimeFormat, expected string) {
				Expect(encode(ts, format)).To(Equal("x" + expected))
			},
			Entry("ISO 8601 in UTC", telem.TimeStamp(1500000006),
				payload.TimeFormatISO8601, `"1970-01-01T00:00:01.500000006Z"`),
			Entry(
				"whole Unix seconds",
				2*telem.SecondTS,
				payload.TimeFormatUnixSec,
				"2",
			),
			Entry("Unix seconds with a fraction", telem.TimeStamp(1767323045123456789),
				payload.TimeFormatUnixSec, "1767323045.123456789"),
			Entry("a fraction with leading zeros", telem.TimeStamp(1000000005),
				payload.TimeFormatUnixSec, "1.000000005"),
			Entry("a fraction with trailing zeros", telem.TimeStamp(1500000000),
				payload.TimeFormatUnixSec, "1.5"),
			Entry("a negative fraction", telem.TimeStamp(-1500000000),
				payload.TimeFormatUnixSec, "-1.5"),
			Entry("a negative fraction before 1970", telem.TimeStamp(-1500),
				payload.TimeFormatUnixSec, "-0.0000015"),
			Entry("the smallest timestamp in Unix nanoseconds",
				telem.TimeStamp(math.MinInt64), payload.TimeFormatUnixNs,
				"-9223372036854775808"),
		)
		DescribeTable(
			"Should append a whole number of milliseconds or microseconds, rounded down",
			func(ts telem.TimeStamp, format payload.TimeFormat, expected string) {
				Expect(encode(ts, format)).To(Equal("x" + expected))
			},
			Entry("whole milliseconds", telem.TimeStamp(1500000000),
				payload.TimeFormatUnixMs, "1500"),
			Entry("a fraction of a millisecond", telem.TimeStamp(1999999),
				payload.TimeFormatUnixMs, "1"),
			Entry("a fraction of a microsecond", telem.TimeStamp(1500),
				payload.TimeFormatUnixUs, "1"),
			Entry("a millisecond before 1970", telem.TimeStamp(-1),
				payload.TimeFormatUnixMs, "-1"),
			Entry("a microsecond before 1970", telem.TimeStamp(-1500),
				payload.TimeFormatUnixUs, "-2"),
			Entry("the smallest timestamp in milliseconds",
				telem.TimeStamp(math.MinInt64), payload.TimeFormatUnixMs,
				"-9223372036855"),
		)
	})
})
