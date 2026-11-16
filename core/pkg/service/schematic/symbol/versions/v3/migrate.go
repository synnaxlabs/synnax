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

	"github.com/synnaxlabs/gorp/migrate"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v2"
)

// MigrateSymbol lifts a v2 symbol into v3. The symbol sits in no group.
func MigrateSymbol(ctx context.Context, old v2.Symbol) (Symbol, error) {
	return autoMigrateSymbol(ctx, old)
}

var Migration = migrate.NewEntry("v61_add_group", MigrateSymbol)

// MembersMigration sets the group of each stored schematic symbol from the ontology
// edge that placed it in a group.
var MembersMigration = groupv1.NewMembersMigration(groupv1.MembersConfig[Key, Symbol]{
	Name:         "v61_schematic_symbol_group_membership",
	Type:         resource.TypeSchematicSymbol,
	SetGroup:     func(e *Symbol, g groupv1.Key) { e.Group = &g },
	ProjectOwned: false,
})
