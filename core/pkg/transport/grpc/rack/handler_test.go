// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/rack"
	svcrack "github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("gRPC Rack Translators", func() {
	Describe("RetrieveRequest", func() {
		t := retrieveRequestTranslator{}
		It("Should round-trip a filter tree", func(ctx SpecContext) {
			msg := rack.RetrieveRequest{
				Where: &svcrack.Where{
					Or: []svcrack.Where{
						{Key: &filter.Set[rack.Key]{In: []rack.Key{65537}}},
						{Integrations: &filter.Contains[string]{Any: []string{"ni"}}},
					},
					Not:        &svcrack.Where{Embedded: &filter.Bool{Eq: new(true)}},
					HostIsNode: new(false),
				},
				Limit:         new(uint32(10)),
				IncludeStatus: true,
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
			b := protowire.AppendTag(nil, 1, protowire.VarintType)
			b = protowire.AppendVarint(b, 65537)
			var decoded RetrieveRequest
			Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
			Expect(t.Backward(ctx, &decoded)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("grpc.rack.RetrieveRequest has unknown fields"),
				),
			))
		})

		It("Should reject an unknown field inside the request", func(ctx SpecContext) {
			inner := protowire.AppendTag(nil, 99, protowire.VarintType)
			inner = protowire.AppendVarint(inner, 1)
			b := protowire.AppendTag(nil, 5, protowire.BytesType)
			b = protowire.AppendBytes(b, inner)
			var decoded RetrieveRequest
			Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
			Expect(t.Backward(ctx, &decoded)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"api.rack.pb.RetrieveRequest has unknown fields",
				)),
			))
		})
	})
})
