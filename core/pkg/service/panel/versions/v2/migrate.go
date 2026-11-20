// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v1 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v1"
)

// MigratePanel lifts a v1 panel into v2, which adds an optional icon and color.
func MigratePanel(ctx context.Context, old v1.Panel) (Panel, error) {
	return autoMigratePanel(ctx, old)
}

// Migration lifts stored panels from v1 to v2.
var Migration = migrate.NewEntry("v60_add_icon_and_color", MigratePanel)
