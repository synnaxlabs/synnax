// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project_test

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	projectv56 "github.com/synnaxlabs/synnax/pkg/service/project/migrations/v56"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Workspace to project migration", func() {
	migrations := []migrate.Migration{
		gorp.CodecMigration[project.Key, projectv56.Workspace]("msgpack_to_orc"),
		migrate.WithAddedDeps(
			gorp.NewMigration(
				"v56_migrate_workspace_to_project",
				project.MigrateWorkspaceToProject,
			),
			"msgpack_to_orc",
		),
	}
	openProjectTable := func(ctx context.Context, db *gorp.DB) *gorp.Table[project.Key, project.Project] {
		return MustOpen(gorp.OpenTable[project.Key, project.Project](
			ctx, gorp.TableConfig[project.Key, project.Project]{DB: db, Migrations: migrations},
		))
	}
	rel := func(from, to ontology.ID) ontology.Relationship {
		return ontology.Relationship{From: from, Type: ontology.RelationshipTypeParentOf, To: to}
	}
	hasRel := func(ctx context.Context, db *gorp.DB, r ontology.Relationship) error {
		return MustOpen(gorp.OpenTable[string, ontology.Relationship](
			ctx, gorp.TableConfig[string, ontology.Relationship]{DB: db},
		)).NewRetrieve().Where(gorp.MatchKeys[string, ontology.Relationship](r.GorpKey())).
			Entry(&ontology.Relationship{}).Exec(ctx, db)
	}

	It("Should lift workspaces, repoint the ontology, and rename the group", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		// Seed two legacy workspace records under the "Workspace" gorp prefix.
		wsA, wsB := uuid.New(), uuid.New()
		wsTable := MustOpen(gorp.OpenTable[projectv56.Key, projectv56.Workspace](
			ctx, gorp.TableConfig[projectv56.Key, projectv56.Workspace]{DB: db},
		))
		seedA := projectv56.Workspace{
			Key:    wsA,
			Name:   "Ops",
			Layout: msgpack.EncodedJSON{"mosaic": "tree"},
		}
		Expect(wsTable.NewCreate().Entry(&seedA).Exec(ctx, db)).To(Succeed())
		seedB := projectv56.Workspace{Key: wsB, Name: "Analysis"}
		Expect(wsTable.NewCreate().Entry(&seedB).Exec(ctx, db)).To(Succeed())

		// Seed the root "Workspaces" group and the ontology nodes.
		groupKey := uuid.New()
		groupTable := MustOpen(gorp.OpenTable[group.Key, group.Group](
			ctx, gorp.TableConfig[group.Key, group.Group]{DB: db},
		))
		Expect(groupTable.NewCreate().Entry(&group.Group{Key: groupKey, Name: "Workspaces"}).
			Exec(ctx, db)).To(Succeed())
		groupID := group.OntologyID(groupKey)
		wsAID := ontology.ID{Type: ontology.ResourceType("workspace"), Key: wsA.String()}
		wsBID := ontology.ID{Type: ontology.ResourceType("workspace"), Key: wsB.String()}
		lpID := ontology.ID{Type: ontology.ResourceTypeLineplot, Key: uuid.NewString()}
		resTable := MustOpen(gorp.OpenTable[string, ontology.Resource](
			ctx, gorp.TableConfig[string, ontology.Resource]{DB: db},
		))
		for _, id := range []ontology.ID{wsAID, wsBID, lpID} {
			Expect(resTable.NewCreate().Entry(&ontology.Resource{ID: id}).Exec(ctx, db)).To(Succeed())
		}

		// Seed relationships: group -> each workspace, workspace -> line plot, and an
		// unrelated group -> line plot that must be left untouched.
		relTable := MustOpen(gorp.OpenTable[string, ontology.Relationship](
			ctx, gorp.TableConfig[string, ontology.Relationship]{DB: db},
		))
		groupToWSA := rel(groupID, wsAID)
		groupToWSB := rel(groupID, wsBID)
		wsAToLP := rel(wsAID, lpID)
		groupToLP := rel(groupID, lpID)
		for _, r := range []ontology.Relationship{groupToWSA, groupToWSB, wsAToLP, groupToLP} {
			rr := r
			Expect(relTable.NewCreate().Entry(&rr).Exec(ctx, db)).To(Succeed())
		}

		projectTable := openProjectTable(ctx, db)

		By("Lifting every workspace into a project with identical fields")
		var pA project.Project
		Expect(projectTable.NewRetrieve().Where(gorp.MatchKeys[project.Key, project.Project](wsA)).
			Entry(&pA).Exec(ctx, db)).To(Succeed())
		Expect(pA).To(Equal(project.Project{
			Key:    wsA,
			Name:   "Ops",
			Layout: msgpack.EncodedJSON{"mosaic": "tree"},
		}))
		Expect(projectTable.NewRetrieve().Where(gorp.MatchKeys[project.Key, project.Project](wsB)).
			Exists(ctx, db)).To(BeTrue())

		By("Removing the legacy workspace records")
		Expect(wsTable.NewRetrieve().Where(gorp.MatchKeys[projectv56.Key, projectv56.Workspace](wsA, wsB)).
			Entries(&[]projectv56.Workspace{}).Exec(ctx, db)).To(MatchError(query.ErrNotFound))

		By("Re-keying the workspace resource nodes to project")
		projectAID := ontology.ID{Type: ontology.ResourceTypeProject, Key: wsA.String()}
		Expect(resTable.NewRetrieve().Where(gorp.MatchKeys[string, ontology.Resource](projectAID.String())).
			Exists(ctx, db)).To(BeTrue())
		Expect(resTable.NewRetrieve().Where(gorp.MatchKeys[string, ontology.Resource](wsAID.String())).
			Exists(ctx, db)).To(BeFalse())

		By("Repointing workspace relationships and leaving unrelated ones intact")
		projectBID := ontology.ID{Type: ontology.ResourceTypeProject, Key: wsB.String()}
		Expect(hasRel(ctx, db, rel(groupID, projectAID))).To(Succeed())
		Expect(hasRel(ctx, db, rel(groupID, projectBID))).To(Succeed())
		Expect(hasRel(ctx, db, rel(projectAID, lpID))).To(Succeed())
		Expect(hasRel(ctx, db, groupToWSA)).To(MatchError(query.ErrNotFound))
		Expect(hasRel(ctx, db, wsAToLP)).To(MatchError(query.ErrNotFound))
		Expect(hasRel(ctx, db, groupToLP)).To(Succeed())

		By("Renaming the Workspaces group to Projects")
		var g group.Group
		Expect(groupTable.NewRetrieve().Where(gorp.MatchKeys[group.Key, group.Group](groupKey)).
			Entry(&g).Exec(ctx, db)).To(Succeed())
		Expect(g.Name).To(Equal("Projects"))
	})

	It("Should be a no-op on a cluster that never created a workspace", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		// Opening the table runs the migration; it must succeed with nothing to lift.
		Expect(openProjectTable(ctx, db).NewRetrieve().
			Where(gorp.MatchKeys[project.Key, project.Project](uuid.New())).
			Exists(ctx, db)).To(BeFalse())
	})
})

