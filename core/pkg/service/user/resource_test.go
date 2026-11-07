// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each user by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		u := MustSucceed(svc.NewWriter(tx).Create(ctx, user.User{Name: "Patrick Star"}))
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{u.ResourceID()})).
			To(Equal([]resource.Resource{{ID: u.ResourceID(), Name: "Patrick Star"}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypeUser, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid user key "pump"`)),
		))
	})

	It("Should leave out an ID that names no user", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		u := MustSucceed(svc.NewWriter(tx).Create(ctx, user.User{Name: "Sandy Cheeks"}))
		missing := resource.ID{Type: resource.TypeUser, Key: uuid.New().String()}
		Expect(svc.Resources().Retrieve(
			ctx,
			tx,
			[]resource.ID{missing, u.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: u.ResourceID(), Name: u.Name}}))
	})

	It("Should list each user by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		u := MustSucceed(svc.NewWriter(tx).Create(ctx, user.User{Name: "Sandy Cheeks"}))
		Expect(svc.Resources().List(ctx, tx, 10000, nil)).To(ContainElement(
			resource.Resource{ID: u.ResourceID(), Name: u.Name},
		))
	})
})
