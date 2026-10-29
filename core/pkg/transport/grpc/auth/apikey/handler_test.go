// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package apikey_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	apiapikey "github.com/synnaxlabs/synnax/pkg/api/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	grpcauth "github.com/synnaxlabs/synnax/pkg/transport/grpc/auth"
	grpcapikey "github.com/synnaxlabs/synnax/pkg/transport/grpc/auth/apikey"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = grpc.UnaryServer[
		apiapikey.CreateRequest,
		*grpcauth.APIKeyCreateRequest,
		apiapikey.CreateResponse,
		*grpcauth.APIKeyCreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		apiapikey.RetrieveRequest,
		*grpcauth.APIKeyRetrieveRequest,
		apiapikey.RetrieveResponse,
		*grpcauth.APIKeyRetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		apiapikey.DeleteRequest,
		*grpcauth.APIKeyDeleteRequest,
		struct{},
		*emptypb.Empty,
	]
)

const errInvalidUUID = "invalid uuid"

var (
	bob    = user.User{Key: uuid.New()}
	apiKey = apiapikey.APIKey{
		Key:     uuid.New(),
		Subject: bob.OntologyID(),
		Name:    "ci",
		Created: telem.TimeStamp(9),
	}
)

var _ = Describe("APIKey", func() {
	var (
		transport   api.Transport
		bindable    = grpcapikey.New(&transport).(grpc.CompoundBindableTransport)
		keyCreate   = transport.AuthAPIKeyCreate.(*createServer)
		keyRetrieve = transport.AuthAPIKeyRetrieve.(*retrieveServer)
		keyDelete   = transport.AuthAPIKeyDelete.(*deleteServer)
	)

	Describe("New", func() {
		It("Should bind every API key server onto the API transport", func() {
			Expect(bindable).To(HaveLen(3))
			Expect(
				transport.AuthAPIKeyCreate,
			).To(BeIdenticalTo(bindable[0].(*createServer)))
			Expect(
				transport.AuthAPIKeyRetrieve,
			).To(BeIdenticalTo(bindable[1].(*retrieveServer)))
			Expect(
				transport.AuthAPIKeyDelete,
			).To(BeIdenticalTo(bindable[2].(*deleteServer)))
		})
	})

	Describe("APIKeyCreateRequest", func() {
		It("Should round-trip the subject and name", func(ctx SpecContext) {
			t := keyCreate.RequestTranslator
			msg := apiapikey.CreateRequest{Subject: bob.OntologyID(), Name: "ci"}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
	})

	Describe("APIKeyCreateResponse", func() {
		It("Should round-trip the token and key", func(ctx SpecContext) {
			t := keyCreate.ResponseTranslator
			msg := apiapikey.CreateResponse{Token: "syk_secret", Key: apiKey}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
		It("Should read a missing key as the zero key", func(ctx SpecContext) {
			res := MustSucceed(keyCreate.ResponseTranslator.Backward(
				ctx,
				&grpcauth.APIKeyCreateResponse{Token: "syk_secret"},
			))
			Expect(res.Key).To(BeZero())
		})
		It("Should reject a malformed key", func(ctx SpecContext) {
			Expect(keyCreate.ResponseTranslator.Backward(
				ctx,
				&grpcauth.APIKeyCreateResponse{Key: &grpcauth.APIKey{Key: "bad"}},
			)).Error().To(MatchError(errInvalidUUID))
		})
	})

	Describe("APIKeyRetrieveRequest", func() {
		It("Should round-trip the subjects", func(ctx SpecContext) {
			t := keyRetrieve.RequestTranslator
			msg := apiapikey.RetrieveRequest{Subjects: []ontology.ID{bob.OntologyID()}}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
	})

	Describe("APIKeyRetrieveResponse", func() {
		It("Should round-trip the keys", func(ctx SpecContext) {
			t := keyRetrieve.ResponseTranslator
			msg := apiapikey.RetrieveResponse{Keys: []apiapikey.APIKey{apiKey}}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
		It("Should reject a malformed key", func(ctx SpecContext) {
			Expect(keyRetrieve.ResponseTranslator.Backward(
				ctx,
				&grpcauth.APIKeyRetrieveResponse{
					Keys: []*grpcauth.APIKey{{Key: "bad"}},
				},
			)).Error().To(MatchError(errInvalidUUID))
		})
	})

	Describe("APIKeyDeleteRequest", func() {
		It("Should round-trip the keys", func(ctx SpecContext) {
			t := keyDelete.RequestTranslator
			msg := apiapikey.DeleteRequest{Keys: []uuid.UUID{uuid.New()}}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
		It("Should reject a malformed key", func(ctx SpecContext) {
			Expect(keyDelete.RequestTranslator.Backward(
				ctx,
				&grpcauth.APIKeyDeleteRequest{Keys: []string{"bad"}},
			)).Error().To(MatchError(errInvalidUUID))
		})
	})
})
