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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	It("Should lift workspaces into projects and repoint the ontology", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		// Seed a legacy workspace record under the "Workspace" gorp prefix.
		wsKey, authorKey := uuid.New(), uuid.New()
		wsTable := MustOpen(gorp.OpenTable[projectv56.Key, projectv56.Workspace](
			ctx, gorp.TableConfig[projectv56.Key, projectv56.Workspace]{DB: db},
		))
		seed := projectv56.Workspace{
			Key:    wsKey,
			Name:   "Ops",
			Author: authorKey,
			Layout: msgpack.EncodedJSON{"mosaic": "tree"},
		}
		Expect(wsTable.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())

		// Seed ontology nodes: the workspace and an unrelated line plot.
		wsID := ontology.ID{Type: ontology.ResourceType("workspace"), Key: wsKey.String()}
		lpID := ontology.ID{Type: ontology.ResourceTypeLineplot, Key: uuid.NewString()}
		groupID := ontology.ID{Type: ontology.ResourceTypeGroup, Key: uuid.NewString()}
		resTable := MustOpen(gorp.OpenTable[string, ontology.Resource](
			ctx, gorp.TableConfig[string, ontology.Resource]{DB: db},
		))
		Expect(resTable.NewCreate().Entry(&ontology.Resource{ID: wsID}).Exec(ctx, db)).To(Succeed())
		Expect(resTable.NewCreate().Entry(&ontology.Resource{ID: lpID}).Exec(ctx, db)).To(Succeed())

		// Seed relationships: group -> workspace, workspace -> line plot, and an
		// unrelated group -> line plot that must be left untouched.
		relTable := MustOpen(gorp.OpenTable[string, ontology.Relationship](
			ctx, gorp.TableConfig[string, ontology.Relationship]{DB: db},
		))
		parent := ontology.RelationshipTypeParentOf
		groupToWS := ontology.Relationship{From: groupID, Type: parent, To: wsID}
		wsToLP := ontology.Relationship{From: wsID, Type: parent, To: lpID}
		groupToLP := ontology.Relationship{From: groupID, Type: parent, To: lpID}
		for _, rel := range []ontology.Relationship{groupToWS, wsToLP, groupToLP} {
			r := rel
			Expect(relTable.NewCreate().Entry(&r).Exec(ctx, db)).To(Succeed())
		}

		// Open the project table, running the migrations.
		projectTable := MustOpen(gorp.OpenTable[project.Key, project.Project](
			ctx, gorp.TableConfig[project.Key, project.Project]{
				DB: db,
				Migrations: []migrate.Migration{
					gorp.CodecMigration[project.Key, projectv56.Workspace]("msgpack_to_orc"),
					migrate.WithAddedDeps(
						gorp.NewMigration(
							"v56_migrate_workspace_to_project",
							project.MigrateWorkspaceToProject,
						),
						"msgpack_to_orc",
					),
				},
			},
		))

		By("Lifting the workspace into a project with identical fields")
		var p project.Project
		Expect(projectTable.NewRetrieve().Where(gorp.MatchKeys[project.Key, project.Project](wsKey)).Entry(&p).Exec(ctx, db)).To(Succeed())
		Expect(p).To(Equal(project.Project{
			Key:    wsKey,
			Name:   "Ops",
			Author: authorKey,
			Layout: msgpack.EncodedJSON{"mosaic": "tree"},
		}))

		By("Removing the legacy workspace record")
		Expect(wsTable.NewRetrieve().Where(gorp.MatchKeys[projectv56.Key, projectv56.Workspace](wsKey)).Entry(&projectv56.Workspace{}).Exec(ctx, db)).
			To(MatchError(query.ErrNotFound))

		By("Re-keying the workspace resource node to project")
		projectID := ontology.ID{Type: ontology.ResourceTypeProject, Key: wsKey.String()}
		var res ontology.Resource
		Expect(resTable.NewRetrieve().Where(gorp.MatchKeys[string, ontology.Resource](projectID.String())).Entry(&res).Exec(ctx, db)).To(Succeed())
		Expect(res.ID).To(Equal(projectID))
		Expect(resTable.NewRetrieve().Where(gorp.MatchKeys[string, ontology.Resource](wsID.String())).Entry(&ontology.Resource{}).Exec(ctx, db)).
			To(MatchError(query.ErrNotFound))

		By("Repointing workspace relationships and leaving unrelated ones intact")
		groupToProject := ontology.Relationship{From: groupID, Type: parent, To: projectID}
		projectToLP := ontology.Relationship{From: projectID, Type: parent, To: lpID}
		hasRel := func(rel ontology.Relationship) error {
			return relTable.NewRetrieve().Where(gorp.MatchKeys[string, ontology.Relationship](rel.GorpKey())).
				Entry(&ontology.Relationship{}).Exec(ctx, db)
		}
		Expect(hasRel(groupToProject)).To(Succeed())
		Expect(hasRel(projectToLP)).To(Succeed())
		Expect(hasRel(groupToWS)).To(MatchError(query.ErrNotFound))
		Expect(hasRel(wsToLP)).To(MatchError(query.ErrNotFound))
		Expect(hasRel(groupToLP)).To(Succeed())
	})
})
