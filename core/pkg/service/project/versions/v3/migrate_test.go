// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("NameMigration", func() {
	migrated := func(ctx SpecContext, in v3.Project) v3.Project {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		Expect(gorp.NewCreate[v3.Key, v3.Project]().Entry(&in).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Project",
			Migrations: []gorp.Migration{v3.NameMigration},
		})).To(Succeed())
		var out v3.Project
		Expect(gorp.NewRetrieve[v3.Key, v3.Project]().
			Where(gorp.MatchKeys[v3.Key, v3.Project](in.Key)).
			Entry(&out).Exec(ctx, db)).To(Succeed())
		return out
	}
	key := uuid.MustParse("1a2b3c4d-0000-0000-0000-000000000000")
	DescribeTable(
		"Should name only nameless projects",
		func(ctx SpecContext, name, expected string) {
			Expect(migrated(ctx, v3.Project{Key: key, Name: name}).Name).
				To(Equal(expected))
		},
		Entry("empty name", "", "Project 1a2b3c4d"),
		Entry("named project", "Named", "Named"),
	)
})
