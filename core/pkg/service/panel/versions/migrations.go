// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// compositionNamespace is the namespace of the service layer's composition pass, which
// ran the format migrations below before they moved into the table chain.
const compositionNamespace = "Composition"

// Migrations is the ordered migration chain for stored panels.
var Migrations = []gorp.Migration{
	v0.Migration,
	gorp.MovedMigration(compositionNamespace, v0.ResourceTabTypesMigration),
	gorp.MovedMigration(compositionNamespace, v1.Migration),
	gorp.MovedMigration(compositionNamespace, v2.Migration),
	v3.Migration,
	v3.MembersMigration,
	v3.OwnersMigration,
	v3.ArcExplorerTabsMigration,
	v4.Migration,
}

// CompositionMigrations returns the migrations the service layer runs after every
// service table is open. They write each panel they create to table.
func CompositionMigrations(table *gorp.Table[Key, Panel]) []gorp.Migration {
	return v0.CompositionMigrations(
		func(ctx context.Context, tx gorp.Tx, project resource.ID, old v0.Panel) error {
			p, err := migratePanel(ctx, old)
			if err != nil {
				return err
			}
			owner, err := uuid.Parse(project.Key)
			if err != nil {
				return err
			}
			p.Project = &owner
			return table.NewUncheckedWriter(tx).Set(ctx, p)
		},
	)
}

// migratePanel lifts a v0 panel to the latest version.
func migratePanel(ctx context.Context, p0 v0.Panel) (Panel, error) {
	p1, err := v1.MigratePanel(ctx, p0)
	if err != nil {
		return Panel{}, err
	}
	p2, err := v2.MigratePanel(ctx, p1)
	if err != nil {
		return Panel{}, err
	}
	p3, err := v3.MigratePanel(ctx, p2)
	if err != nil {
		return Panel{}, err
	}
	return v4.MigratePanel(ctx, p3)
}

// Latest is the portable schema version stamped on a panel bundle member and the
// highest version an import accepts. It tracks the stored schema version, which the
// bundle body mirrors: the same fields, with each resource reference in the tree
// rewritten to a bundle path.
const Latest = v4.Version
