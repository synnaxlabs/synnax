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
	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
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
			Migrations: []gorp.Migration{v0.NameMigration},
		})).To(Succeed())
		var out v0.View
		Expect(gorp.NewRetrieve[v0.Key, v0.View]().
			Where(gorp.MatchKeys[v0.Key, v0.View](in.Key)).
			Entry(&out).Exec(ctx, db)).To(Succeed())
		return out
	}
	key := uuid.MustParse("1a2b3c4d-0000-0000-0000-000000000000")
	DescribeTable(
		"Should name only nameless views",
		func(ctx SpecContext, name, expected string) {
			Expect(migrated(ctx, v0.View{Key: key, Name: name}).Name).
				To(Equal(expected))
		},
		Entry("empty name", "", "View 1a2b3c4d"),
		Entry("named view", "Named", "Named"),
	)
})
