// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/ranger"
	"github.com/synnaxlabs/synnax/pkg/api/ranger/pb"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	svcranger "github.com/synnaxlabs/synnax/pkg/service/ranger"
	grpcranger "github.com/synnaxlabs/synnax/pkg/transport/grpc/ranger"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ = Describe("Retrieve", func() {
	var (
		client   grpcranger.RangeRetrieveServiceClient
		received ranger.RetrieveRequest
	)

	BeforeEach(func() {
		client = grpcranger.NewRangeRetrieveServiceClient(conn)
		received = ranger.RetrieveRequest{}
		transport.RangeRetrieve.BindHandler(func(
			_ context.Context,
			req ranger.RetrieveRequest,
		) (ranger.RetrieveResponse, error) {
			received = req
			return ranger.RetrieveResponse{}, nil
		})
	})

	It("Should decode a filter tree", func(ctx SpecContext) {
		key := uuid.New()
		msg := ranger.RetrieveRequest{
			Where: &svcranger.Where{
				Or: []svcranger.Where{
					{Key: &filter.Set[ranger.Key]{In: []ranger.Key{key}}},
					{
						Not: &svcranger.Where{
							Name: &filter.String{Eq: new("Cold flow")},
						},
					},
				},
				Overlaps: &telem.TimeRange{Start: 1, End: 2},
				Parents:  []ranger.Key{key},
			},
			Limit: new(uint32(10)),
		}
		MustSucceed(client.Exec(ctx, &grpcranger.RetrieveRequest{
			Request: MustSucceed(pb.RetrieveRequestToPB(msg)),
		}))
		Expect(received).To(Equal(msg))
	})

	It("Should reject a request that sends the reserved keys field", func(
		ctx SpecContext,
	) {
		b := protowire.AppendTag(nil, 1, protowire.BytesType)
		b = protowire.AppendString(b, uuid.New().String())
		var decoded grpcranger.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---grpc.ranger.RetrieveRequest has unknown fields",
		)))
	})

	It("Should reject an unknown field inside the request", func(ctx SpecContext) {
		inner := protowire.AppendTag(nil, 99, protowire.VarintType)
		inner = protowire.AppendVarint(inner, 1)
		b := protowire.AppendTag(nil, 3, protowire.BytesType)
		b = protowire.AppendBytes(b, inner)
		var decoded grpcranger.RetrieveRequest
		Expect(proto.Unmarshal(b, &decoded)).To(Succeed())
		Expect(client.Exec(ctx, &decoded)).Error().To(MatchError(ContainSubstring(
			"sy.validation---api.ranger.pb.RetrieveRequest has unknown fields",
		)))
	})
})

var _ = Describe("Dispatch", func() {
	var (
		client   grpcranger.RangeDispatchServiceClient
		received ranger.DispatchRequest
	)

	BeforeEach(func() {
		client = grpcranger.NewRangeDispatchServiceClient(conn)
		received = ranger.DispatchRequest{}
		transport.RangeDispatch.BindHandler(func(
			_ context.Context,
			req ranger.DispatchRequest,
		) (struct{}, error) {
			received = req
			return struct{}{}, nil
		})
	})

	It("Should decode each action from the JSON form that HTTP takes", func(
		ctx SpecContext,
	) {
		key := uuid.New()
		MustSucceed(client.Exec(ctx, &grpcranger.DispatchRequest{
			Key:         key.String(),
			DispatchKey: "dk",
			Actions: [][]byte{
				[]byte(
					`{"type": "set", "set": {"path": ["name"], "value": "Cold flow"}}`,
				),
			},
		}))
		Expect(received).To(Equal(ranger.DispatchRequest{
			Key:         key,
			DispatchKey: "dk",
			Actions: []svcranger.Action{svcranger.NewSetAction(svcranger.SetPayload{
				Path:  []string{"name"},
				Value: "Cold flow",
			})},
		}))
	})

	It("Should reject an action that does not decode", func(ctx SpecContext) {
		Expect(client.Exec(ctx, &grpcranger.DispatchRequest{
			Key:     uuid.New().String(),
			Actions: [][]byte{[]byte(`{"type": 1}`)},
		})).Error().To(MatchError(ContainSubstring(`"path":["actions","0"]`)))
	})

	It("Should encode each action as JSON that decodes to the same action", func(
		ctx SpecContext,
	) {
		translator := transport.RangeDispatch.(*fgrpc.UnaryServer[
			ranger.DispatchRequest,
			*grpcranger.DispatchRequest,
			struct{},
			*emptypb.Empty,
		]).RequestTranslator.(fgrpc.Translator[
			ranger.DispatchRequest,
			*grpcranger.DispatchRequest,
		])
		req := ranger.DispatchRequest{
			Key:         uuid.New(),
			DispatchKey: "dk",
			Actions: []svcranger.Action{svcranger.NewSetAction(svcranger.SetPayload{
				Path:   []string{"name"},
				Value:  "Cold flow",
				Guards: []patch.Guard{{Path: []string{"name"}, Value: "Hot fire"}},
			})},
		}
		Expect(translator.Backward(ctx, MustSucceed(translator.Forward(ctx, req)))).
			To(Equal(req))
	})
})

var _ = Describe("Dispatch to a range", func() {
	It("Should store a time range end above 2^53 exactly", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		groups := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		svc := MustOpen(svcranger.OpenService(ctx, svcranger.ServiceConfig{
			DB:    db,
			Group: groups,
		}))
		r := svcranger.Range{
			Name:      "Hot fire",
			TimeRange: telem.TimeRange{Start: 1, End: 2},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &r)
		})).To(Succeed())
		transport.RangeDispatch.BindHandler(func(
			ctx context.Context,
			req ranger.DispatchRequest,
		) (struct{}, error) {
			return struct{}{}, db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).
					WithDispatchKey(req.DispatchKey).
					Apply(ctx, req.Key, req.Actions...)
			})
		})
		MustSucceed(grpcranger.NewRangeDispatchServiceClient(conn).Exec(
			ctx,
			&grpcranger.DispatchRequest{
				Key: r.Key.String(),
				Actions: [][]byte{[]byte(`{"type": "set", "set": {
					"path": ["time_range", "end"],
					"value": "1700000000000000123"
				}}`)},
			},
		))
		var stored svcranger.Range
		Expect(svc.NewRetrieve().
			Where(svcranger.MatchKeys(r.Key)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		Expect(stored.TimeRange.End).To(Equal(telem.TimeStamp(1700000000000000123)))
	})
})
