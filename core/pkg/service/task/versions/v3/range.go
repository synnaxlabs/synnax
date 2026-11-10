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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	ranger "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
)

// Migration moves each stored task to the v3 shape, which adds a range.
var Migration = migrate.NewEntry("v61_task_add_range", autoMigrateTask)

// RangeMigration sets the range of each stored snapshot from the parent edge that
// points at it from a stored range. It then deletes every parent edge into a task from
// a range or an Arc. It reads ranges in their v2 shape, so it runs after the range
// migrations.
var RangeMigration = ontology.NewFieldMigration(ontology.FieldConfig[Key, Task]{
	Name: "v61_task_range",
	Converts: func(rel ontology.Relationship) bool {
		return rel.Type == ontology.RelationshipTypeParentOf &&
			rel.To.Type == resource.TypeTask &&
			(rel.From.Type == resource.TypeRange || rel.From.Type == resource.TypeArc)
	},
	Holder: func(rel ontology.Relationship) string { return rel.To.Key },
	Set:    setRange,
})

func setRange(
	ctx context.Context,
	tx gorp.Tx,
	t *Task,
	rel ontology.Relationship,
) (bool, error) {
	if rel.From.Type != resource.TypeRange || !t.Snapshot || t.Range != nil {
		return false, nil
	}
	key, err := uuid.Parse(rel.From.Key)
	if err != nil {
		return false, errors.Wrapf(err, "range of task %s", t.Key)
	}
	exists, err := gorp.NewRetrieve[ranger.Key, ranger.Range]().
		Where(gorp.MatchKeys[ranger.Key, ranger.Range](key)).
		Exists(ctx, tx)
	if err != nil || !exists {
		return false, err
	}
	t.Range = &key
	return true, nil
}
