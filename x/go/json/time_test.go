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
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("TimeFormat", func() {
	Describe("ParseTimeFormat", func() {
		DescribeTable("Should round trip the config form",
			func(text string) {
				Expect(
					MustSucceed(xjson.ParseTimeFormat(text)).String(),
				).To(Equal(text))
			},
			Entry("ISO 8601", "iso8601"),
			Entry("Unix seconds", "unix_sec"),
			Entry("Unix milliseconds", "unix_ms"),
			Entry("Unix microseconds", "unix_us"),
			Entry("Unix nanoseconds", "unix_ns"),
		)
		It("Should reject an unknown format", func() {
			Expect(xjson.ParseTimeFormat("rfc822")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`unknown time format "rfc822"`)),
			))
		})
	})

	Describe("AppendTimeStamp", func() {
		DescribeTable("Should append every digit of a timestamp in ISO 8601 or seconds",
			func(ts telem.TimeStamp, format, expected string) {
				f := MustSucceed(xjson.ParseTimeFormat(format))
				Expect(string(xjson.AppendTimeStamp([]byte("x"), ts, f))).
					To(Equal("x" + expected))
			},
			Entry("ISO 8601 in UTC", telem.TimeStamp(1500000006), "iso8601",
				`"1970-01-01T00:00:01.500000006Z"`),
			Entry("whole Unix seconds", 2*telem.SecondTS, "unix_sec", "2"),
			Entry("Unix seconds with a fraction", telem.TimeStamp(1767323045123456789),
				"unix_sec", "1767323045.123456789"),
			Entry("a fraction with leading zeros", telem.TimeStamp(1000000005),
				"unix_sec", "1.000000005"),
			Entry("a fraction with trailing zeros", telem.TimeStamp(1500000000),
				"unix_sec", "1.5"),
			Entry("a negative fraction", telem.TimeStamp(-1500000000),
				"unix_sec", "-1.5"),
			Entry("a negative fraction before 1970", telem.TimeStamp(-1500),
				"unix_sec", "-0.0000015"),
			Entry("the smallest timestamp in Unix nanoseconds",
				telem.TimeStamp(math.MinInt64), "unix_ns",
				"-9223372036854775808"),
		)
		DescribeTable(
			"Should append a whole number of milliseconds or microseconds, rounded down",
			func(ts telem.TimeStamp, format, expected string) {
				f := MustSucceed(xjson.ParseTimeFormat(format))
				Expect(string(xjson.AppendTimeStamp([]byte("x"), ts, f))).
					To(Equal("x" + expected))
			},
			Entry("whole milliseconds", telem.TimeStamp(1500000000),
				"unix_ms", "1500"),
			Entry("a fraction of a millisecond", telem.TimeStamp(1999999),
				"unix_ms", "1"),
			Entry("a fraction of a microsecond", telem.TimeStamp(1500),
				"unix_us", "1"),
			Entry("a millisecond before 1970", telem.TimeStamp(-1),
				"unix_ms", "-1"),
			Entry("a microsecond before 1970", telem.TimeStamp(-1500),
				"unix_us", "-2"),
			Entry("the smallest timestamp in milliseconds",
				telem.TimeStamp(math.MinInt64), "unix_ms",
				"-9223372036855"),
		)
	})
})
