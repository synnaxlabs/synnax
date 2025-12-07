// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth_test

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/cmd/auth"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("ProvisionRootUser", Ordered, func() {
	var (
		tx   gorp.Tx
		dist *distribution.Layer
	)
	BeforeAll(func() {
		tx = db.OpenTx()
		dist = &distribution.Layer{DB: db}
	})
	AfterAll(func() {
		Expect(tx.Close()).To(Succeed())
	})

	Describe("New root user creation", func() {
		It("Should create a new root user with correct credentials", func() {
			creds := svcauth.InsecureCredentials{
				Username: "newroot",
				Password: "password123",
			}
			Expect(auth.ProvisionRootUser(ctx, creds, dist, svc)).To(Succeed())

			var u user.User
			Expect(svc.User.NewRetrieve().
				WhereUsernames("newroot").
				Entry(&u).
				Exec(ctx, tx)).To(Succeed())
			Expect(u.Username).To(Equal("newroot"))
			Expect(u.RootUser).To(BeTrue())
		})

		It("Should assign Owner role to newly created root user", func() {
			var u user.User
			Expect(svc.User.NewRetrieve().
				WhereUsernames("newroot").
				Entry(&u).
				Exec(ctx, tx)).To(Succeed())

			hasOwner := userHasRole(ctx, tx, user.OntologyID(u.Key), "Owner")
			Expect(hasOwner).To(BeTrue())
		})

		It("Should NOT assign Operator role to root user", func() {
			var u user.User
			Expect(svc.User.NewRetrieve().
				WhereUsernames("newroot").
				Entry(&u).
				Exec(ctx, tx)).To(Succeed())

			hasOperator := userHasRole(ctx, tx, user.OntologyID(u.Key), "Operator")
			Expect(hasOperator).To(BeFalse())
		})
	})

	Describe("Existing root user", func() {
		It("Should not create duplicate user if root user already exists", func() {
			creds := svcauth.InsecureCredentials{
				Username: "newroot",
				Password: "password123",
			}
			Expect(auth.ProvisionRootUser(ctx, creds, dist, svc)).To(Succeed())

			var users []user.User
			Expect(svc.User.NewRetrieve().
				WhereUsernames("newroot").
				Entries(&users).
				Exec(ctx, tx)).To(Succeed())
			Expect(users).To(HaveLen(1))
		})
	})

	Describe("Authentication", func() {
		It("Should allow authentication with root user credentials", func() {
			creds := svcauth.InsecureCredentials{
				Username: "newroot",
				Password: "password123",
			}
			Expect(svc.Auth.Authenticate(ctx, creds)).To(Succeed())
		})

		It("Should reject invalid password", func() {
			creds := svcauth.InsecureCredentials{
				Username: "newroot",
				Password: "wrongpassword",
			}
			Expect(svc.Auth.Authenticate(ctx, creds)).ToNot(Succeed())
		})
	})
})

func userHasRole(
	ctx context.Context,
	tx gorp.Tx,
	userID ontology.ID,
	roleName string,
) bool {
	var roles []ontology.Resource
	if err := otg.NewRetrieve().
		WhereIDs(userID).
		TraverseTo(ontology.Parents).
		WhereTypes(role.OntologyType).
		Entries(&roles).
		Exec(ctx, tx); err != nil {
		return false
	}
	for _, r := range roles {
		var rl role.Role
		if err := gorp.NewRetrieve[uuid.UUID, role.Role]().
			WhereKeys(uuid.MustParse(r.ID.Key)).
			Entry(&rl).
			Exec(ctx, tx); err != nil {
			continue
		}
		if rl.Name == roleName {
			return true
		}
	}
	return false
}
