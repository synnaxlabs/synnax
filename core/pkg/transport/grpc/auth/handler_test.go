// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

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
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var (
	bob     = user.User{Key: uuid.New(), Username: "bob"}
	session = svcauth.Session{
		Key:        uuid.New(),
		Subject:    bob.OntologyID(),
		Credential: uuid.New(),
		LastActive: telem.TimeStamp(42),
	}
)

var _ = Describe("gRPC Auth Translators", func() {
	Describe("LoginRequest", func() {
		t := loginRequestTranslator{}
		It("Should round-trip the credentials", func(ctx SpecContext) {
			msg := auth.LoginRequest{Username: "bob", Password: "p"}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.Username).To(Equal("bob"))
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
	})

	Describe("LoginResponse", func() {
		t := loginResponseTranslator{}
		It("Should round-trip the token, user, and cluster info", func(
			ctx SpecContext,
		) {
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
			Expect(t.Backward(ctx, &LoginResponse{
				User:        &User{Key: "bad"},
				ClusterInfo: &ClusterInfo{},
			})).Error().To(HaveOccurred())
		})
	})

	Describe("SessionCreateRequest", func() {
		t := sessionCreateRequestTranslator{}
		It("Should round-trip a password proof", func(ctx SpecContext) {
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
			Expect(t.Forward(ctx, auth.SessionCreateRequest{})).Error().
				To(MatchError(validate.ErrValidation))
		})
		It("Should reject a missing proof", func(ctx SpecContext) {
			Expect(t.Backward(ctx, &SessionCreateRequest{})).Error().
				To(MatchError(validate.ErrValidation))
		})
		It("Should reject an unknown method", func(ctx SpecContext) {
			Expect(t.Backward(ctx, &SessionCreateRequest{
				Proof: &Proof{Method: "carrier_pigeon"},
			})).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("SessionCreateResponse", func() {
		t := sessionCreateResponseTranslator{}
		It("Should round-trip the token, session, and user", func(ctx SpecContext) {
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
			msg := auth.SessionCreateResponse{Token: "tk", Session: session}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.User).To(BeNil())
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
		DescribeTable("Should reject a malformed key", func(
			ctx SpecContext,
			res *SessionCreateResponse,
		) {
			Expect(t.Backward(ctx, res)).Error().To(HaveOccurred())
		},
			Entry("session", &SessionCreateResponse{Session: &Session{Key: "bad"}}),
			Entry("credential", &SessionCreateResponse{Session: &Session{
				Key:        uuid.New().String(),
				Credential: "bad",
			}}),
			Entry("user", &SessionCreateResponse{
				Session: sessionForward(session),
				User:    &User{Key: "bad"},
			}),
		)
	})

	Describe("SessionRetrieveRequest", func() {
		t := sessionRetrieveRequestTranslator{}
		It("Should round-trip the subjects", func(ctx SpecContext) {
			msg := auth.SessionRetrieveRequest{
				Subjects: []ontology.ID{bob.OntologyID()},
			}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
		It("Should read a missing subject as the zero ID", func(ctx SpecContext) {
			res := MustSucceed(t.Backward(ctx, &SessionRetrieveRequest{
				Subjects: []*Subject{nil},
			}))
			Expect(res.Subjects).To(Equal([]ontology.ID{{}}))
		})
	})

	Describe("SessionRetrieveResponse", func() {
		t := sessionRetrieveResponseTranslator{}
		It("Should round-trip the sessions", func(ctx SpecContext) {
			msg := auth.SessionRetrieveResponse{Sessions: []svcauth.Session{session}}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
		It("Should reject a malformed session key", func(ctx SpecContext) {
			Expect(t.Backward(ctx, &SessionRetrieveResponse{
				Sessions: []*Session{{Key: "bad"}},
			})).Error().To(HaveOccurred())
		})
	})

	Describe("SessionDeleteRequest", func() {
		t := sessionDeleteRequestTranslator{}
		It("Should round-trip the keys", func(ctx SpecContext) {
			msg := auth.SessionDeleteRequest{Keys: []svcauth.SessionKey{uuid.New()}}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
		It("Should reject a malformed key", func(ctx SpecContext) {
			Expect(t.Backward(ctx, &SessionDeleteRequest{Keys: []string{"bad"}})).
				Error().To(HaveOccurred())
		})
	})

	Describe("New", func() {
		It("Should bind every auth server onto the API transport", func() {
			var transport api.Transport
			bindable := New(&transport).(grpc.CompoundBindableTransport)
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
})
