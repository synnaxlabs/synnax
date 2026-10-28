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

var _ = Describe("Auth", func() {
	var (
		transport api.Transport
		bindable  = grpcauth.New(&transport).(grpc.CompoundBindableTransport)
		login     = transport.AuthLogin.(*loginServer)
		create    = transport.AuthSessionCreate.(*sessionCreateServer)
		renew     = transport.AuthSessionRenew.(*sessionRenewServer)
		retrieve  = transport.AuthSessionRetrieve.(*sessionRetrieveServer)
		del       = transport.AuthSessionDelete.(*sessionDeleteServer)
	)

	Describe("New", func() {
		It("Should bind every auth server onto the API transport", func() {
			Expect(bindable).To(HaveLen(5))
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
		})
	})

	Describe("LoginRequest", func() {
		It("Should round-trip the credentials", func(ctx SpecContext) {
			t := login.RequestTranslator
			msg := auth.LoginRequest{Username: "bob", Password: "p"}
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
			msg := auth.LoginResponse{
				Token: "tk",
				User:  bob,
				ClusterInfo: auth.ClusterInfo{
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
			msg := auth.SessionCreateRequest{Proof: svcauth.Proof{
				Variant: svcauth.PasswordProof{
					Credentials: svcauth.Credentials{Username: "bob", Password: "p"},
				},
			}}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.Proof.Method).To(Equal(string(svcauth.MethodPassword)))
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
		It("Should reject a proof with no variant", func(ctx SpecContext) {
			Expect(create.RequestTranslator.Forward(
				ctx,
				auth.SessionCreateRequest{},
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
			msg := auth.SessionCreateResponse{
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
			msg := auth.SessionCreateResponse{Token: "tk", Session: session}
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
			msg := auth.SessionRenewResponse{Session: session, User: &bob}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
		It("Should round-trip a subject that is not a user", func(ctx SpecContext) {
			t := renew.ResponseTranslator
			msg := auth.SessionRenewResponse{Session: session}
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
			msg := auth.SessionRetrieveRequest{
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
			msg := auth.SessionRetrieveResponse{Sessions: []svcauth.Session{session}}
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
			msg := auth.SessionDeleteRequest{Keys: []svcauth.SessionKey{uuid.New()}}
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
