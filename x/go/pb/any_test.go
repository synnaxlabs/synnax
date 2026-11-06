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
	"github.com/synnaxlabs/x/pb"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/typepb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type details struct {
	ConfigHash string  `json:"config_hash"`
	Running    bool    `json:"running"`
	Rate       float64 `json:"rate,omitzero"`
}

var _ = Describe("Any", func() {
	Describe("NilToPBAny", func() {
		It("Should return nil", func() {
			Expect(pb.NilToPBAny(struct{}{})).To(BeNil())
		})
	})

	Describe("NilFromPBAny", func() {
		It("Should return the empty value for any input", func() {
			a := MustSucceed(anypb.New(wrapperspb.String("hello")))
			Expect(pb.NilFromPBAny(a)).To(Equal(struct{}{}))
			Expect(pb.NilFromPBAny(nil)).To(Equal(struct{}{}))
		})
	})

	Describe("AnyToPBAny", func() {
		It("Should return nil for nil input", func() {
			Expect(pb.AnyToPBAny(nil)).To(BeNil())
		})

		It("Should pass through an existing *anypb.Any", func() {
			original := MustSucceed(anypb.New(wrapperspb.String("hello")))
			result := MustSucceed(pb.AnyToPBAny(original))
			Expect(result).To(Equal(original))
		})

		It("Should pack a proto.Message directly", func() {
			msg := wrapperspb.String("hello")
			result := MustSucceed(pb.AnyToPBAny(msg))
			Expect(result.TypeUrl).To(ContainSubstring("StringValue"))
		})

		It("Should convert a map[string]any to structpb.Value", func() {
			m := map[string]any{"key": "value", "count": float64(42)}
			result := MustSucceed(pb.AnyToPBAny(m))
			Expect(result.TypeUrl).To(ContainSubstring("Value"))
		})

		It("Should convert an arbitrary struct via JSON marshaling", func() {
			type custom struct {
				Name  string `json:"name"`
				Value int    `json:"value"`
			}
			result := MustSucceed(pb.AnyToPBAny(custom{Name: "test", Value: 5}))
			Expect(result.TypeUrl).To(ContainSubstring("Value"))
		})

		DescribeTable("Should round-trip every JSON type",
			func(original any) {
				packed := MustSucceed(pb.AnyToPBAny(original))
				Expect(pb.AnyFromPBAny(packed)).To(Equal(original))
			},
			Entry("number", float64(42)),
			Entry("string", "sensor"),
			Entry("boolean", true),
			Entry("array", []any{float64(1), "two"}),
			Entry("object", map[string]any{"key": "value"}),
		)
	})

	Describe("AnyFromPBAny", func() {
		It("Should return nil for nil input", func() {
			Expect(pb.AnyFromPBAny(nil)).To(BeNil())
		})

		It("Should unpack a null structpb.Value to nil", func() {
			packed := MustSucceed(anypb.New(structpb.NewNullValue()))
			Expect(pb.AnyFromPBAny(packed)).To(BeNil())
		})

		It("Should unpack a structpb.Struct to a map", func() {
			s := MustSucceed(structpb.NewStruct(map[string]any{
				"key": "value",
				"num": float64(42),
			}))
			packed := MustSucceed(anypb.New(s))
			result := MustSucceed(pb.AnyFromPBAny(packed))
			m, ok := result.(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(m["key"]).To(Equal("value"))
			Expect(m["num"]).To(BeNumerically("==", 42))
		})

		It("Should key a typed proto by the field names of its schema", func() {
			packed := MustSucceed(anypb.New(&typepb.Field{JsonName: "hash"}))
			Expect(pb.AnyFromPBAny(packed)).To(And(
				HaveKeyWithValue("json_name", "hash"),
				Not(HaveKey("jsonName")),
			))
		})

		It("Should return nil for an unregistered type URL", func() {
			packed := &anypb.Any{
				TypeUrl: "type.googleapis.com/unknown.Type",
				Value:   []byte{1, 2, 3},
			}
			Expect(pb.AnyFromPBAny(packed)).To(BeNil())
		})
	})

	Describe("ToAny", func() {
		It("Should key the JSON value by the json tags of the type", func() {
			packed := MustSucceed(pb.ToAny(details{ConfigHash: "abc", Running: true}))
			var v structpb.Value
			Expect(packed.UnmarshalTo(&v)).To(Succeed())
			Expect(v.AsInterface()).To(Equal(map[string]any{
				"config_hash": "abc",
				"running":     true,
			}))
		})
	})

	Describe("FromAny", func() {
		It("Should return the zero value for nil input", func() {
			Expect(pb.FromAny[details](nil)).To(Equal(details{}))
		})

		It("Should decode a structpb.Struct from an older peer", func() {
			s := MustSucceed(structpb.NewStruct(map[string]any{
				"config_hash": "abc",
				"rate":        2.5,
			}))
			packed := MustSucceed(anypb.New(s))
			Expect(pb.FromAny[details](packed)).
				To(Equal(details{ConfigHash: "abc", Rate: 2.5}))
		})

		It("Should return an error for an Any that holds no JSON", func() {
			packed := MustSucceed(anypb.New(wrapperspb.String("abc")))
			Expect(pb.FromAny[details](packed)).Error().To(MatchError(ContainSubstring(
				"failed to unpack Any of type type.googleapis.com/google.protobuf.StringValue",
			)))
		})

		It("Should return the zero value when a later field fails to decode", func() {
			v := MustSucceed(structpb.NewValue(map[string]any{
				"config_hash": "abc",
				"running":     "yes",
			}))
			packed := MustSucceed(anypb.New(v))
			res, err := pb.FromAny[details](packed)
			Expect(err).To(MatchError(ContainSubstring(
				`cannot unmarshal JSON string into Go bool within "/running"`,
			)))
			Expect(res).To(Equal(details{}))
		})

		It("Should round-trip a value through ToAny", func() {
			original := details{ConfigHash: "abc", Running: true, Rate: 2.5}
			packed := MustSucceed(pb.ToAny(original))
			Expect(pb.FromAny[details](packed)).To(Equal(original))
		})
	})

	Describe("Round Trip", func() {
		It("Should round-trip a map through ToPBAny and FromPBAny", func() {
			original := map[string]any{
				"name":   "test",
				"count":  float64(10),
				"nested": map[string]any{"inner": "data"},
			}
			packed := MustSucceed(pb.AnyToPBAny(original))
			m, ok := MustSucceed(pb.AnyFromPBAny(packed)).(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(m["name"]).To(Equal("test"))
			Expect(m["count"]).To(BeNumerically("==", 10))
			nested, ok := m["nested"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(nested["inner"]).To(Equal("data"))
		})
	})
})
