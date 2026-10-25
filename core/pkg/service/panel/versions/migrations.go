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
	"slices"

	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v2"
)

// Migrations is the ordered migration chain for stored panels.
var Migrations = []gorp.Migration{v0.Migration}

// CompositionMigrations is the ordered set of migrations the service layer runs after
// every service table is open, concatenating each version's contribution. The lifts
// past v0 run here, not in the table chain, because the v0 migrations before them write
// v0 panels.
var CompositionMigrations = slices.Concat(
	v0.CompositionMigrations,
	[]gorp.Migration{v1.Migration, v2.Migration},
)

// Latest is the portable schema version stamped on a panel bundle member and the
// highest version an import accepts. It tracks the stored schema version, which the
// bundle body mirrors: the same fields, with each resource reference in the tree
// rewritten to a bundle path.
const Latest = v2.Version
