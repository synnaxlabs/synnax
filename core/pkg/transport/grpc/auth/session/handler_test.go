// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package session_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	apisession "github.com/synnaxlabs/synnax/pkg/api/auth/session"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	grpcauth "github.com/synnaxlabs/synnax/pkg/transport/grpc/auth"
	grpcsession "github.com/synnaxlabs/synnax/pkg/transport/grpc/auth/session"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	loginServer = grpc.UnaryServer[
		apisession.LoginRequest,
		*grpcauth.LoginRequest,
		apisession.LoginResponse,
		*grpcauth.LoginResponse,
	]
	createServer = grpc.UnaryServer[
		apisession.CreateRequest,
		*grpcauth.SessionCreateRequest,
		apisession.CreateResponse,
		*grpcauth.SessionCreateResponse,
	]
	renewServer = grpc.UnaryServer[
		struct{},
		*emptypb.Empty,
		apisession.RenewResponse,
		*grpcauth.SessionRenewResponse,
	]
	retrieveServer = grpc.UnaryServer[
		apisession.RetrieveRequest,
		*grpcauth.SessionRetrieveRequest,
		apisession.RetrieveResponse,
		*grpcauth.SessionRetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		apisession.DeleteRequest,
		*grpcauth.SessionDeleteRequest,
		struct{},
		*emptypb.Empty,
	]
)

const errInvalidUUID = "invalid uuid"

var (
	bob     = user.User{Key: uuid.New()}
	session = svcauth.Session{
		Key:        uuid.New(),
		Subject:    bob.OntologyID(),
		Credential: uuid.New(),
		LastActive: telem.TimeStamp(42),
	}
)

