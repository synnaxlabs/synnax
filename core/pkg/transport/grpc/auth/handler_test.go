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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	grpcauth "github.com/synnaxlabs/synnax/pkg/transport/grpc/auth"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	loginServer = grpc.UnaryServer[
		auth.LoginRequest,
		*grpcauth.LoginRequest,
		auth.LoginResponse,
		*grpcauth.LoginResponse,
	]
	sessionCreateServer = grpc.UnaryServer[
		auth.SessionCreateRequest,
		*grpcauth.SessionCreateRequest,
		auth.SessionCreateResponse,
		*grpcauth.SessionCreateResponse,
	]
	sessionRenewServer = grpc.UnaryServer[
		struct{},
		*emptypb.Empty,
		auth.SessionRenewResponse,
		*grpcauth.SessionRenewResponse,
	]
	sessionRetrieveServer = grpc.UnaryServer[
		auth.SessionRetrieveRequest,
		*grpcauth.SessionRetrieveRequest,
		auth.SessionRetrieveResponse,
		*grpcauth.SessionRetrieveResponse,
	]
	sessionDeleteServer = grpc.UnaryServer[
		auth.SessionDeleteRequest,
		*grpcauth.SessionDeleteRequest,
		struct{},
		*emptypb.Empty,
	]
	keyCreateServer = grpc.UnaryServer[
		auth.KeyCreateRequest,
		*grpcauth.KeyCreateRequest,
		auth.KeyCreateResponse,
		*grpcauth.KeyCreateResponse,
	]
	keyRetrieveServer = grpc.UnaryServer[
		auth.KeyRetrieveRequest,
		*grpcauth.KeyRetrieveRequest,
		auth.KeyRetrieveResponse,
		*grpcauth.KeyRetrieveResponse,
	]
	keyDeleteServer = grpc.UnaryServer[
		auth.KeyDeleteRequest,
		*grpcauth.KeyDeleteRequest,
		struct{},
		*emptypb.Empty,
	]
)

const errInvalidUUID = "invalid uuid"

func matchSession(s svcauth.Session) types.GomegaMatcher {
	return PointTo(MatchFields(IgnoreExtras, Fields{
		"Key": Equal(s.Key.String()),
		"Subject": PointTo(MatchFields(IgnoreExtras, Fields{
			"Type": Equal(string(s.Subject.Type)),
			"Key":  Equal(s.Subject.Key),
		})),
		"Credential": Equal(s.Credential.String()),
		"LastActive": Equal(int64(s.LastActive)),
	}))
}

func matchAPIKey(k auth.APIKey) types.GomegaMatcher {
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
	apiKey = auth.APIKey{
		Key:     uuid.New(),
		Subject: bob.OntologyID(),
		Name:    "ci",
		Created: telem.TimeStamp(9),
	}
	session = svcauth.Session{
		Key:        uuid.New(),
		Subject:    bob.OntologyID(),
		Credential: uuid.New(),
		LastActive: telem.TimeStamp(42),
	}
)

