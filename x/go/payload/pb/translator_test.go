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
	It("Should round trip the enum entry of each kind", func() {
		roundTrip(pb.FloatEntryToPB, pb.FloatEntryFromPB, payload.FloatEntry{
			Label: "HIGH", Value: 2.5,
		})
		roundTrip(pb.IntegerEntryToPB, pb.IntegerEntryFromPB, payload.IntegerEntry{
			Label: "LOW", Value: -3,
		})
		roundTrip(pb.UnsignedEntryToPB, pb.UnsignedEntryFromPB, payload.UnsignedEntry{
			Label: "MAX", Value: 1 << 63,
		})
		roundTrip(pb.StringEntryToPB, pb.StringEntryFromPB, payload.StringEntry{
			Label: "RUN", Value: "running",
		})
		roundTrip(pb.BooleanEntryToPB, pb.BooleanEntryFromPB, payload.BooleanEntry{
			Label: "ON", Value: true,
		})
		roundTrip(pb.EnumEntryToPB, pb.EnumEntryFromPB, payload.EnumEntry{
			Label: "FAULT", Value: 4,
		})
	})

	It("Should round trip the read of each kind", func() {
		roundTrip(pb.FloatReadToPB, pb.FloatReadFromPB, payload.FloatRead{
			DataType:   new(telem.Float32T),
			EnumValues: []payload.FloatEntry{{Label: "HIGH", Value: 2.5}},
			Fill:       new(payload.Float(1.5)),
		})
		roundTrip(pb.IntegerReadToPB, pb.IntegerReadFromPB, payload.IntegerRead{
			DataType:   new(telem.Int16T),
			EnumValues: []payload.IntegerEntry{{Label: "LOW", Value: -3}},
			Fill:       new(payload.Integer(-1)),
		})
		roundTrip(pb.UnsignedReadToPB, pb.UnsignedReadFromPB, payload.UnsignedRead{
			DataType:   new(telem.Uint8T),
			EnumValues: []payload.UnsignedEntry{{Label: "MAX", Value: 255}},
			Fill:       new(payload.Unsigned(1)),
		})
		roundTrip(pb.StringReadToPB, pb.StringReadFromPB, payload.StringRead{
			EnumValues: []payload.StringEntry{{Label: "RUN", Value: "running"}},
			Fill:       new("idle"),
		})
		roundTrip(pb.BooleanReadToPB, pb.BooleanReadFromPB, payload.BooleanRead{
			EnumValues: []payload.BooleanEntry{{Label: "ON", Value: true}},
			Fill:       new(false),
		})
		roundTrip(pb.TimestampReadToPB, pb.TimestampReadFromPB, payload.TimestampRead{
			TimeFormat: payload.TimeFormatUnixMs,
			Fill:       new(telem.TimeStamp(1000)),
		})
	})

	It("Should round trip a read with no data type or fill", func() {
		roundTrip(pb.FloatReadToPB, pb.FloatReadFromPB, payload.FloatRead{
			EnumValues: []payload.FloatEntry{},
		})
	})

	It("Should round trip each value of each enum", func() {
		for _, v := range []payload.Kind{
			payload.KindFloat, payload.KindInteger, payload.KindUnsigned,
			payload.KindString, payload.KindBoolean, payload.KindTimestamp,
			payload.KindJSON, payload.KindUUID, payload.KindBytes,
		} {
			roundTrip(pb.KindToPB, pb.KindFromPB, v)
		}
		for _, v := range []payload.TimeFormat{
			payload.TimeFormatISO8601, payload.TimeFormatUnixSec,
			payload.TimeFormatUnixMs, payload.TimeFormatUnixUs,
			payload.TimeFormatUnixNs,
		} {
			roundTrip(pb.TimeFormatToPB, pb.TimeFormatFromPB, v)
		}
		for _, v := range []payload.Format{
			payload.FormatJSON, payload.FormatText, payload.FormatBinary,
		} {
			roundTrip(pb.FormatToPB, pb.FormatFromPB, v)
		}
		for _, v := range []payload.JSONType{
			payload.JSONTypeNumber, payload.JSONTypeString, payload.JSONTypeBoolean,
		} {
			roundTrip(pb.JSONTypeToPB, pb.JSONTypeFromPB, v)
		}
		for _, v := range []payload.GeneratorType{
			payload.GeneratorTypeUUID, payload.GeneratorTypeTimestamp,
		} {
			roundTrip(pb.GeneratorTypeToPB, pb.GeneratorTypeFromPB, v)
		}
	})

	It("Should reject a value that the enum does not hold", func() {
		Expect(pb.KindToPB("complex")).Error().
			To(MatchError("unrecognized payload.Kind value: complex"))
		Expect(pb.KindFromPB(pb.Kind(99))).Error().
			To(MatchError("unrecognized Kind value: 99"))
	})
})
