// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v3 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v3"
)

// MigratePanel lifts a v3 panel to v4, which stores every color alpha as a 0-255
// byte.
func MigratePanel(ctx context.Context, old v3.Panel) (Panel, error) {
	return autoMigratePanel(ctx, old)
}

// Migration lifts stored panels from v3 to v4.
var Migration = migrate.NewEntry("v61_color_alpha_byte", MigratePanel)
