// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	taskversions "github.com/synnaxlabs/synnax/pkg/service/arc/task/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// NewTaskLink returns the task.ServiceConfig.Link function for Arc tasks. It makes the
// Arc that a task's config names the only Arc parent of the task in the ontology. It
// skips snapshots, other task types, and configs that name no Arc. It returns an error
// wrapping validate.ErrValidation when the named Arc does not exist.
func NewTaskLink(
	otg *ontology.Ontology,
) func(context.Context, gorp.Tx, task.Task) error {
	return func(ctx context.Context, tx gorp.Tx, t task.Task) error {
		if t.Type != TaskType || t.Snapshot {
			return nil
		}
		var cfg taskversions.Config
		if err := t.Config.Unmarshal(&cfg); err != nil {
			return err
		}
		if cfg.ArcKey == uuid.Nil() {
			return nil
		}
		var parents []ontology.Resource
		if err := otg.NewRetrieve().
			WhereIDs(t.OntologyID()).
			TraverseTo(ontology.ParentsTraverser).
			WhereTypes(ontology.ResourceTypeArc).
			Entries(&parents).
			ExcludeFieldData(true).
			Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
			return err
		}
		arcID, w := OntologyID(cfg.ArcKey), otg.NewWriter(tx)
		for _, parent := range parents {
			if parent.ID == arcID {
				continue
			}
			if err := w.DeleteRelationships(ctx, ontology.Relationship{
				From: parent.ID,
				Type: ontology.RelationshipTypeParentOf,
				To:   t.OntologyID(),
			}); err != nil {
				return err
			}
		}
		err := w.DefineRelationships(
			ctx, arcID, ontology.RelationshipTypeParentOf, t.OntologyID(),
		)
		if errors.Is(err, query.ErrNotFound) {
			return errors.Wrapf(
				validate.ErrValidation, "arc %s does not exist", cfg.ArcKey,
			)
		}
		return err
	}
}

// NewTaskLinkMigration returns the migration that links the Arc tasks that released
// Cores stored with no ontology link. A task whose Arc is gone stays unlinked: it
// cannot run, and a user can delete it.
func NewTaskLinkMigration(otg *ontology.Ontology, tasks *task.Service) gorp.Migration {
	link := NewTaskLink(otg)
	return migrate.New(
		"v61_arc_task_links",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var entries []task.Task
			if err := tasks.NewRetrieve().
				Where(task.MatchTypes(TaskType)).
				Entries(&entries).
				Exec(ctx, tx); err != nil {
				return err
			}
			for _, t := range entries {
				if err := errors.Skip(
					link(ctx, tx, t), validate.ErrValidation,
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
