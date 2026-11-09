// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v4 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v4"
)

// MigrateTable lifts a v4 table to v5, which stores every color alpha as a 0-255
// byte.
func MigrateTable(ctx context.Context, old v4.Table) (Table, error) {
	return autoMigrateTable(ctx, old)
}

// Migration lifts stored tables from v4 to v5.
var Migration = migrate.NewEntry("v61_color_alpha_byte", MigrateTable)
