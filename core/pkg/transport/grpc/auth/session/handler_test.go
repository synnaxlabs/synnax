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
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/auth/session"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	grpcauth "github.com/synnaxlabs/synnax/pkg/transport/grpc/auth"
	grpcsession "github.com/synnaxlabs/synnax/pkg/transport/grpc/auth/session"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	loginServer = grpc.UnaryServer[
		session.LoginRequest,
		*grpcauth.LoginRequest,
		session.LoginResponse,
		*grpcauth.LoginResponse,
	]
	createServer = grpc.UnaryServer[
		session.CreateRequest,
		*grpcauth.SessionCreateRequest,
		session.CreateResponse,
		*grpcauth.SessionCreateResponse,
	]
	renewServer = grpc.UnaryServer[
		struct{},
		*emptypb.Empty,
		session.RenewResponse,
		*grpcauth.SessionRenewResponse,
	]
	retrieveServer = grpc.UnaryServer[
		session.RetrieveRequest,
		*grpcauth.SessionRetrieveRequest,
		session.RetrieveResponse,
		*grpcauth.SessionRetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		session.DeleteRequest,
		*grpcauth.SessionDeleteRequest,
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

var (
	bob    = session.User{Key: uuid.New()}
	opened = svcauth.Session{
		Key:        uuid.New(),
		Subject:    bob.ResourceID(),
		Credential: uuid.New(),
		LastActive: telem.TimeStamp(42),
	}
)

var _ = Describe("Session", func() {
	var transport api.Transport
	bindable, ok := grpcsession.New(
		&transport,
		api.NewErrorCodec(),
	).(grpc.CompoundBindableTransport)
	Expect(ok).To(BeTrue())
	login, ok := transport.AuthLogin.(*loginServer)
	Expect(ok).To(BeTrue())
	create, ok := transport.AuthSessionCreate.(*createServer)
	Expect(ok).To(BeTrue())
	renew, ok := transport.AuthSessionRenew.(*renewServer)
	Expect(ok).To(BeTrue())
	retrieve, ok := transport.AuthSessionRetrieve.(*retrieveServer)
	Expect(ok).To(BeTrue())
	del, ok := transport.AuthSessionDelete.(*deleteServer)
	Expect(ok).To(BeTrue())

	Describe("New", func() {
		It("Should bind every session server onto the API transport", func() {
			Expect(bindable).To(HaveExactElements(
				BeIdenticalTo(login),
				BeIdenticalTo(create),
				BeIdenticalTo(renew),
				BeIdenticalTo(retrieve),
				BeIdenticalTo(del),
			))
		})
	})

	Describe("LoginRequest", func() {
		It("Should translate the credentials", func(ctx SpecContext) {
			Expect(MustSucceed(login.RequestTranslator.Backward(
				ctx,
				&grpcauth.LoginRequest{Username: "bob", Password: "p"},
			))).To(Equal(session.LoginRequest{Username: "bob", Password: "p"}))
		})
	})

	Describe("LoginResponse", func() {
		It("Should translate the token, user, and cluster info", func(
			ctx SpecContext,
		) {
			pb := MustSucceed(login.ResponseTranslator.Forward(
				ctx,
				session.LoginResponse{
					Token: "tk",
					User:  bob,
					ClusterInfo: auth.ClusterInfo{
						ClusterKey:  "cluster",
						NodeVersion: "0.1.0",
						NodeKey:     1,
						NodeTime:    telem.TimeStamp(7),
					},
				},
			))
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
			).To(Equal(session.CreateRequest{Proof: svcauth.Proof{
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
			).To(Equal(session.CreateRequest{Proof: svcauth.Proof{
				Variant: svcauth.APIKeyProof{
					APIKeyCredentials: svcauth.APIKeyCredentials{Key: "syk_secret"},
				},
			}}))
		})
		It("Should translate an OIDC proof", func(ctx SpecContext) {
			Expect(
				MustSucceed(create.RequestTranslator.Backward(
					ctx,
					&grpcauth.SessionCreateRequest{Proof: &grpcauth.Proof{
						Method:      string(svcauth.MethodOIDC),
						Provider:    "idp",
						Code:        "code",
						Verifier:    "verifier",
						RedirectUri: "http://127.0.0.1:4242/callback",
					}},
				)),
			).To(Equal(session.CreateRequest{Proof: svcauth.Proof{
				Variant: svcauth.OIDCProof{
					OIDCCredentials: svcauth.OIDCCredentials{
						Provider:    "idp",
						Code:        "code",
						Verifier:    "verifier",
						RedirectURI: "http://127.0.0.1:4242/callback",
					},
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
				session.CreateResponse{Token: "tk", Session: opened, User: &bob},
			))
			Expect(pb.Token).To(Equal("tk"))
			Expect(pb.Session).To(matchSession(opened))
			Expect(pb.User.Key).To(Equal(bob.Key.String()))
		})
		It("Should translate a subject that is not a user", func(ctx SpecContext) {
			pb := MustSucceed(create.ResponseTranslator.Forward(
				ctx,
				session.CreateResponse{Token: "tk", Session: opened},
			))
			Expect(pb.User).To(BeNil())
		})
	})

	Describe("SessionRenewResponse", func() {
		It("Should translate the session and user", func(ctx SpecContext) {
			pb := MustSucceed(renew.ResponseTranslator.Forward(
				ctx,
				session.RenewResponse{Session: opened, User: &bob},
			))
			Expect(pb.Session).To(matchSession(opened))
			Expect(pb.User.Key).To(Equal(bob.Key.String()))
		})
		It("Should translate a subject that is not a user", func(ctx SpecContext) {
			pb := MustSucceed(renew.ResponseTranslator.Forward(
				ctx,
				session.RenewResponse{Session: opened},
			))
			Expect(pb.User).To(BeNil())
		})
	})

	Describe("SessionRetrieveRequest", func() {
		It("Should translate the subjects", func(ctx SpecContext) {
			id := bob.ResourceID()
			Expect(MustSucceed(retrieve.RequestTranslator.Backward(
				ctx,
				&grpcauth.SessionRetrieveRequest{Subjects: []*grpcauth.Subject{
					{Type: string(id.Type), Key: id.Key},
				}},
			)).Subjects).To(Equal([]resource.ID{id}))
		})
		It("Should read a missing subject as the zero ID", func(ctx SpecContext) {
			res := MustSucceed(retrieve.RequestTranslator.Backward(
				ctx,
				&grpcauth.SessionRetrieveRequest{Subjects: []*grpcauth.Subject{nil}},
			))
			Expect(res.Subjects).To(Equal([]resource.ID{{}}))
		})
	})

	Describe("SessionRetrieveResponse", func() {
		It("Should translate the sessions", func(ctx SpecContext) {
			pb := MustSucceed(retrieve.ResponseTranslator.Forward(
				ctx,
				session.RetrieveResponse{Sessions: []svcauth.Session{opened}},
			))
			Expect(pb.Sessions).To(ConsistOf(matchSession(opened)))
		})
	})

	Describe("SessionDeleteRequest", func() {
		It("Should translate the keys", func(ctx SpecContext) {
			key := uuid.New()
			Expect(MustSucceed(del.RequestTranslator.Backward(
				ctx,
				&grpcauth.SessionDeleteRequest{Keys: []string{key.String()}},
			))).To(Equal(session.DeleteRequest{Keys: []svcauth.SessionKey{key}}))
		})
		It("Should reject a malformed key", func(ctx SpecContext) {
			Expect(del.RequestTranslator.Backward(
				ctx,
				&grpcauth.SessionDeleteRequest{Keys: []string{"bad"}},
			)).Error().To(MatchError(errInvalidUUID))
		})
	})
})
