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
	"github.com/synnaxlabs/x/telem"
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
		Entry("an enum entry", func() {
			roundTrip(pb.EnumEntryToPB, pb.EnumEntryFromPB, payload.EnumEntry{
				Label: "FAULT", Value: 4,
			})
		}),
		Entry("a float read", func() {
			roundTrip(pb.FloatReadToPB, pb.FloatReadFromPB, payload.FloatRead{
				DataType:   new(telem.Float32T),
				EnumValues: []payload.FloatEntry{{Label: "HIGH", Value: 2.5}},
				Fill:       new(payload.Float(1.5)),
			})
		}),
		Entry("a float read with no data type or fill", func() {
			roundTrip(pb.FloatReadToPB, pb.FloatReadFromPB, payload.FloatRead{
				EnumValues: []payload.FloatEntry{},
			})
		}),
		Entry("an integer read", func() {
			roundTrip(pb.IntegerReadToPB, pb.IntegerReadFromPB, payload.IntegerRead{
				DataType:   new(telem.Int16T),
				EnumValues: []payload.IntegerEntry{{Label: "LOW", Value: -3}},
				Fill:       new(payload.Integer(-1)),
			})
		}),
		Entry("an unsigned read", func() {
			roundTrip(pb.UnsignedReadToPB, pb.UnsignedReadFromPB, payload.UnsignedRead{
				DataType:   new(telem.Uint8T),
				EnumValues: []payload.UnsignedEntry{{Label: "MAX", Value: 255}},
				Fill:       new(payload.Unsigned(1)),
			})
		}),
		Entry("a string read", func() {
			roundTrip(pb.StringReadToPB, pb.StringReadFromPB, payload.StringRead{
				EnumValues: []payload.StringEntry{{Label: "RUN", Value: "running"}},
				Fill:       new("idle"),
			})
		}),
		Entry("a boolean read", func() {
			roundTrip(pb.BooleanReadToPB, pb.BooleanReadFromPB, payload.BooleanRead{
				EnumValues: []payload.BooleanEntry{{Label: "ON", Value: true}},
				Fill:       new(false),
			})
		}),
		Entry("a timestamp read", func() {
			roundTrip(
				pb.TimestampReadToPB,
				pb.TimestampReadFromPB,
				payload.TimestampRead{
					TimeFormat: payload.TimeFormatUnixMs,
					Fill:       new(telem.TimeStamp(1000)),
				},
			)
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

	DescribeTable("Should round trip a format", func(v payload.Format) {
		roundTrip(pb.FormatToPB, pb.FormatFromPB, v)
	},
		Entry("JSON", payload.FormatJSON),
		Entry("text", payload.FormatText),
		Entry("binary", payload.FormatBinary),
	)

	DescribeTable("Should round trip a JSON type", func(v payload.JSONType) {
		roundTrip(pb.JSONTypeToPB, pb.JSONTypeFromPB, v)
	},
		Entry("number", payload.JSONTypeNumber),
		Entry("string", payload.JSONTypeString),
		Entry("boolean", payload.JSONTypeBoolean),
	)

	DescribeTable("Should round trip a generator type", func(v payload.GeneratorType) {
		roundTrip(pb.GeneratorTypeToPB, pb.GeneratorTypeFromPB, v)
	},
		Entry("UUID", payload.GeneratorTypeUUID),
		Entry("timestamp", payload.GeneratorTypeTimestamp),
	)

	DescribeTable("Should reject a value that the enum does not hold",
		func(translate func() error, message string) {
			Expect(translate()).To(MatchError(message))
		},
		Entry("to protobuf", func() error {
			_, err := pb.KindToPB("complex")
			return err
		}, "unrecognized payload.Kind value: complex"),
		Entry("from protobuf", func() error {
			_, err := pb.KindFromPB(pb.Kind(99))
			return err
		}, "unrecognized Kind value: 99"),
	)
})
