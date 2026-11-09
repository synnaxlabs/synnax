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
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api/task"
	"github.com/synnaxlabs/synnax/pkg/api/task/pb"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	svctask "github.com/synnaxlabs/synnax/pkg/service/task"
	grpctask "github.com/synnaxlabs/synnax/pkg/transport/grpc/task"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
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

var _ = Describe("Dispatch", func() {
	var (
		client   grpctask.TaskDispatchServiceClient
		received task.DispatchRequest
	)

	BeforeEach(func() {
		client = grpctask.NewTaskDispatchServiceClient(conn)
		received = task.DispatchRequest{}
		transport.TaskDispatch.BindHandler(func(
			_ context.Context,
			req task.DispatchRequest,
		) (task.DispatchResponse, error) {
			received = req
			return task.DispatchResponse{}, nil
		})
	})

	It("Should decode each action from the JSON form that HTTP takes", func(
		ctx SpecContext,
	) {
		key := uuid.New()
		MustSucceed(client.Exec(ctx, &grpctask.DispatchRequest{
			Key:         key.String(),
			DispatchKey: "dk",
			Actions: [][]byte{
				[]byte(`{"type": "set", "set": {"path": ["name"], "value": "Read"}}`),
			},
		}))
		Expect(received).To(Equal(task.DispatchRequest{
			Key:         key,
			DispatchKey: "dk",
			Actions: []svctask.Action{svctask.NewSetAction(svctask.SetPayload{
				Path:  []string{"name"},
				Value: "Read",
			})},
		}))
	})

	It("Should reject an action that does not decode", func(ctx SpecContext) {
		Expect(client.Exec(ctx, &grpctask.DispatchRequest{
			Key:     uuid.New().String(),
			Actions: [][]byte{[]byte(`{"type": 1}`)},
		})).Error().To(MatchError(ContainSubstring(`"path":["actions","0"]`)))
	})

	It("Should encode each action as JSON that decodes to the same action", func(
		ctx SpecContext,
	) {
		translator := transport.TaskDispatch.(*fgrpc.UnaryServer[
			task.DispatchRequest,
			*grpctask.DispatchRequest,
			task.DispatchResponse,
			*emptypb.Empty,
		]).RequestTranslator.(fgrpc.Translator[
			task.DispatchRequest,
			*grpctask.DispatchRequest,
		])
		req := task.DispatchRequest{
			Key:         uuid.New(),
			DispatchKey: "dk",
			Actions: []svctask.Action{svctask.NewSetAction(svctask.SetPayload{
				Path:   []string{"name"},
				Value:  "Read",
				Guards: []patch.Guard{{Path: []string{"name"}, Value: "Write"}},
			})},
		}
		Expect(translator.Backward(ctx, MustSucceed(translator.Forward(ctx, req)))).
			To(Equal(req))
	})
})
