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
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("Service", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			var err error
			if u, err = userSvc.NewWriter(tx).Create(ctx, user.User{
				Username: "change-password-" + uuid.New().String(),
			}); err != nil {
				return err
			}
			return authSvc.NewWriter(tx).Register(ctx, u.Key, "old")
		})).To(Succeed())
	})

	changePassword := func(
		ctx SpecContext,
		username, password string,
	) error {
		GinkgoHelper()
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := apiSvc.ChangePassword(ctx, tx, apiauth.ChangePasswordRequest{
				Username:    username,
				Password:    password,
				NewPassword: "new",
			})
			return err
		})
	}

	Describe("ChangePassword", func() {
		It("Should replace the password of the named user", func(ctx SpecContext) {
			Expect(changePassword(ctx, u.Username, "old")).To(Succeed())
			Expect(authSvc.Authenticate(ctx, nil, u.Key, "new")).To(Succeed())
		})
		It(
			"Should return ErrInvalidCredentials when the password is wrong",
			func(ctx SpecContext) {
				Expect(changePassword(ctx, u.Username, "wrong")).
					To(MatchError(auth.ErrInvalidCredentials))
				Expect(authSvc.Authenticate(ctx, nil, u.Key, "old")).To(Succeed())
			},
		)
		It(
			"Should return ErrInvalidCredentials when no user holds the username",
			func(ctx SpecContext) {
				Expect(changePassword(ctx, "unknown-"+uuid.New().String(), "old")).
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
	})
})
