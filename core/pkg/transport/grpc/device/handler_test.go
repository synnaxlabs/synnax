// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/device"
	svcdevice "github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("gRPC Device Translators", func() {
	Describe("RetrieveRequest", func() {
		t := retrieveRequestTranslator{}
		It("Should round-trip a filter tree", func(ctx SpecContext) {
			msg := device.RetrieveRequest{
				Where: &svcdevice.Where{
					Or: []svcdevice.Where{
						{Key: &filter.Set[string]{In: []string{"dev1"}}},
						{Make: &filter.String{Eq: new("NI")}},
					},
					Not: &svcdevice.Where{
						Rack: &filter.Set[rack.Key]{In: []rack.Key{1}},
					},
					Name: &filter.String{Regex: new("^PT")},
				},
				SearchTerm:    "cDAQ",
				IncludeStatus: true,
				IncludeParent: true,
			}
			fwd := MustSucceed(t.Forward(ctx, msg))
			b := MustSucceed(proto.Marshal(fwd))
			var decoded RetrieveRequest
			Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
			Expect(MustSucceed(t.Backward(ctx, &decoded))).To(Equal(msg))
		})

		It("Should reject a request that sends the reserved keys field", func(
			ctx SpecContext,
		) {
			b := protowire.AppendTag(nil, 1, protowire.BytesType)
			b = protowire.AppendString(b, "dev1")
			var decoded RetrieveRequest
			Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
			Expect(t.Backward(ctx, &decoded)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("grpc.device.RetrieveRequest has unknown fields"),
				),
			))
		})

		It("Should reject an unknown field inside the request", func(ctx SpecContext) {
			inner := protowire.AppendTag(nil, 99, protowire.VarintType)
			inner = protowire.AppendVarint(inner, 1)
			b := protowire.AppendTag(nil, 13, protowire.BytesType)
			b = protowire.AppendBytes(b, inner)
			var decoded RetrieveRequest
			Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
			Expect(t.Backward(ctx, &decoded)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"api.device.pb.RetrieveRequest has unknown fields",
				)),
			))
		})
	})
})