var _ = Describe("Auth", func() {
	var (
		transport   api.Transport
		bindable    = grpcauth.New(&transport).(grpc.CompoundBindableTransport)
		login       = transport.AuthLogin.(*loginServer)
		create      = transport.AuthSessionCreate.(*sessionCreateServer)
		renew       = transport.AuthSessionRenew.(*sessionRenewServer)
		retrieve    = transport.AuthSessionRetrieve.(*sessionRetrieveServer)
		del         = transport.AuthSessionDelete.(*sessionDeleteServer)
		keyCreate   = transport.AuthKeyCreate.(*keyCreateServer)
		keyRetrieve = transport.AuthKeyRetrieve.(*keyRetrieveServer)
		keyDelete   = transport.AuthKeyDelete.(*keyDeleteServer)
	)

	Describe("New", func() {
		It("Should bind every auth server onto the API transport", func() {
			Expect(bindable).To(HaveLen(8))
			Expect(transport.AuthLogin).To(BeIdenticalTo(bindable[0].(*loginServer)))
			Expect(
				transport.AuthSessionCreate,
			).To(BeIdenticalTo(bindable[1].(*sessionCreateServer)))
			Expect(
				transport.AuthSessionRenew,
			).To(BeIdenticalTo(bindable[2].(*sessionRenewServer)))
			Expect(
				transport.AuthSessionRetrieve,
			).To(BeIdenticalTo(bindable[3].(*sessionRetrieveServer)))
			Expect(
				transport.AuthSessionDelete,
			).To(BeIdenticalTo(bindable[4].(*sessionDeleteServer)))
			Expect(
				transport.AuthKeyCreate,
			).To(BeIdenticalTo(bindable[5].(*keyCreateServer)))
			Expect(
				transport.AuthKeyRetrieve,
			).To(BeIdenticalTo(bindable[6].(*keyRetrieveServer)))
			Expect(
				transport.AuthKeyDelete,
			).To(BeIdenticalTo(bindable[7].(*keyDeleteServer)))
		})
	})

	Describe("LoginRequest", func() {
		It("Should translate the credentials", func(ctx SpecContext) {
			Expect(MustSucceed(login.RequestTranslator.Backward(
				ctx,
				&grpcauth.LoginRequest{Username: "bob", Password: "p"},
			))).To(Equal(auth.LoginRequest{Username: "bob", Password: "p"}))
		})
	})

	Describe("LoginResponse", func() {
		It("Should translate the token, user, and cluster info", func(
			ctx SpecContext,
		) {
			pb := MustSucceed(login.ResponseTranslator.Forward(ctx, auth.LoginResponse{
				Token: "tk",
				User:  bob,
				ClusterInfo: auth.ClusterInfo{
					ClusterKey:  "cluster",
					NodeVersion: "0.1.0",
					NodeKey:     1,
					NodeTime:    telem.TimeStamp(7),
				},
			}))
			Expect(pb.Token).To(Equal("tk"))
			Expect(pb.User.Key).To(Equal(bob.Key.String()))
			Expect(pb.ClusterInfo.ClusterKey).To(Equal("cluster"))
			Expect(pb.ClusterInfo.NodeVersion).To(Equal("0.1.0"))
			Expect(pb.ClusterInfo.NodeKey).To(Equal(uint32(1)))
			Expect(pb.ClusterInfo.NodeTime).To(Equal(int64(7)))
		})
	})

	Describe("SessionCreateRequest", func() {
		It("Should translate a password proof", func(ctx SpecContext) {
			Expect(
				MustSucceed(create.RequestTranslator.Backward(
					ctx,
					&grpcauth.SessionCreateRequest{Proof: &grpcauth.Proof{
						Method:   string(svcauth.MethodPassword),
						Username: "bob",
						Password: "p",
					}},
				)),
			).To(Equal(auth.SessionCreateRequest{Proof: svcauth.Proof{
				Variant: svcauth.PasswordProof{
					Credentials: svcauth.Credentials{Username: "bob", Password: "p"},
				},
			}}))
		})
		It("Should translate an API key proof", func(ctx SpecContext) {
			Expect(
				MustSucceed(create.RequestTranslator.Backward(
					ctx,
					&grpcauth.SessionCreateRequest{Proof: &grpcauth.Proof{
						Method: string(svcauth.MethodAPIKey),
						Key:    "syk_secret",
					}},
				)),
			).To(Equal(auth.SessionCreateRequest{Proof: svcauth.Proof{
				Variant: svcauth.APIKeyProof{
					APIKeyCredentials: svcauth.APIKeyCredentials{Key: "syk_secret"},
				},
			}}))
		})
		It("Should reject a missing proof", func(ctx SpecContext) {
			Expect(create.RequestTranslator.Backward(
				ctx,
				&grpcauth.SessionCreateRequest{},
			)).Error().To(MatchError(validate.ErrValidation))
		})
		It("Should reject an unknown method", func(ctx SpecContext) {
			Expect(
				create.RequestTranslator.Backward(ctx, &grpcauth.SessionCreateRequest{
					Proof: &grpcauth.Proof{Method: "carrier_pigeon"},
				}),
			).Error().
				To(MatchError(validate.ErrValidation))
		})
	})

	Describe("SessionCreateResponse", func() {
		It("Should translate the token, session, and user", func(ctx SpecContext) {
			pb := MustSucceed(create.ResponseTranslator.Forward(
				ctx,
				auth.SessionCreateResponse{Token: "tk", Session: session, User: &bob},
			))
			Expect(pb.Token).To(Equal("tk"))
			Expect(pb.Session).To(matchSession(session))
			Expect(pb.User.Key).To(Equal(bob.Key.String()))
		})
		It("Should translate a subject that is not a user", func(ctx SpecContext) {
			pb := MustSucceed(create.ResponseTranslator.Forward(
				ctx,
				auth.SessionCreateResponse{Token: "tk", Session: session},
			))
			Expect(pb.User).To(BeNil())
		})
	})

	Describe("SessionRenewResponse", func() {
		It("Should translate the session and user", func(ctx SpecContext) {
			pb := MustSucceed(renew.ResponseTranslator.Forward(
				ctx,
				auth.SessionRenewResponse{Session: session, User: &bob},
			))
			Expect(pb.Session).To(matchSession(session))
			Expect(pb.User.Key).To(Equal(bob.Key.String()))
		})
		It("Should translate a subject that is not a user", func(ctx SpecContext) {
			pb := MustSucceed(renew.ResponseTranslator.Forward(
				ctx,
				auth.SessionRenewResponse{Session: session},
			))
			Expect(pb.User).To(BeNil())
		})
	})

	Describe("SessionRetrieveRequest", func() {
		It("Should translate the subjects", func(ctx SpecContext) {
			id := bob.OntologyID()
			Expect(MustSucceed(retrieve.RequestTranslator.Backward(
				ctx,
				&grpcauth.SessionRetrieveRequest{Subjects: []*grpcauth.Subject{
					{Type: string(id.Type), Key: id.Key},
				}},
			)).Subjects).To(Equal([]ontology.ID{id}))
		})
		It("Should read a missing subject as the zero ID", func(ctx SpecContext) {
			res := MustSucceed(retrieve.RequestTranslator.Backward(
				ctx,
				&grpcauth.SessionRetrieveRequest{Subjects: []*grpcauth.Subject{nil}},
			))
			Expect(res.Subjects).To(Equal([]ontology.ID{{}}))
		})
	})

	Describe("SessionRetrieveResponse", func() {
		It("Should translate the sessions", func(ctx SpecContext) {
			pb := MustSucceed(retrieve.ResponseTranslator.Forward(
				ctx,
				auth.SessionRetrieveResponse{Sessions: []svcauth.Session{session}},
			))
			Expect(pb.Sessions).To(ConsistOf(matchSession(session)))
		})
	})

	Describe("SessionDeleteRequest", func() {
		It("Should translate the keys", func(ctx SpecContext) {
			key := uuid.New()
			Expect(MustSucceed(del.RequestTranslator.Backward(
				ctx,
				&grpcauth.SessionDeleteRequest{Keys: []string{key.String()}},
			))).To(Equal(auth.SessionDeleteRequest{Keys: []svcauth.SessionKey{key}}))
		})
		It("Should reject a malformed key", func(ctx SpecContext) {
			Expect(del.RequestTranslator.Backward(
				ctx,
				&grpcauth.SessionDeleteRequest{Keys: []string{"bad"}},
			)).Error().To(MatchError(errInvalidUUID))
		})
	})

	Describe("KeyCreateRequest", func() {
		It("Should translate the subject and name", func(ctx SpecContext) {
			id := bob.OntologyID()
			Expect(MustSucceed(keyCreate.RequestTranslator.Backward(
				ctx,
				&grpcauth.KeyCreateRequest{
					Subject: &grpcauth.Subject{Type: string(id.Type), Key: id.Key},
					Name:    "ci",
				},
			))).To(Equal(auth.KeyCreateRequest{Subject: id, Name: "ci"}))
		})
	})

	Describe("KeyCreateResponse", func() {
		It("Should translate the token and key", func(ctx SpecContext) {
			pb := MustSucceed(keyCreate.ResponseTranslator.Forward(
				ctx,
				auth.KeyCreateResponse{Token: "syk_secret", Key: apiKey},
			))
			Expect(pb.Token).To(Equal("syk_secret"))
			Expect(pb.Key).To(matchAPIKey(apiKey))
		})
	})

	Describe("KeyRetrieveRequest", func() {
		It("Should translate the subjects", func(ctx SpecContext) {
			id := bob.OntologyID()
			Expect(MustSucceed(keyRetrieve.RequestTranslator.Backward(
				ctx,
				&grpcauth.KeyRetrieveRequest{Subjects: []*grpcauth.Subject{
					{Type: string(id.Type), Key: id.Key},
				}},
			)).Subjects).To(Equal([]ontology.ID{id}))
		})
	})

	Describe("KeyRetrieveResponse", func() {
		It("Should translate the keys", func(ctx SpecContext) {
			pb := MustSucceed(keyRetrieve.ResponseTranslator.Forward(
				ctx,
				auth.KeyRetrieveResponse{Keys: []auth.APIKey{apiKey}},
			))
			Expect(pb.Keys).To(ConsistOf(matchAPIKey(apiKey)))
		})
	})

	Describe("KeyDeleteRequest", func() {
		It("Should translate the keys", func(ctx SpecContext) {
			key := uuid.New()
			Expect(MustSucceed(keyDelete.RequestTranslator.Backward(
				ctx,
				&grpcauth.KeyDeleteRequest{Keys: []string{key.String()}},
			))).To(Equal(auth.KeyDeleteRequest{Keys: []uuid.UUID{key}}))
		})
		It("Should reject a malformed key", func(ctx SpecContext) {
			Expect(keyDelete.RequestTranslator.Backward(
				ctx,
				&grpcauth.KeyDeleteRequest{Keys: []string{"bad"}},
			)).Error().To(MatchError(errInvalidUUID))
		})
	})
})
