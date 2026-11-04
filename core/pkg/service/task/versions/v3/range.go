// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/set"
)

// Migration moves each stored task to the v3 shape, which adds a range.
var Migration = migrate.NewEntry("v61_task_add_range", autoMigrateTask)

// RangeMigration sets the range of each stored task from the parent edge that points
// at it from a range the ontology holds. It then deletes every parent edge into a task
// from a range or an Arc.
var RangeMigration = migrate.New("v61_task_range", migrateRanges)

func migrateRanges(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	edges, err := collectEntries(
		ctx,
		gorp.WrapReader[string, ontology.Relationship](tx),
		func(r ontology.Relationship) bool {
			return r.Type == ontology.RelationshipTypeParentOf &&
				r.To.Type == resource.TypeTask &&
				(r.From.Type == resource.TypeRange || r.From.Type == resource.TypeArc)
		},
	)
	if err != nil || len(edges) == 0 {
		return err
	}
	ranges, err := collectEntries(
		ctx,
		gorp.WrapReader[string, ontology.Resource](tx),
		func(r ontology.Resource) bool { return r.ID.Type == resource.TypeRange },
	)
	if err != nil {
		return err
	}
	held := set.New[string]()
	for _, r := range ranges {
		held.Add(r.ID.Key)
	}
	rangeOf := make(map[Key]uuid.UUID, len(edges))
	for _, e := range edges {
		if e.From.Type != resource.TypeRange || !held.Contains(e.From.Key) {
			continue
		}
		taskKey, err := uuid.Parse(e.To.Key)
		if err != nil {
			continue
		}
		rangeKey, err := uuid.Parse(e.From.Key)
		if err != nil {
			continue
		}
		if _, ok := rangeOf[taskKey]; !ok {
			rangeOf[taskKey] = rangeKey
		}
	}
	tasks, err := collectEntries(
		ctx,
		gorp.WrapReader[Key, Task](tx),
		func(t Task) bool { _, ok := rangeOf[t.Key]; return ok },
	)
	if err != nil {
		return err
	}
	for i := range tasks {
		rng := rangeOf[tasks[i].Key]
		tasks[i].Range = &rng
	}
	if err = gorp.WrapWriter[Key, Task](tx).Set(ctx, tasks...); err != nil {
		return err
	}
	keys := make([]string, len(edges))
	for i, e := range edges {
		keys[i] = e.GorpKey()
	}
	return gorp.WrapWriter[string, ontology.Relationship](tx).Delete(ctx, keys...)
}
