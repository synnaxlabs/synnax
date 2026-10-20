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
	"slices"

	"github.com/synnaxlabs/alamos"
	v0 "github.com/synnaxlabs/synnax/pkg/service/auth/password/versions/v0"
	authversions "github.com/synnaxlabs/synnax/pkg/service/auth/versions"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
)

// MoveLegacy moves the credentials a Core stored before the password table existed
// into it, assigning each to the user with its username. It reads the user table, so
// it runs after that table's migrations and before the password service opens. It runs
// once per Core; a later call does nothing.
func MoveLegacy(ctx context.Context, db *gorp.DB, ins alamos.Instrumentation) error {
	return gorp.Migrate(ctx, gorp.MigrateConfig{
		DB: db,
		// steps a Core already ran under this namespace are skipped
		Namespace: authversions.Namespace,
		Migrations: slices.Concat(
			authversions.Migrations,
			[]migrate.Migration{v0.NewMove()},
		),
		Instrumentation: ins,
	})
}