var _ = Describe("Session", func() {
	var (
		transport api.Transport
		bindable  = grpcsession.New(&transport).(grpc.CompoundBindableTransport)
		login     = transport.AuthLogin.(*loginServer)
		create    = transport.AuthSessionCreate.(*createServer)
		renew     = transport.AuthSessionRenew.(*renewServer)
		retrieve  = transport.AuthSessionRetrieve.(*retrieveServer)
		del       = transport.AuthSessionDelete.(*deleteServer)
	)

	Describe("New", func() {
		It("Should bind every session server onto the API transport", func() {
			Expect(bindable).To(HaveLen(5))
			Expect(transport.AuthLogin).To(BeIdenticalTo(bindable[0].(*loginServer)))
			Expect(
				transport.AuthSessionCreate,
			).To(BeIdenticalTo(bindable[1].(*createServer)))
			Expect(
				transport.AuthSessionRenew,
			).To(BeIdenticalTo(bindable[2].(*renewServer)))
			Expect(
				transport.AuthSessionRetrieve,
			).To(BeIdenticalTo(bindable[3].(*retrieveServer)))
			Expect(
				transport.AuthSessionDelete,
			).To(BeIdenticalTo(bindable[4].(*deleteServer)))
		})
	})

	Describe("LoginRequest", func() {
		It("Should round-trip the credentials", func(ctx SpecContext) {
			t := login.RequestTranslator
			msg := apisession.LoginRequest{Username: "bob", Password: "p"}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.Username).To(Equal("bob"))
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
	})

	Describe("LoginResponse", func() {
		It("Should round-trip the token, user, and cluster info", func(
			ctx SpecContext,
		) {
			t := login.ResponseTranslator
			msg := apisession.LoginResponse{
				Token: "tk",
				User:  bob,
				ClusterInfo: apiauth.ClusterInfo{
					ClusterKey:  "cluster",
					NodeVersion: "0.1.0",
					NodeKey:     1,
					NodeTime:    telem.TimeStamp(7),
				},
			}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
		It("Should reject a malformed user key", func(ctx SpecContext) {
			Expect(login.ResponseTranslator.Backward(ctx, &grpcauth.LoginResponse{
				User:        &grpcauth.User{Key: "bad"},
				ClusterInfo: &grpcauth.ClusterInfo{},
			})).Error().To(MatchError(errInvalidUUID))
		})
	})

	Describe("SessionCreateRequest", func() {
		It("Should round-trip a password proof", func(ctx SpecContext) {
			t := create.RequestTranslator
			msg := apisession.CreateRequest{Proof: svcauth.Proof{
				Variant: svcauth.PasswordProof{
					Credentials: svcauth.Credentials{Username: "bob", Password: "p"},
				},
			}}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.Proof.Method).To(Equal(string(svcauth.MethodPassword)))
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
		It("Should round-trip an API key proof", func(ctx SpecContext) {
			t := create.RequestTranslator
			msg := apisession.CreateRequest{Proof: svcauth.Proof{
				Variant: svcauth.APIKeyProof{
					APIKeyCredentials: svcauth.APIKeyCredentials{Key: "syk_secret"},
				},
			}}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.Proof.Method).To(Equal(string(svcauth.MethodAPIKey)))
			Expect(fwd.Proof.Key).To(Equal("syk_secret"))
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
		It("Should round-trip an OIDC proof", func(ctx SpecContext) {
			t := create.RequestTranslator
			msg := apisession.CreateRequest{Proof: svcauth.Proof{
				Variant: svcauth.OIDCProof{
					OIDCCredentials: svcauth.OIDCCredentials{
						Provider:    "idp",
						Code:        "code",
						Verifier:    "verifier",
						RedirectURI: "http://127.0.0.1:4242/callback",
					},
				},
			}}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.Proof.Method).To(Equal(string(svcauth.MethodOIDC)))
			Expect(fwd.Proof.RedirectUri).To(Equal("http://127.0.0.1:4242/callback"))
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
		It("Should reject a proof with no variant", func(ctx SpecContext) {
			Expect(create.RequestTranslator.Forward(
				ctx,
				apisession.CreateRequest{},
			)).Error().To(MatchError(validate.ErrValidation))
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
		It("Should round-trip the token, session, and user", func(ctx SpecContext) {
			t := create.ResponseTranslator
			msg := apisession.CreateResponse{
				Token:   "tk",
				Session: session,
				User:    &bob,
			}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
		It("Should round-trip a subject that is not a user", func(ctx SpecContext) {
			t := create.ResponseTranslator
			msg := apisession.CreateResponse{Token: "tk", Session: session}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.User).To(BeNil())
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
		DescribeTable("Should reject a malformed key", func(
			ctx SpecContext,
			res *grpcauth.SessionCreateResponse,
		) {
			Expect(create.ResponseTranslator.Backward(ctx, res)).Error().
				To(MatchError(errInvalidUUID))
		},
			Entry("session", &grpcauth.SessionCreateResponse{
				Session: &grpcauth.Session{Key: "bad"},
			}),
			Entry("credential", &grpcauth.SessionCreateResponse{
				Session: &grpcauth.Session{Key: uuid.New().String(), Credential: "bad"},
			}),
			Entry("user", &grpcauth.SessionCreateResponse{
				Session: &grpcauth.Session{
					Key:        uuid.New().String(),
					Credential: uuid.New().String(),
				},
				User: &grpcauth.User{Key: "bad"},
			}),
		)
	})

	Describe("SessionRenewResponse", func() {
		It("Should round-trip the session and user", func(ctx SpecContext) {
			t := renew.ResponseTranslator
			msg := apisession.RenewResponse{Session: session, User: &bob}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
		It("Should round-trip a subject that is not a user", func(ctx SpecContext) {
			t := renew.ResponseTranslator
			msg := apisession.RenewResponse{Session: session}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.User).To(BeNil())
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
		DescribeTable("Should reject a malformed key", func(
			ctx SpecContext,
			res *grpcauth.SessionRenewResponse,
		) {
			Expect(renew.ResponseTranslator.Backward(ctx, res)).Error().
				To(MatchError(errInvalidUUID))
		},
			Entry("session", &grpcauth.SessionRenewResponse{
				Session: &grpcauth.Session{Key: "bad"},
			}),
			Entry("user", &grpcauth.SessionRenewResponse{
				Session: &grpcauth.Session{
					Key:        uuid.New().String(),
					Credential: uuid.New().String(),
				},
				User: &grpcauth.User{Key: "bad"},
			}),
		)
	})

	Describe("SessionRetrieveRequest", func() {
		It("Should round-trip the subjects", func(ctx SpecContext) {
			t := retrieve.RequestTranslator
			msg := apisession.RetrieveRequest{
				Subjects: []ontology.ID{bob.OntologyID()},
			}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
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
		It("Should round-trip the sessions", func(ctx SpecContext) {
			t := retrieve.ResponseTranslator
			msg := apisession.RetrieveResponse{Sessions: []svcauth.Session{session}}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
		It("Should reject a malformed session key", func(ctx SpecContext) {
			Expect(retrieve.ResponseTranslator.Backward(
				ctx,
				&grpcauth.SessionRetrieveResponse{
					Sessions: []*grpcauth.Session{{Key: "bad"}},
				},
			)).Error().To(MatchError(errInvalidUUID))
		})
	})

	Describe("SessionDeleteRequest", func() {
		It("Should round-trip the keys", func(ctx SpecContext) {
			t := del.RequestTranslator
			msg := apisession.DeleteRequest{Keys: []svcauth.SessionKey{uuid.New()}}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
		It("Should reject a malformed key", func(ctx SpecContext) {
			Expect(del.RequestTranslator.Backward(
				ctx,
				&grpcauth.SessionDeleteRequest{Keys: []string{"bad"}},
			)).Error().To(MatchError(errInvalidUUID))
		})
	})
})
