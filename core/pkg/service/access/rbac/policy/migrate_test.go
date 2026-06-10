// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Workspace policy object migration", func() {
	It("Should retype workspace objects to project and leave other objects intact", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		wsKey := uuid.NewString()
		lpID := ontology.ID{Type: ontology.ResourceTypeLineplot, Key: uuid.NewString()}
		wsID := ontology.ID{Type: ontology.ResourceType("workspace"), Key: wsKey}
		// A policy granting access to a specific workspace plus an unrelated line plot.
		mixed := policy.Policy{Key: uuid.New(), Objects: []ontology.ID{wsID, lpID}}
		// A control policy that references no workspace and must be left untouched.
		control := policy.Policy{Key: uuid.New(), Objects: []ontology.ID{lpID}}

		seed := MustOpen(gorp.OpenTable[policy.Key, policy.Policy](
			ctx, gorp.TableConfig[policy.Key, policy.Policy]{DB: db},
		))
		Expect(seed.NewCreate().Entry(&mixed).Exec(ctx, db)).To(Succeed())
		Expect(seed.NewCreate().Entry(&control).Exec(ctx, db)).To(Succeed())

		table := MustOpen(gorp.OpenTable[policy.Key, policy.Policy](
			ctx, gorp.TableConfig[policy.Key, policy.Policy]{
				DB: db,
				Migrations: []migrate.Migration{
					gorp.CodecMigration[policy.Key, policy.Policy]("msgpack_to_orc"),
					migrate.WithAddedDeps(
						gorp.NewMigration(
							"v56_rewrite_workspace_policy_objects",
							policy.MigrateWorkspaceObjects,
						),
						"msgpack_to_orc",
					),
				},
			},
		))

		By("Retyping the workspace object to project")
		projectID := ontology.ID{Type: ontology.ResourceTypeProject, Key: wsKey}
		var got policy.Policy
		Expect(table.NewRetrieve().Where(gorp.MatchKeys[policy.Key, policy.Policy](mixed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Objects).To(ConsistOf(projectID, lpID))

		By("Leaving the control policy untouched")
		var gotControl policy.Policy
		Expect(table.NewRetrieve().Where(gorp.MatchKeys[policy.Key, policy.Policy](control.Key)).
			Entry(&gotControl).Exec(ctx, db)).To(Succeed())
		Expect(gotControl.Objects).To(ConsistOf(lpID))
	})
})
