// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user

import (
	"go/types"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	Describe("Delete", func() {
		It("Should be a no-op when none of the supplied keys exist", func(ctx SpecContext) {
			Expect(apiSvc.Delete(rootCtx(ctx), DeleteRequest{Keys: []user.Key{uuid.New()}})).To(Equal(types.Nil{}))
		})
		It("Should delete existing users and ignore unknown keys in the same call", func(ctx SpecContext) {
			username := uuid.NewString()
			created := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{Username: username}))
			Expect(authSvc.NewWriter(nil).Register(ctx, auth.Credentials{
				Username: username,
				Password: "password",
			})).To(Succeed())
			Expect(apiSvc.Delete(
				rootCtx(ctx),
				DeleteRequest{Keys: []user.Key{created.Key, uuid.New()}},
			)).To(Equal(types.Nil{}))
			Expect(userSvc.NewRetrieve().Where(user.MatchKeys(created.Key)).Exists(ctx, nil)).
				To(BeFalse())
			Expect(authSvc.Authenticate(ctx, nil, auth.Credentials{
				Username: username,
				Password: "password",
			})).To(MatchError(auth.ErrInvalidCredentials))
		})
	})
})
