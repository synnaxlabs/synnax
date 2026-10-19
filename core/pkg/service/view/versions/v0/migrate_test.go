// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v0"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("NameMigration", func() {
	migrated := func(ctx SpecContext, in v0.View) v0.View {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		Expect(gorp.NewCreate[v0.Key, v0.View]().Entry(&in).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "View",
			Migrations: []migrate.Migration{v0.NameMigration},
		})).To(Succeed())
		var out v0.View
		Expect(gorp.NewRetrieve[v0.Key, v0.View]().
			Where(gorp.MatchKeys[v0.Key, v0.View](in.Key)).
			Entry(&out).Exec(ctx, db)).To(Succeed())
		return out
	}
	It("Should name a view that has an empty name", func(ctx SpecContext) {
		in := v0.View{Key: uuid.New()}
		Expect(migrated(ctx, in).Name).To(Equal("View " + in.Key.String()[:8]))
	})
	It("Should keep the name of a named view", func(ctx SpecContext) {
		in := v0.View{Key: uuid.New(), Name: "Named"}
		Expect(migrated(ctx, in).Name).To(Equal("Named"))
	})
})
