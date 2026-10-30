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
	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/auth/password/versions/v0"
	authversions "github.com/synnaxlabs/synnax/pkg/service/auth/versions"
	userversions "github.com/synnaxlabs/synnax/pkg/service/user/versions"
	"github.com/synnaxlabs/x/types"
)

// MoveLegacy moves the credentials a Core stored before the password table existed
// into it, assigning each to the user with its username. It first brings the user table
// to its v0 shape, the last shape that holds usernames, so it runs before the user
// service opens. It runs once per Core; a later call does nothing.
func MoveLegacy(ctx context.Context, db *gorp.DB, ins alamos.Instrumentation) error {
	if err := gorp.Migrate(ctx, gorp.MigrateConfig{
		DB:              db,
		Namespace:       types.Name[userversions.User](),
		Migrations:      userversions.V0Migrations,
		Instrumentation: ins,
	}); err != nil {
		return err
	}
	return gorp.Migrate(ctx, gorp.MigrateConfig{
		DB: db,
		// steps a Core already ran under this namespace are skipped
		Namespace: authversions.Namespace,
		Migrations: slices.Concat(
			authversions.Migrations,
			[]gorp.Migration{v0.NewMove()},
		),
		Instrumentation: ins,
	})
}
