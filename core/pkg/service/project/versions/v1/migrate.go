// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"encoding/json"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v0 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
)

// LegacyLayoutKVPrefix is the KV key prefix under which the layout staging migration
// stages each project's legacy layout blob. The remainder of the key is the project's
// key. The panel migration scans this prefix to convert the blobs into panels and
// deletes the entries as it consumes them, so it never reads the project layout field
// and the field can later be removed without ordering a project-table migration after a
// panel-table one.
const LegacyLayoutKVPrefix = "sy_project_legacy_layout/"

// migrateLayoutsToStaging copies every project's layout blob into its own staging KV
// entry keyed by LegacyLayoutKVPrefix plus the project key. The blob is the project
// layout marshalled to
// JSON, which the panel migration unmarshals back into its legacy mosaic structs.
// The layout field is left intact; this migration only forwards a copy so the panel
// migration no longer depends on reading it.
func migrateLayoutsToStaging(
	ctx context.Context,
	tx gorp.Tx,
	_ alamos.Instrumentation,
) error {
	projects, err := collectEntries(
		ctx,
		gorp.WrapReader[Key, Project](tx),
		func(p Project) bool { return len(p.Layout) > 0 },
	)
	if err != nil {
		return err
	}
	for _, p := range projects {
		blob, err := json.Marshal(p.Layout)
		if err != nil {
			return errors.Wrapf(err, "marshal layout for project %s", p.Key)
		}
		if err = tx.Set(ctx, []byte(LegacyLayoutKVPrefix+p.Key.String()), blob); err != nil {
			return err
		}
	}
	return nil
}

// legacyWorkspaceType is the ontology resource type workspaces were stored under
// before the rename to project.
const legacyWorkspaceType = ontology.ResourceType("workspace")

// migrateWorkspaceToProject lifts a renamed workspace into a project. The
// workspace-to-project change was a pure rename, so it lifts every record from the
// legacy "Workspace" gorp prefix into the current Project type and repoints the
// ontology at the renamed type: it re-keys each workspace resource node to project and
// rewrites every relationship endpoint of type workspace to project. The whole
// transition runs in one migration transaction, so it commits atomically.
func migrateWorkspaceToProject(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	if err := liftWorkspaces(ctx, tx); err != nil {
		return err
	}
	if err := rewriteWorkspaceResources(ctx, tx); err != nil {
		return err
	}
	if err := rewriteWorkspaceRelationships(ctx, tx); err != nil {
		return err
	}
	return renameWorkspaceGroup(ctx, tx)
}

// renameWorkspaceGroup renames the root "Workspaces" group to "Projects" so the
// service retrieves the existing group (with the migrated projects already
// parented under it) rather than creating a second one.
func renameWorkspaceGroup(ctx context.Context, tx gorp.Tx) error {
	stale, err := collectEntries(
		ctx,
		gorp.WrapReader[group.Key, group.Group](tx),
		func(g group.Group) bool { return g.Name == "Workspaces" },
	)
	if err != nil {
		return err
	}
	w := gorp.WrapWriter[group.Key, group.Group](tx)
	for _, g := range stale {
		g.Name = "Projects"
		if err := w.Set(ctx, g); err != nil {
			return err
		}
	}
	return nil
}

// liftWorkspaces copies every legacy workspace record into the project table and
// removes it from the workspace table. The DB codec decodes the legacy records (Orc,
// falling back to MessagePack) and re-encodes the projects as Orc.
func liftWorkspaces(ctx context.Context, tx gorp.Tx) error {
	stale, err := collectEntries(
		ctx,
		gorp.WrapReader[Key, v0.Workspace](tx),
		func(v0.Workspace) bool { return true },
	)
	if err != nil || len(stale) == 0 {
		return err
	}
	projects := make([]Project, len(stale))
	keys := make([]Key, len(stale))
	for i, ws := range stale {
		if projects[i], err = autoMigrateProject(ctx, ws); err != nil {
			return err
		}
		keys[i] = ws.Key
	}
	if err := gorp.WrapWriter[Key, Project](tx).Set(ctx, projects...); err != nil {
		return err
	}
	return gorp.WrapWriter[Key, v0.Workspace](tx).Delete(ctx, keys...)
}

