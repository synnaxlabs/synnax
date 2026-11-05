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
			func(text string, format xjson.TimeFormat) {
				Expect(xjson.ParseTimeFormat(text)).To(Equal(format))
				Expect(format.String()).To(Equal(text))
			},
			Entry("ISO 8601", "iso8601", xjson.ISO8601),
			Entry("Unix seconds", "unix_sec", xjson.UnixSecond),
			Entry("Unix milliseconds", "unix_ms", xjson.UnixMillisecond),
			Entry("Unix microseconds", "unix_us", xjson.UnixMicrosecond),
			Entry("Unix nanoseconds", "unix_ns", xjson.UnixNanosecond),
		)
		It("Should reject an unknown format", func() {
			Expect(xjson.ParseTimeFormat("rfc822")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`unknown time format "rfc822"`)),
			))
		})
	})

	Describe("AppendTimeStamp", func() {
		DescribeTable("Should append every digit of a timestamp",
			func(ts telem.TimeStamp, format xjson.TimeFormat, expected string) {
				Expect(string(xjson.AppendTimeStamp([]byte("x"), ts, format))).
					To(Equal("x" + expected))
			},
			Entry("ISO 8601 in UTC", telem.TimeStamp(1500000006), xjson.ISO8601,
				`"1970-01-01T00:00:01.500000006Z"`),
			Entry("whole Unix seconds", 2*telem.SecondTS, xjson.UnixSecond, "2"),
			Entry("Unix seconds with a fraction", telem.TimeStamp(1767323045123456789),
				xjson.UnixSecond, "1767323045.123456789"),
			Entry("a fraction with leading zeros", telem.TimeStamp(1000000005),
				xjson.UnixSecond, "1.000000005"),
			Entry("a fraction with trailing zeros", telem.TimeStamp(1500000000),
				xjson.UnixMillisecond, "1500"),
			Entry("Unix microseconds", telem.TimeStamp(1500), xjson.UnixMicrosecond,
				"1.5"),
			Entry("a negative fraction", telem.TimeStamp(-1500000000),
				xjson.UnixSecond, "-1.5"),
			Entry("the smallest timestamp in Unix nanoseconds",
				telem.TimeStamp(math.MinInt64), xjson.UnixNanosecond,
				"-9223372036854775808"),
		)
	})
})

var _ = Describe("Marshal", func() {
	It("Should write object names in sorted order", func() {
		Expect(xjson.Marshal(map[string]any{"z": 1, "a": []any{true, "s"}})).
			To(Equal([]byte(`{"a":[true,"s"],"z":1}`)))
	})
	It("Should replace invalid UTF-8", func() {
		Expect(string(MustSucceed(xjson.Marshal("a\xffb")))).To(Equal("\"a\ufffdb\""))
	})
})
