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
	v2 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v2"
)

// MigrateBase lifts the base of a v2 view to v3, with no icon or color.
func MigrateBase(ctx context.Context, old v2.Base) (Base, error) {
	return autoMigrateBase(ctx, old)
}

// Migration moves each stored view to its v3 shape. A range view keeps its where and
// has no sort or layout.
var Migration = migrate.NewEntry("v62_view_shape", autoMigrateView)
