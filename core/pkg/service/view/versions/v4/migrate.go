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
	v3 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v3"
)

// MigrateView lifts a v3 view to v4. Range and status views keep their shape, and no
// channel view exists before v4.
func MigrateView(ctx context.Context, old v3.View) (View, error) {
	return autoMigrateView(ctx, old)
}

// Migration moves each stored view to its v4 shape.
var Migration = migrate.NewEntry("v63_channel_view", MigrateView)
