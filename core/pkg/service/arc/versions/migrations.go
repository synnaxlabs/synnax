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
	"uuid"

	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v1"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
)

// NewMigrations returns the ordered migration chain for stored arcs. root is the
// directory that holds each Arc stored in no directory. cfg is the parser config of the
// Core that stored the Arcs.
func NewMigrations(root uuid.UUID, cfg parser.Config) []gorp.Migration {
	return slices.Concat(
		[]gorp.Migration{v0.NormalizeKeys, v0.Migration},
		v1.Migrations,
		[]gorp.Migration{
			v3.Migration,
			v3.ModeMigration,
			v4.NewMigration(cfg),
			v4.NewRootMigration(root),
		},
	)
}
