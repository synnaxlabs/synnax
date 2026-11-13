// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package chassis_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/api/ni/chassis/pb"
	svcchassis "github.com/synnaxlabs/synnax/pkg/service/ni/chassis"
	chassispb "github.com/synnaxlabs/synnax/pkg/service/ni/chassis/pb"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	grpcchassis "github.com/synnaxlabs/synnax/pkg/transport/grpc/ni/chassis"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Handler", func() {
	It("Should create chassis and return their keys", func(ctx SpecContext) {
		key := uuid.New()
		var received chassis.CreateRequest
		transport.NIChassisCreate.BindHandler(func(
			_ context.Context,
			req chassis.CreateRequest,
		) (chassis.CreateResponse, error) {
			received = req
			req.Chassis[0].Key = key
			return chassis.CreateResponse(req), nil
		})
		pbChassis := MustSucceed(chassispb.ChassisToPB(svcchassis.Chassis{
			Rack:         rack.NewKey(1, 2),
			Name:         "cDAQ-9189",
			ResourceName: "{GUID}",
			Model:        "cDAQ-9189",
		}))
		res := MustSucceed(grpcchassis.NewNIChassisCreateServiceClient(conn).Exec(
			ctx,
			&grpcchassis.CreateRequest{Chassis: []*chassispb.Chassis{pbChassis}},
		))
		Expect(received.Chassis[0].ResourceName).To(Equal("{GUID}"))
		Expect(res.Chassis[0].Key).To(Equal(key.String()))
	})

	It("Should retrieve chassis by serial", func(ctx SpecContext) {
		var received chassis.RetrieveRequest
		transport.NIChassisRetrieve.BindHandler(func(
			_ context.Context,
			req chassis.RetrieveRequest,
		) (chassis.RetrieveResponse, error) {
			received = req
			return chassis.RetrieveResponse{}, nil
		})
		where := MustSucceed(chassispb.WhereToPB(svcchassis.Where{
			Serial: &filter.String{Eq: new("01A")},
		}))
		MustSucceed(grpcchassis.NewNIChassisRetrieveServiceClient(conn).Exec(
			ctx,
			&pb.RetrieveRequest{Where: where},
		))
		Expect(*received.Where.Serial.Eq).To(Equal("01A"))
	})

	It("Should update the configuration group of a chassis", func(ctx SpecContext) {
		key := uuid.New()
		var received chassis.UpdateRequest
		transport.NIChassisUpdate.BindHandler(func(
			_ context.Context,
			req chassis.UpdateRequest,
		) (struct{}, error) {
			received = req
			return struct{}{}, nil
		})
		update := svcchassis.Update{Variant: svcchassis.ConfigurationUpdate{
			Configuration: svcchassis.Configuration{Name: "Bench chassis"},
		}}
		MustSucceed(grpcchassis.NewNIChassisUpdateServiceClient(conn).Exec(
			ctx,
			&grpcchassis.UpdateRequest{
				Key:    key.String(),
				Update: MustSucceed(chassispb.UpdateToPB(update)),
			},
		))
		Expect(received.Key).To(Equal(key))
		Expect(received.Update).To(Equal(update))
	})

	It("Should delete chassis by key", func(ctx SpecContext) {
		key := uuid.New()
		var received chassis.DeleteRequest
		transport.NIChassisDelete.BindHandler(func(
			_ context.Context,
			req chassis.DeleteRequest,
		) (struct{}, error) {
			received = req
			return struct{}{}, nil
		})
		MustSucceed(grpcchassis.NewNIChassisDeleteServiceClient(conn).Exec(
			ctx,
			&grpcchassis.DeleteRequest{Keys: []string{key.String()}},
		))
		Expect(received.Keys).To(Equal([]svcchassis.Key{key}))
	})

	It("Should reject a key that is not a UUID", func(ctx SpecContext) {
		Expect(grpcchassis.NewNIChassisDeleteServiceClient(conn).Exec(
			ctx,
			&grpcchassis.DeleteRequest{Keys: []string{"not-a-uuid"}},
		)).Error().To(MatchError(ContainSubstring("invalid uuid")))
	})
})
