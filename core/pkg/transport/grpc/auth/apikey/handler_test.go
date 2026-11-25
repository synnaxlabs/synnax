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
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	grpcauth "github.com/synnaxlabs/synnax/pkg/transport/grpc/auth"
	grpcapikey "github.com/synnaxlabs/synnax/pkg/transport/grpc/auth/apikey"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = grpc.UnaryServer[
		apikey.CreateRequest,
		*grpcauth.APIKeyCreateRequest,
		apikey.CreateResponse,
		*grpcauth.APIKeyCreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		apikey.RetrieveRequest,
		*grpcauth.APIKeyRetrieveRequest,
		apikey.RetrieveResponse,
		*grpcauth.APIKeyRetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		apikey.DeleteRequest,
		*grpcauth.APIKeyDeleteRequest,
		struct{},
		*emptypb.Empty,
	]
)

const errInvalidUUID = "invalid uuid"

func matchAPIKey(k apikey.APIKey) types.GomegaMatcher {
	return PointTo(MatchFields(IgnoreExtras, Fields{
		"Key": Equal(k.Key.String()),
		"Subject": PointTo(MatchFields(IgnoreExtras, Fields{
			"Type": Equal(string(k.Subject.Type)),
			"Key":  Equal(k.Subject.Key),
		})),
		"Name":    Equal(k.Name),
		"Created": Equal(int64(k.Created)),
	}))
}

var (
	bob    = user.User{Key: uuid.New()}
	apiKey = apikey.APIKey{
		Key:     uuid.New(),
		Subject: bob.ResourceID(),
		Name:    "ci",
		Created: telem.TimeStamp(9),
	}
)

var _ = Describe("APIKey", func() {
	var transport api.Transport
	bindable, ok := grpcapikey.New(
		&transport,
		api.NewErrorCodec(),
	).(grpc.CompoundBindableTransport)
	Expect(ok).To(BeTrue())
	keyCreate, ok := transport.AuthAPIKeyCreate.(*createServer)
	Expect(ok).To(BeTrue())
	keyRetrieve, ok := transport.AuthAPIKeyRetrieve.(*retrieveServer)
	Expect(ok).To(BeTrue())
	keyDelete, ok := transport.AuthAPIKeyDelete.(*deleteServer)
	Expect(ok).To(BeTrue())

	Describe("New", func() {
		It("Should bind every API key server onto the API transport", func() {
			Expect(bindable).To(HaveExactElements(
				BeIdenticalTo(keyCreate),
				BeIdenticalTo(keyRetrieve),
				BeIdenticalTo(keyDelete),
			))
		})
	})

	Describe("APIKeyCreateRequest", func() {
		It("Should translate the subject and name", func(ctx SpecContext) {
			id := bob.ResourceID()
			Expect(MustSucceed(keyCreate.RequestTranslator.Backward(
				ctx,
				&grpcauth.APIKeyCreateRequest{
					Subject: &grpcauth.Subject{Type: string(id.Type), Key: id.Key},
					Name:    "ci",
				},
			))).To(Equal(apikey.CreateRequest{Subject: id, Name: "ci"}))
		})
	})

	Describe("APIKeyCreateResponse", func() {
		It("Should translate the token and key", func(ctx SpecContext) {
			pb := MustSucceed(keyCreate.ResponseTranslator.Forward(
				ctx,
				apikey.CreateResponse{Token: "syk_secret", Key: apiKey},
			))
			Expect(pb.Token).To(Equal("syk_secret"))
			Expect(pb.Key).To(matchAPIKey(apiKey))
		})
	})

	Describe("APIKeyRetrieveRequest", func() {
		It("Should translate the subjects", func(ctx SpecContext) {
			id := bob.ResourceID()
			Expect(MustSucceed(keyRetrieve.RequestTranslator.Backward(
				ctx,
				&grpcauth.APIKeyRetrieveRequest{Subjects: []*grpcauth.Subject{
					{Type: string(id.Type), Key: id.Key},
				}},
			)).Subjects).To(Equal([]resource.ID{id}))
		})
	})

	Describe("APIKeyRetrieveResponse", func() {
		It("Should translate the keys", func(ctx SpecContext) {
			pb := MustSucceed(keyRetrieve.ResponseTranslator.Forward(
				ctx,
				apikey.RetrieveResponse{Keys: []apikey.APIKey{apiKey}},
			))
			Expect(pb.Keys).To(ConsistOf(matchAPIKey(apiKey)))
		})
	})

	Describe("APIKeyDeleteRequest", func() {
		It("Should translate the keys", func(ctx SpecContext) {
			key := uuid.New()
			Expect(MustSucceed(keyDelete.RequestTranslator.Backward(
				ctx,
				&grpcauth.APIKeyDeleteRequest{Keys: []string{key.String()}},
			))).To(Equal(apikey.DeleteRequest{Keys: []uuid.UUID{key}}))
		})
		It("Should reject a malformed key", func(ctx SpecContext) {
			Expect(keyDelete.RequestTranslator.Backward(
				ctx,
				&grpcauth.APIKeyDeleteRequest{Keys: []string{"bad"}},
			)).Error().To(MatchError(errInvalidUUID))
		})
	})
})
