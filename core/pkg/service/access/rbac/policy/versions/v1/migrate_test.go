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
	v1 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("WorkspaceObjectsMigration", func() {
	// A v0.56 policy could name workspaces, which v0.57 turned into projects under the
	// same keys.
	It("Should point policy objects that name workspaces at projects", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		p := v1.Policy{
			Key:  uuid.New(),
			Name: "custom",
			Objects: []ontology.ID{
				{Type: "workspace", Key: "a1b2"},
				{Type: "workspace"},
				{Type: ontology.ResourceTypeChannel, Key: "1"},
			},
		}
		Expect(gorp.NewCreate[v1.Key, v1.Policy]().Entry(&p).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Policy",
			Migrations: []gorp.Migration{v1.WorkspaceObjectsMigration},
		})).To(Succeed())
		var got v1.Policy
		Expect(gorp.NewRetrieve[v1.Key, v1.Policy]().
			Where(gorp.MatchKeys[v1.Key, v1.Policy](p.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Objects).To(Equal([]ontology.ID{
			{Type: ontology.ResourceTypeProject, Key: "a1b2"},
			{Type: ontology.ResourceTypeProject},
			{Type: ontology.ResourceTypeChannel, Key: "1"},
		}))
	})
})

var _ = Describe("NameMigration", func() {
	migrated := func(ctx SpecContext, in v1.Policy) v1.Policy {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		Expect(gorp.NewCreate[v1.Key, v1.Policy]().Entry(&in).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Policy",
			Migrations: []gorp.Migration{v1.NameMigration},
		})).To(Succeed())
		var out v1.Policy
		Expect(gorp.NewRetrieve[v1.Key, v1.Policy]().
			Where(gorp.MatchKeys[v1.Key, v1.Policy](in.Key)).
			Entry(&out).Exec(ctx, db)).To(Succeed())
		return out
	}
	key := uuid.MustParse("1a2b3c4d-0000-0000-0000-000000000000")
	DescribeTable(
		"Should name only nameless policys",
		func(ctx SpecContext, name, expected string) {
			Expect(migrated(ctx, v1.Policy{Key: key, Name: name}).Name).
				To(Equal(expected))
		},
		Entry("empty name", "", "Policy 1a2b3c4d"),
		Entry("named policy", "Named", "Named"),
	)
})
