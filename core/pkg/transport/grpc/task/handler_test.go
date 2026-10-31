// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/task"
	"github.com/synnaxlabs/synnax/pkg/api/task/pb"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	svctask "github.com/synnaxlabs/synnax/pkg/service/task"
	grpctask "github.com/synnaxlabs/synnax/pkg/transport/grpc/task"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("Retrieve", func() {
	var (
		client   grpctask.TaskRetrieveServiceClient
		received task.RetrieveRequest
	)

	BeforeEach(func() {
		client = grpctask.NewTaskRetrieveServiceClient(conn)
		received = task.RetrieveRequest{}
		transport.TaskRetrieve.BindHandler(func(
			_ context.Context,
			req task.RetrieveRequest,
		) (task.RetrieveResponse, error) {
			received = req
			return task.RetrieveResponse{}, nil
		})
	})

	It("Should decode a filter tree", func(ctx SpecContext) {
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
		MustSucceed(client.Exec(ctx, &grpctask.RetrieveRequest{
			Request: MustSucceed(pb.RetrieveRequestToPB(msg)),
		}))
		Expect(received).To(Equal(msg))
	})

	It("Should reject a request that sends the reserved keys field", func(
		ctx SpecContext,
	) {
		b := protowire.AppendTag(nil, 2, protowire.BytesType)
		b = protowire.AppendString(b, uuid.New().String())
		var decoded grpctask.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---grpc.task.RetrieveRequest has unknown fields",
		)))
	})

	It("Should reject an unknown field inside the request", func(ctx SpecContext) {
		inner := protowire.AppendTag(nil, 99, protowire.VarintType)
		inner = protowire.AppendVarint(inner, 1)
		b := protowire.AppendTag(nil, 6, protowire.BytesType)
		b = protowire.AppendBytes(b, inner)
		var decoded grpctask.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---api.task.pb.RetrieveRequest has unknown fields",
		)))
	})
})
