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
	. "github.com/synnaxlabs/x/testutil"

	"github.com/synnaxlabs/x/pb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var _ = Describe("Any", func() {
	Describe("AnyToPBAny", func() {
		It("Should return nil for nil input", func() {
			result, err := pb.AnyToPBAny(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
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

		It("Should convert a map[string]any to structpb.Struct", func() {
			m := map[string]any{"key": "value", "count": float64(42)}
			result := MustSucceed(pb.AnyToPBAny(m))
			Expect(result.TypeUrl).To(ContainSubstring("Struct"))
		})

		It("Should convert an arbitrary struct via JSON marshaling", func() {
			type custom struct {
				Name  string `json:"name"`
				Value int    `json:"value"`
			}
			result := MustSucceed(pb.AnyToPBAny(custom{Name: "test", Value: 5}))
			Expect(result.TypeUrl).To(ContainSubstring("Struct"))
		})
	})

	Describe("AnyFromPBAny", func() {
		It("Should return nil for nil input", func() {
			result, err := pb.AnyFromPBAny(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
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

		It("Should return nil for an unregistered type URL", func() {
			packed := &anypb.Any{
				TypeUrl: "type.googleapis.com/unknown.Type",
				Value:   []byte{1, 2, 3},
			}
			result, err := pb.AnyFromPBAny(packed)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
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
			result := MustSucceed(pb.AnyFromPBAny(packed))
			m, ok := result.(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(m["name"]).To(Equal("test"))
			Expect(m["count"]).To(BeNumerically("==", 10))
			nested, ok := m["nested"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(nested["inner"]).To(Equal("data"))
		})
	})
})
