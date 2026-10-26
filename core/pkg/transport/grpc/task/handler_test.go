// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/task"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	svctask "github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("gRPC Task Translators", func() {
	Describe("RetrieveRequest", func() {
		t := retrieveRequestTranslator{}
		It("Should round-trip a filter tree", func(ctx SpecContext) {
			msg := task.RetrieveRequest{
				Where: &svctask.Where{
					Or: []svctask.Where{
						{Key: &filter.Set[uuid.UUID]{In: []uuid.UUID{uuid.New()}}},
						{Type: &filter.String{Eq: new("ni_analog_read")}},
					},
					Not: &svctask.Where{
						Rack: &filter.Set[rack.Key]{Eq: new(rack.Key(1))},
					},
					Name:     &filter.String{Regex: new("^PT")},
					Internal: &filter.Bool{Eq: new(false)},
				},
				SearchTerm:    "read",
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
			b := protowire.AppendTag(nil, 2, protowire.BytesType)
			b = protowire.AppendString(b, uuid.New().String())
			var decoded RetrieveRequest
			Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
			Expect(t.Backward(ctx, &decoded)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("grpc.task.RetrieveRequest has unknown fields"),
				),
			))
		})

		It("Should reject an unknown field inside the request", func(ctx SpecContext) {
			inner := protowire.AppendTag(nil, 99, protowire.VarintType)
			inner = protowire.AppendVarint(inner, 1)
			b := protowire.AppendTag(nil, 6, protowire.BytesType)
			b = protowire.AppendBytes(b, inner)
			var decoded RetrieveRequest
			Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
			Expect(t.Backward(ctx, &decoded)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"api.task.pb.RetrieveRequest has unknown fields",
				)),
			))
		})
	})
})
