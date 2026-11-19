// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apirole "github.com/synnaxlabs/synnax/pkg/api/access/role"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Restrict", func() {
	createRole := func(ctx SpecContext, name string) role.Role {
		GinkgoHelper()
		r := role.Role{Name: name}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.RBAC.Role.NewWriter(tx, false).Create(ctx, &r)
		})).To(Succeed())
		return r
	}
	roleExists := func(ctx SpecContext, key role.Key) bool {
		GinkgoHelper()
		return MustSucceed(svc.RBAC.Role.NewRetrieve().
			Where(role.MatchKeys(key)).
			Exists(ctx, db))
	}

	It("Should name the records of a delete that a reference refuses", func(
		ctx SpecContext,
	) {
		r := createRole(ctx, "Operator")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := svc.User.NewWriter(tx).Create(ctx, user.User{
				Name: "Holder",
				Role: r.Key,
			})
			return err
		})).To(Succeed())
		Expect(post(
			"/api/v1/access/role/delete",
			apirole.DeleteRequest{Keys: []role.Key{r.Key}},
			nil,
		)).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"cannot delete role Operator because user Holder uses it as its role",
			)),
		))
		Expect(roleExists(ctx, r.Key)).To(BeTrue())
	})

	It("Should delete a record that no reference holds", func(ctx SpecContext) {
		r := createRole(ctx, "Unused")
		Expect(post(
			"/api/v1/access/role/delete",
			apirole.DeleteRequest{Keys: []role.Key{r.Key}},
			nil,
		)).To(Succeed())
		Expect(roleExists(ctx, r.Key)).To(BeFalse())
	})

	It("Should keep an error that holds no refused delete", func() {
		// A batch.Payload does not encode, so the body is a map.
		Expect(post("/api/v1/batch", map[string]any{
			"requests": []map[string]any{
				{"endpoint": "/does-not-exist", "payload": nil},
			},
		}, nil)).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`"/does-not-exist" cannot run in a batch`)),
		))
	})
})
