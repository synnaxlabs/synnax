// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v0 "github.com/synnaxlabs/synnax/pkg/service/label/versions/v0"
)

// MigrateLabel lifts a v0 label to v1, which stores every color alpha as a 0-255
// byte.
func MigrateLabel(ctx context.Context, old v0.Label) (Label, error) {
	return autoMigrateLabel(ctx, old)
}

// Migration lifts stored labels from v0 to v1.
var Migration = migrate.NewEntry("v61_color_alpha_byte", MigrateLabel)
