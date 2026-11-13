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
	v2 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v2"
)

// MigrateRange lifts a v2 range to v3, which stores every color alpha as a 0-255
// byte.
func MigrateRange(ctx context.Context, old v2.Range) (Range, error) {
	return autoMigrateRange(ctx, old)
}

// Migration lifts stored ranges from v2 to v3.
var Migration = migrate.NewEntry("v62_color_alpha_byte", MigrateRange)
