// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	grpcauth "github.com/synnaxlabs/synnax/pkg/transport/grpc/auth"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/types/known/emptypb"
)

type methodsServer = grpc.UnaryServer[
	struct{},
	*emptypb.Empty,
	auth.MethodsResponse,
	*grpcauth.MethodsResponse,
]

var _ = Describe("Auth", func() {
	var (
		transport api.Transport
		bindable  = grpcauth.New(&transport)
		methods   = transport.AuthMethods.(*methodsServer)
	)

	Describe("New", func() {
		It("Should bind the methods server onto the API transport", func() {
			Expect(transport.AuthMethods).To(BeIdenticalTo(bindable.(*methodsServer)))
		})
	})

	Describe("Subject", func() {
		It("Should round-trip an ontology ID", func() {
			id := ontology.ID{Type: ontology.ResourceTypeUser, Key: "bob"}
			Expect(grpcauth.SubjectBackward(grpcauth.SubjectForward(id))).To(Equal(id))
		})
		It("Should read a missing subject as the zero ID", func() {
			Expect(grpcauth.SubjectBackward(nil)).To(BeZero())
		})
	})

	Describe("MethodsResponse", func() {
		It("Should translate the methods", func(ctx SpecContext) {
			Expect(MustSucceed(methods.ResponseTranslator.Forward(
				ctx,
				auth.MethodsResponse{Methods: []svcauth.Method{
					svcauth.MethodAPIKey,
					svcauth.MethodPassword,
				}},
			)).Methods).To(Equal([]string{"api_key", "password"}))
		})
		It("Should translate an empty method list", func(ctx SpecContext) {
			Expect(MustSucceed(methods.ResponseTranslator.Forward(
				ctx,
				auth.MethodsResponse{Methods: []svcauth.Method{}},
			)).Methods).To(BeEmpty())
		})
	})
})
