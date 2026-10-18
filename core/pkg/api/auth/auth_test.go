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
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	apicfg "github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("NewService", func() {
	It("Should return an error when the config is invalid", func() {
		Expect(apiauth.NewService(apicfg.LayerConfig{})).Error().
			To(MatchError(ContainSubstring("service: must be non-nil")))
	})
})

var _ = Describe("Login", func() {
	It(
		"Should return a token for the user and the cluster info",
		func(ctx SpecContext) {
			res := MustSucceed(
				apiSvc.Login(ctx, apiauth.LoginRequest{Credentials: creds}),
			)
			Expect(res.User).To(Equal(root))
			Expect(MustSucceed(tokenSvc.Validate(res.Token))).To(Equal(root.Key))
			Expect(res.ClusterInfo.ClusterKey).To(Equal(node.Cluster.Key().String()))
			Expect(res.ClusterInfo.NodeKey).To(Equal(node.Cluster.HostKey()))
			Expect(res.ClusterInfo.NodeTime).ToNot(BeZero())
		},
	)
	It("Should reject a wrong password", func(ctx SpecContext) {
		Expect(apiSvc.Login(ctx, apiauth.LoginRequest{Credentials: {
			Username: creds.Username,
			Password: "wrong",
		}})).Error().To(MatchError(auth.ErrInvalidCredentials))
	})
	It("Should reject an unknown username", func(ctx SpecContext) {
		Expect(apiSvc.Login(ctx, apiauth.LoginRequest{Credentials: {
			Username: "nobody",
			Password: creds.Password,
		}})).Error().To(MatchError(auth.ErrInvalidCredentials))
	})
})

var _ = Describe("ChangePassword", func() {
	var own auth.Credentials
	BeforeEach(func(ctx SpecContext) {
		own = auth.Credentials{Username: uuid.New().String(), Password: "old"}
		u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
			Username: own.Username,
		}))
		Expect(passwordSvc.NewWriter(nil).Set(ctx, u.OntologyID(), own)).To(Succeed())
	})
	It(
		"Should replace the password when the current one is right",
		func(ctx SpecContext) {
			Expect(apiSvc.ChangePassword(ctx, nil, apiauth.ChangePasswordRequest{
				Credentials: own,
				NewPassword: "new",
			})).Error().ToNot(HaveOccurred())
			MustSucceed(
				apiSvc.Login(ctx, apiauth.LoginRequest{Credentials: {
					Username: own.Username,
					Password: "new",
				}}),
			)
			Expect(apiSvc.Login(ctx, apiauth.LoginRequest{Credentials: own})).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		},
	)
	It("Should keep the password when the current one is wrong", func(ctx SpecContext) {
		Expect(apiSvc.ChangePassword(ctx, nil, apiauth.ChangePasswordRequest{
			Credentials: auth.Credentials{Username: own.Username, Password: "wrong"},
			NewPassword: "new",
		})).Error().To(MatchError(auth.ErrInvalidCredentials))
		MustSucceed(apiSvc.Login(ctx, apiauth.LoginRequest{Credentials: own}))
	})
})
