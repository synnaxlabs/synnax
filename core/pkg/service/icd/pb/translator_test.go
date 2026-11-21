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
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/pb"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func jsonBase(name string) icd.JSONBaseField {
	return icd.JSONBaseField{Key: uuid.New(), Name: name, Scale: 1, Pointer: "/" + name}
}

func jsonMessage(match icd.JSONMatchVariant) icd.ICD {
	return icd.ICD{
		Key:  uuid.New(),
		Name: "Rig",
		Messages: []icd.Message{{
			Key:  uuid.New(),
			Name: "Status",
			Payload: icd.Payload{Variant: icd.JSONPayload{
				Match: &icd.JSONMatch{Variant: match},
				Fields: []icd.JSONField{{Variant: icd.JSONJSONField{
					JSONBaseField: jsonBase("raw"),
				}}},
			}},
		}},
	}
}

var _ = Describe("Translator", func() {
	It("Should round trip an ICD with every payload and field variant", func() {
		mode := uuid.New()
		ic := icd.ICD{
			Key:  uuid.New(),
			Name: "Rig",
			Messages: []icd.Message{
				{
					Key:       uuid.New(),
					Name:      "Engine",
					CycleTime: new(100 * telem.Millisecond),
					Payload: icd.Payload{Variant: icd.BinaryPayload{
						Length: new(uint16(8)),
						Identifier: &icd.Identifier{Variant: icd.CanIdentifier{
							ID:              0x100,
							Extended:        true,
							Fd:              true,
							BitrateSwitched: true,
							Mask:            new(uint32(0x7FF)),
						}},
						Fields: []icd.BinaryField{
							{
								Key:       mode,
								Name:      "mode",
								Scale:     1,
								BitLength: 4,
								ByteOrder: icd.ByteOrderLittleEndian,
								Kind:      payload.KindUnsigned,
							},
							{
								Key:             uuid.New(),
								Name:            "rpm",
								Scale:           0.25,
								Offset:          -40,
								Units:           "rpm",
								Min:             new(0.0),
								Max:             new(8000.0),
								StartBit:        15,
								BitLength:       16,
								ByteOrder:       icd.ByteOrderBigEndian,
								Kind:            payload.KindInteger,
								Multiplexor:     &mode,
								MultiplexValues: []int32{1, 3},
							},
						},
					}},
				},
				{
					Key:  uuid.New(),
					Name: "Line",
					Payload: icd.Payload{Variant: icd.TextPayload{
						Delimiter: ";",
						Prefix:    "$GP",
						Fields: []icd.TextField{
							{Variant: icd.DelimitedTextField{
								Key:      uuid.New(),
								Name:     "speed",
								Scale:    1,
								Units:    "m/s",
								Position: 2,
							}},
							{Variant: icd.TaggedTextField{
								Key:   uuid.New(),
								Name:  "heading",
								Scale: 1,
								Tag:   "HDG=",
							}},
						},
					}},
				},
				{
					Key:  uuid.New(),
					Name: "Status",
					Payload: icd.Payload{Variant: icd.JSONPayload{
						Match: &icd.JSONMatch{Variant: icd.StringJSONMatch{
							JSONBaseMatch: icd.JSONBaseMatch{Pointer: "/kind"},
							Value:         "status",
						}},
						Fields: []icd.JSONField{
							{Variant: icd.FloatJSONField{
								JSONBaseField: jsonBase("float"),
								EnumValues: []payload.FloatEntry{
									{Label: "HIGH", Value: payload.Float(math.Inf(1))},
								},
								Default: new(payload.Float(1.5)),
							}},
							{Variant: icd.IntegerJSONField{
								JSONBaseField: jsonBase("integer"),
								EnumValues: []payload.IntegerEntry{
									{Label: "MIN", Value: math.MinInt64},
								},
								Default: new(payload.Integer(-3)),
							}},
							{Variant: icd.UnsignedJSONField{
								JSONBaseField: jsonBase("unsigned"),
								EnumValues: []payload.UnsignedEntry{
									{Label: "MAX", Value: math.MaxUint64},
								},
								Default: new(payload.Unsigned(7)),
							}},
							{Variant: icd.StringJSONField{
								JSONBaseField: jsonBase("string"),
								EnumValues: []payload.StringEntry{
									{Label: "on", Value: "ON"},
								},
								Default: new("OFF"),
							}},
							{Variant: icd.BooleanJSONField{
								JSONBaseField: jsonBase("boolean"),
								EnumValues: []payload.BooleanEntry{
									{Label: "yes", Value: true},
								},
								Default: new(false),
							}},
							{Variant: icd.TimestampJSONField{
								JSONBaseField: jsonBase("timestamp"),
								TimeFormat:    payload.TimeFormatUnixMs,
								Default: new(
									telem.TimeStamp(1700000000000000000),
								),
							}},
							{
								Variant: icd.JSONJSONField{
									JSONBaseField: jsonBase("json"),
								},
							},
							{
								Variant: icd.UuidJSONField{
									JSONBaseField: jsonBase("uuid"),
								},
							},
							{
								Variant: icd.BytesJSONField{
									JSONBaseField: jsonBase("bytes"),
								},
							},
						},
					}},
				},
			},
		}
		translated := MustSucceed(pb.ICDToPB(ic))
		Expect(pb.ICDFromPB(translated)).To(Equal(ic))
	})

	DescribeTable("Should round trip a JSON match", func(match icd.JSONMatchVariant) {
		ic := jsonMessage(match)
		translated := MustSucceed(pb.ICDToPB(ic))
		Expect(pb.ICDFromPB(translated)).To(Equal(ic))
	},
		Entry("of a float", icd.FloatJSONMatch{Value: payload.Float(2.5)}),
		Entry("of an integer", icd.IntegerJSONMatch{Value: -1}),
		Entry("of an unsigned integer", icd.UnsignedJSONMatch{Value: 1}),
		Entry("of a boolean", icd.BooleanJSONMatch{Value: true}),
	)

	It("Should round trip a field identifier", func() {
		id := icd.Identifier{Variant: icd.FieldIdentifier{Field: uuid.New(), Value: -2}}
		translated := MustSucceed(pb.IdentifierToPB(id))
		Expect(pb.IdentifierFromPB(translated)).To(Equal(id))
	})

	DescribeTable(
		"Should round trip a filter",
		func(w icd.Where) {
			translated := MustSucceed(pb.WhereToPB(w))
			Expect(pb.WhereFromPB(translated)).To(Equal(w))
		},
		Entry("with no clauses", icd.Where{}),
		Entry("of one key", icd.Where{Key: &filter.Set[icd.Key]{Eq: new(uuid.New())}}),
		Entry("of several keys", icd.Where{
			Key: &filter.Set[icd.Key]{In: []icd.Key{uuid.New(), uuid.New()}},
		}),
		Entry("of no keys", icd.Where{Key: &filter.Set[icd.Key]{In: []icd.Key{}}}),
		Entry("of nested clauses", icd.Where{
			And: []icd.Where{
				{Key: &filter.Set[icd.Key]{Eq: new(uuid.New())}},
				{Not: &icd.Where{Key: &filter.Set[icd.Key]{In: []icd.Key{uuid.New()}}}},
			},
			Or: []icd.Where{{Key: &filter.Set[icd.Key]{Eq: new(uuid.New())}}},
		}),
		Entry(
			"of empty clause lists",
			icd.Where{And: []icd.Where{}, Or: []icd.Where{}},
		),
	)

	It("Should keep an absent key set apart from an empty one", func() {
		absent := MustSucceed(pb.UuidSetToPB(filter.Set[icd.Key]{}))
		Expect(absent.In).To(BeNil())
		Expect(MustSucceed(pb.UuidSetFromPB(absent)).In).To(BeNil())
		empty := MustSucceed(pb.UuidSetToPB(filter.Set[icd.Key]{In: []icd.Key{}}))
		Expect(empty.In).ToNot(BeNil())
		Expect(MustSucceed(pb.UuidSetFromPB(empty)).In).To(BeEmpty())
		Expect(MustSucceed(pb.UuidSetFromPB(empty)).In).ToNot(BeNil())
	})

	DescribeTable(
		"Should reject a key that is not a UUID",
		func(check func()) { check() },
		Entry("in a key set", func() {
			Expect(pb.UuidSetFromPB(&pb.UuidSet{
				In: &pb.UuidList{Values: []string{"rig"}},
			})).Error().To(MatchError("invalid uuid"))
		}),
		Entry("in a filter", func() {
			Expect(pb.WhereFromPB(&pb.Where{Key: &pb.UuidSet{Eq: new("rig")}})).
				Error().To(MatchError("invalid uuid"))
		}),
	)

	It("Should reject a message key that is not a UUID", func() {
		Expect(pb.MessageFromPB(&pb.Message{Key: "engine"})).Error().
			To(MatchError("invalid uuid"))
	})
})