var _ = Describe("Remove author relationships migration", func() {
	parentOf := func(from, to ontology.ID) ontology.Relationship {
		return ontology.Relationship{From: from, Type: ontology.RelationshipTypeParentOf, To: to}
	}
	hasRel := func(ctx context.Context, db *gorp.DB, r ontology.Relationship) error {
		return MustOpen(gorp.OpenTable[string, ontology.Relationship](
			ctx, gorp.TableConfig[string, ontology.Relationship]{DB: db},
		)).NewRetrieve().Where(gorp.MatchKeys[string, ontology.Relationship](r.GorpKey())).
			Entry(&ontology.Relationship{}).Exec(ctx, db)
	}

	It("Should delete user-to-project parent relationships and leave others intact", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		relTable := MustOpen(gorp.OpenTable[string, ontology.Relationship](
			ctx, gorp.TableConfig[string, ontology.Relationship]{DB: db},
		))
		projectID := ontology.ID{Type: ontology.ResourceTypeProject, Key: uuid.NewString()}
		userID := ontology.ID{Type: ontology.ResourceTypeUser, Key: uuid.NewString()}
		groupID := group.OntologyID(uuid.New())
		lpID := ontology.ID{Type: ontology.ResourceTypeLineplot, Key: uuid.NewString()}

		authorToProject := parentOf(userID, projectID)
		groupToProject := parentOf(groupID, projectID)
		projectToLP := parentOf(projectID, lpID)
		for _, r := range []ontology.Relationship{authorToProject, groupToProject, projectToLP} {
			rr := r
			Expect(relTable.NewCreate().Entry(&rr).Exec(ctx, db)).To(Succeed())
		}

		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		tx := db.OpenTx()
		Expect(project.RemoveAuthorRelationships(ctx, tx, otg)).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())

		By("Deleting the author user-to-project relationship")
		Expect(hasRel(ctx, db, authorToProject)).To(MatchError(query.ErrNotFound))

		By("Leaving the group parent and project children untouched")
		Expect(hasRel(ctx, db, groupToProject)).To(Succeed())
		Expect(hasRel(ctx, db, projectToLP)).To(Succeed())
	})

	It("Should be a no-op when no author relationships exist", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		tx := db.OpenTx()
		Expect(project.RemoveAuthorRelationships(ctx, tx, otg)).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
	})
})

var _ = Describe("Project layout staging migration", func() {
	It("Should stage each non-empty layout under its key and skip empty ones", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		table := MustOpen(gorp.OpenTable[project.Key, project.Project](
			ctx, gorp.TableConfig[project.Key, project.Project]{DB: db},
		))
		withLayout := uuid.New()
		layout := msgpack.EncodedJSON{"active": "plot-1", "pinned": true}
		Expect(table.NewCreate().Entry(&project.Project{
			Key: withLayout, Name: "Ops", Layout: layout,
		}).Exec(ctx, db)).To(Succeed())
		empty := uuid.New()
		Expect(table.NewCreate().Entry(&project.Project{
			Key: empty, Name: "Empty",
		}).Exec(ctx, db)).To(Succeed())

		tx := db.OpenTx()
		Expect(project.MigrateLayoutsToStaging(ctx, tx, alamos.Instrumentation{})).
			To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())

		By("Writing the layout blob under the project's staging key")
		blob, closer := MustSucceed2(db.Get(ctx, project.LegacyLayoutKVKey(withLayout)))
		var got msgpack.EncodedJSON
		Expect(json.Unmarshal(blob, &got)).To(Succeed())
		Expect(closer.Close()).To(Succeed())
		Expect(got).To(Equal(layout))

		By("Leaving no staging entry for a project without a layout")
		Expect(db.Get(ctx, project.LegacyLayoutKVKey(empty))).Error().
			To(MatchError(query.ErrNotFound))
	})
})
