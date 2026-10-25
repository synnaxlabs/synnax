// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("gRPC Channel Translators", func() {
	Describe("RetrieveRequest", func() {
		t := retrieveRequestTranslator{}
		It("Should round-trip a filter tree", func(ctx SpecContext) {
			msg := channel.RetrieveRequest{
				Where: &channel.Where{
					Or: []channel.Where{
						{Key: &filter.Set[channel.Key]{In: []channel.Key{1048577}}},
						{Name: &filter.String{Regex: new("^pressure")}},
					},
					Not: &channel.Where{
						DataType: &filter.Set[telem.DataType]{
							In: []telem.DataType{telem.StringT},
						},
					},
					Leaseholder: &filter.Ordered[node.Key]{Eq: new(node.Key(1))},
					IsIndex:     &filter.Bool{Eq: new(false)},
					Virtual:     new(false),
				},
				SearchTerm:    new("pressure"),
				Limit:         new(uint32(10)),
				IncludeStatus: new(true),
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
			b := protowire.AppendTag(nil, 2, protowire.VarintType)
			b = protowire.AppendVarint(b, 1048577)
			var decoded RetrieveRequest
			Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
			_, err := t.Backward(ctx, &decoded)
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring(
				"retrieve request has unknown fields",
			)))
		})
	})
})