// rewriteWorkspaceResources re-keys every ontology resource node of type workspace to
// type project.
func rewriteWorkspaceResources(ctx context.Context, tx gorp.Tx) error {
	stale, err := collectEntries(
		ctx,
		gorp.WrapReader[string, ontology.Resource](tx),
		func(r ontology.Resource) bool { return r.ID.Type == legacyWorkspaceType },
	)
	if err != nil {
		return err
	}
	w := gorp.WrapWriter[string, ontology.Resource](tx)
	for _, r := range stale {
		oldKey := r.GorpKey()
		r.ID.Type = ontology.ResourceTypeProject
		if err := w.Set(ctx, r); err != nil {
			return err
		}
		if err := w.Delete(ctx, oldKey); err != nil {
			return err
		}
	}
	return nil
}

// rewriteWorkspaceRelationships repoints every relationship endpoint of type workspace
// to type project, preserving the relationship's direction and type.
func rewriteWorkspaceRelationships(ctx context.Context, tx gorp.Tx) error {
	stale, err := collectEntries(
		ctx,
		gorp.WrapReader[string, ontology.Relationship](tx),
		func(rel ontology.Relationship) bool {
			return rel.From.Type == legacyWorkspaceType || rel.To.Type == legacyWorkspaceType
		},
	)
	if err != nil {
		return err
	}
	w := gorp.WrapWriter[string, ontology.Relationship](tx)
	for _, rel := range stale {
		oldKey := rel.GorpKey()
		if rel.From.Type == legacyWorkspaceType {
			rel.From.Type = ontology.ResourceTypeProject
		}
		if rel.To.Type == legacyWorkspaceType {
			rel.To.Type = ontology.ResourceTypeProject
		}
		if err := w.Set(ctx, rel); err != nil {
			return err
		}
		if err := w.Delete(ctx, oldKey); err != nil {
			return err
		}
	}
	return nil
}

// removeAuthorRelationships deletes the parent-of relationships that pointed from a
// project's author user to the project. The author field is no longer part of a
// project, so its relationships are orphaned on existing clusters; a project's only
// parent is now the Projects group. This must run after the workspace-to-project
// migration, which re-types the legacy user-to-workspace relationships to
// user-to-project before they can be matched here.
func removeAuthorRelationships(ctx context.Context, tx gorp.Tx, otg *ontology.Ontology) error {
	stale, err := collectEntries(
		ctx,
		gorp.WrapReader[string, ontology.Relationship](tx),
		func(rel ontology.Relationship) bool {
			return rel.Type == ontology.RelationshipTypeParentOf &&
				rel.From.Type == ontology.ResourceTypeUser &&
				rel.To.Type == ontology.ResourceTypeProject
		},
	)
	if err != nil {
		return err
	}
	return otg.NewWriter(tx).DeleteRelationships(ctx, stale...)
}

// collectEntries drains a reader into a slice of the entries matching keep.
// Mutating a gorp table while iterating it is unsafe, so callers gather first and
// write after.
func collectEntries[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	r gorp.Reader[K, E],
	keep func(E) bool,
) (out []E, err error) {
	iter, err := r.OpenIterator(gorp.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		if e := iter.Value(ctx); e != nil && keep(*e) {
			out = append(out, *e)
		}
	}
	return out, iter.Error()
}

// workspaceToProjectMigration lifts stored workspaces into projects. It
// depends on the codec migration so it always reads Orc-encoded entries.
var workspaceToProjectMigration = migrate.WithAddedDeps(
	gorp.NewMigration("v56_migrate_workspace_to_project", migrateWorkspaceToProject),
	v0.Migration.Key(),
)

// layoutsToStagingMigration stages each project's legacy layout blob for the
// panel service to adopt.
var layoutsToStagingMigration = migrate.WithAddedDeps(
	gorp.NewMigration("v56_stage_project_layouts", migrateLayoutsToStaging),
	workspaceToProjectMigration.Key(),
)

// removeAuthorRelationshipsMigration drops the legacy author relationships
// projects held in the ontology.
func removeAuthorRelationshipsMigration(otg *ontology.Ontology) migrate.Migration {
	return migrate.WithAddedDeps(
		gorp.NewMigration(
			"v56_remove_project_author_relationships",
			func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
				return removeAuthorRelationships(ctx, tx, otg)
			},
		),
		workspaceToProjectMigration.Key(),
	)
}

// MigrationsConfig is the configuration for NewMigrations.
type MigrationsConfig struct {
	// Ontology is the cluster ontology; the author-relationship cleanup runs
	// against it.
	Ontology *ontology.Ontology
}

// NewMigrations returns the ordered migration chain for stored projects.
func NewMigrations(cfg MigrationsConfig) []migrate.Migration {
	return []migrate.Migration{
		v0.Migration,
		workspaceToProjectMigration,
		layoutsToStagingMigration,
		removeAuthorRelationshipsMigration(cfg.Ontology),
	}
}
