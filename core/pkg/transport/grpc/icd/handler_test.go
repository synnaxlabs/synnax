// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/icd"
	"github.com/synnaxlabs/synnax/pkg/api/icd/pb"
	svcicd "github.com/synnaxlabs/synnax/pkg/service/icd"
	icdpb "github.com/synnaxlabs/synnax/pkg/service/icd/pb"
	grpcicd "github.com/synnaxlabs/synnax/pkg/transport/grpc/icd"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/payload"
	. "github.com/synnaxlabs/x/testutil"
)

var statusICD = svcicd.ICD{
	Key:  uuid.New(),
	Name: "Rig",
	Messages: []svcicd.Message{{
		Key:  uuid.New(),
		Name: "Status",
		Payload: svcicd.Payload{Variant: svcicd.JSONPayload{
			Fields: []svcicd.JSONField{{Variant: svcicd.FloatJSONField{
				JSONBaseField: svcicd.JSONBaseField{
					Key:     uuid.New(),
					Name:    "pressure",
					Scale:   1,
					Pointer: "/pressure",
				},
				EnumValues: []payload.FloatEntry{{Label: "HIGH", Value: 100}},
				Default:    new(payload.Float(0)),
			}}},
		}},
	}},
}

var _ = Describe("Handler", func() {
	BeforeEach(func() {
		transport.ICDCreate.BindHandler(func(
			context.Context,
			icd.CreateRequest,
		) (icd.CreateResponse, error) {
			return icd.CreateResponse{}, nil
		})
	})

	It("Should create ICDs and return their keys", func(ctx SpecContext) {
		key := uuid.New()
		var received icd.CreateRequest
		transport.ICDCreate.BindHandler(func(
			_ context.Context,
			req icd.CreateRequest,
		) (icd.CreateResponse, error) {
			received = req
			created := req.ICDs[0]
			created.Key = key
			return icd.CreateResponse{ICDs: []svcicd.ICD{created}}, nil
		})
		res := MustSucceed(grpcicd.NewICDCreateServiceClient(conn).Exec(
			ctx,
			&grpcicd.CreateRequest{
				Icds: []*icdpb.ICD{MustSucceed(icdpb.ICDToPB(statusICD))},
			},
		))
		Expect(received.ICDs).To(Equal([]svcicd.ICD{statusICD}))
		Expect(res.Icds[0].Key).To(Equal(key.String()))
	})

	It("Should retrieve ICDs by key", func(ctx SpecContext) {
		var received icd.RetrieveRequest
		transport.ICDRetrieve.BindHandler(func(
			_ context.Context,
			req icd.RetrieveRequest,
		) (icd.RetrieveResponse, error) {
			received = req
			return icd.RetrieveResponse{ICDs: []svcicd.ICD{statusICD}}, nil
		})
		where := MustSucceed(icdpb.WhereToPB(svcicd.Where{
			Key: &filter.Set[svcicd.Key]{Eq: &statusICD.Key},
		}))
		res := MustSucceed(grpcicd.NewICDRetrieveServiceClient(conn).Exec(
			ctx,
			&pb.RetrieveRequest{Where: where},
		))
		Expect(*received.Where.Key.Eq).To(Equal(statusICD.Key))
		Expect(icdpb.ICDSFromPB(res.Icds)).To(Equal([]svcicd.ICD{statusICD}))
	})

	It("Should reject a created ICD with an invalid key", func(ctx SpecContext) {
		Expect(grpcicd.NewICDCreateServiceClient(conn).Exec(
			ctx,
			&grpcicd.CreateRequest{Icds: []*icdpb.ICD{{Key: "not-a-uuid"}}},
		)).Error().To(MatchError(ContainSubstring("invalid uuid")))
	})
})
