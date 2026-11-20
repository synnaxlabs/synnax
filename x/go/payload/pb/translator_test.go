// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/payload/pb"
	. "github.com/synnaxlabs/x/testutil"
)

// roundTrip expects v to come back unchanged from to, then from.
func roundTrip[T, P any](to func(T) (P, error), from func(P) (T, error), v T) {
	GinkgoHelper()
	Expect(from(MustSucceed(to(v)))).To(Equal(v))
}

var _ = Describe("Translator", func() {
	DescribeTable("Should round trip a value", func(check func()) { check() },
		Entry("a float entry", func() {
			roundTrip(pb.FloatEntryToPB, pb.FloatEntryFromPB, payload.FloatEntry{
				Label: "HIGH", Value: 2.5,
			})
		}),
		Entry("an integer entry", func() {
			roundTrip(pb.IntegerEntryToPB, pb.IntegerEntryFromPB, payload.IntegerEntry{
				Label: "LOW", Value: -3,
			})
		}),
		Entry("an unsigned entry", func() {
			roundTrip(
				pb.UnsignedEntryToPB,
				pb.UnsignedEntryFromPB,
				payload.UnsignedEntry{Label: "MAX", Value: 1 << 63},
			)
		}),
		Entry("a string entry", func() {
			roundTrip(pb.StringEntryToPB, pb.StringEntryFromPB, payload.StringEntry{
				Label: "RUN", Value: "running",
			})
		}),
		Entry("a boolean entry", func() {
			roundTrip(pb.BooleanEntryToPB, pb.BooleanEntryFromPB, payload.BooleanEntry{
				Label: "ON", Value: true,
			})
		}),
	)

	DescribeTable("Should round trip a kind", func(v payload.Kind) {
		roundTrip(pb.KindToPB, pb.KindFromPB, v)
	},
		Entry("float", payload.KindFloat),
		Entry("integer", payload.KindInteger),
		Entry("unsigned", payload.KindUnsigned),
		Entry("string", payload.KindString),
		Entry("boolean", payload.KindBoolean),
		Entry("timestamp", payload.KindTimestamp),
		Entry("JSON", payload.KindJSON),
		Entry("UUID", payload.KindUUID),
		Entry("bytes", payload.KindBytes),
	)

	DescribeTable("Should round trip a time format", func(v payload.TimeFormat) {
		roundTrip(pb.TimeFormatToPB, pb.TimeFormatFromPB, v)
	},
		Entry("ISO 8601", payload.TimeFormatISO8601),
		Entry("Unix seconds", payload.TimeFormatUnixSec),
		Entry("Unix milliseconds", payload.TimeFormatUnixMs),
		Entry("Unix microseconds", payload.TimeFormatUnixUs),
		Entry("Unix nanoseconds", payload.TimeFormatUnixNs),
	)

	DescribeTable("Should reject a value that the enum does not hold",
		func(check func()) { check() },
		Entry("to protobuf", func() {
			Expect(pb.KindToPB("complex")).Error().
				To(MatchError("unrecognized payload.Kind value: complex"))
		}),
		Entry("from protobuf", func() {
			Expect(pb.KindFromPB(pb.Kind(99))).Error().
				To(MatchError("unrecognized Kind value: 99"))
		}),
	)
})
