// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("v0 -> v1 User migration", func() {
	migrateUser := func(ctx SpecContext, old v0.User) v1.User {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		Expect(gorp.WrapWriter[v0.Key, v0.User](db).Set(ctx, old)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "User",
			Migrations: []gorp.Migration{v1.Migration},
		})).To(Succeed())
		var u v1.User
		Expect(gorp.NewRetrieve[v1.Key, v1.User]().
			Where(gorp.MatchKeys[v1.Key, v1.User](old.Key)).
			Entry(&u).
			Exec(ctx, db)).To(Succeed())
		return u
	}
	DescribeTable("Should name each user", func(
		ctx SpecContext,
		first, last, name string,
	) {
		u := migrateUser(ctx, v0.User{
			Key:       uuid.New(),
			Username:  "jdoe",
			FirstName: first,
			LastName:  last,
		})
		Expect(u.Name).To(Equal(name))
	},
		Entry("from both names", "Jane", "Doe", "Jane Doe"),
		Entry("from the first name alone", "Jane", "", "Jane"),
		Entry("from the last name alone", "", "Doe", "Doe"),
		Entry("from the username when both names are empty", "", "", "jdoe"),
	)
	It("Should name a user with no names or username", func(ctx SpecContext) {
		Expect(migrateUser(ctx, v0.User{Key: uuid.New()}).Name).
			To(Equal("Unnamed user"))
	})
	It("Should keep the key and root flag", func(ctx SpecContext) {
		key := uuid.New()
		u := migrateUser(ctx, v0.User{Key: key, Username: "root", RootUser: true})
		Expect(u.Key).To(Equal(key))
		Expect(u.RootUser).To(BeTrue())
	})
})
