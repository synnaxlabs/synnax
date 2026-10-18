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

	v0 "github.com/synnaxlabs/synnax/pkg/service/auth/password/versions/v0"
	authversions "github.com/synnaxlabs/synnax/pkg/service/auth/versions"
	"github.com/synnaxlabs/x/migrate"
)

// LegacyConfig configures [NewLegacyMigrations].
type LegacyConfig = v0.MoveConfig

// SubjectLookup resolves the subject that owns a stored credential's username.
type SubjectLookup = v0.SubjectLookup

// LegacyNamespace is the applied-migration namespace the legacy chain runs under: the
// credentials table's own, so steps a Core already ran are not run again.
var LegacyNamespace = authversions.Namespace

// NewLegacyMigrations returns the chain that lifts the credentials table to its last
// shape and then moves it into the password table.
func NewLegacyMigrations(cfg LegacyConfig) []migrate.Migration {
	return slices.Concat(
		authversions.Migrations,
		[]migrate.Migration{v0.NewMove(cfg)},
	)
}
