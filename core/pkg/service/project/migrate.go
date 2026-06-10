// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	projectv56 "github.com/synnaxlabs/synnax/pkg/service/project/migrations/v56"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

// legacyWorkspaceType is the ontology resource type workspaces were stored under
// before the rename to project.
const legacyWorkspaceType = ontology.ResourceType("workspace")

// MigrateWorkspaceToProject lifts a renamed workspace into a project. The
// workspace-to-project change was a pure rename, so it lifts every record from the
// legacy "Workspace" gorp prefix into the current Project type and repoints the
// ontology at the renamed type: it re-keys each workspace resource node to project
// and rewrites every relationship endpoint of type workspace to project. The whole
// transition runs in one migration transaction, so it commits atomically.
func MigrateWorkspaceToProject(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	if err := liftWorkspaces(ctx, tx); err != nil {
		return err
	}
	if err := rewriteWorkspaceResources(ctx, tx); err != nil {
		return err
	}
	return rewriteWorkspaceRelationships(ctx, tx)
}

// liftWorkspaces copies every legacy workspace record into the project table and
// removes it from the workspace table. The DB codec decodes the legacy records
// (orc, falling back to msgpack) and re-encodes the projects as orc.
func liftWorkspaces(ctx context.Context, tx gorp.Tx) error {
	stale, err := collectEntries(
		ctx,
		gorp.WrapReader[Key, projectv56.Workspace](tx),
		func(projectv56.Workspace) bool { return true },
	)
	if err != nil || len(stale) == 0 {
		return err
	}
	projects := make([]Project, len(stale))
	keys := make([]Key, len(stale))
	for i, ws := range stale {
		if projects[i], err = AutoMigrateProject(ctx, ws); err != nil {
			return err
		}
		keys[i] = ws.Key
	}
	if err := gorp.WrapWriter[Key, Project](tx).Set(ctx, projects...); err != nil {
		return err
	}
	return gorp.WrapWriter[Key, projectv56.Workspace](tx).Delete(ctx, keys...)
}

// rewriteWorkspaceResources re-keys every ontology resource node of type workspace
// to type project.
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

// rewriteWorkspaceRelationships repoints every relationship endpoint of type
// workspace to type project, preserving the relationship's direction and type.
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
