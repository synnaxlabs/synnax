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
	v0 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v0"
)

// Migration lifts stored panels from v0 to v1, dropping the args of every view tab.
var Migration = migrate.NewEntry("v60_drop_view_args", autoMigratePanel)

// MigrateView lifts a v0 view into the v1 shape, dropping its args.
func MigrateView(ctx context.Context, old v0.View) (View, error) {
	return autoMigrateView(ctx, old)
}
