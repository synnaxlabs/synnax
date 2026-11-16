// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/versions"
)

// newMigrations returns the line plot version chain followed by the migration that
// drops missing targets.
func newMigrations() []gorp.Migration {
	return append(
		slices.Clone(versions.Migrations),
		gorp.DanglingMigration("v61_dangling_references"),
	)
}
