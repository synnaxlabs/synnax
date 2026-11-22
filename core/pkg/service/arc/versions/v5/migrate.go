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
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
)

// MigrateBase lifts a v4 Base with no channel lists. The next compile of the Arc fills
// them.
func MigrateBase(ctx context.Context, old v4.Base) (Base, error) {
	return autoMigrateBase(ctx, old)
}

// MigrateArc lifts a v4 Arc as MigrateBase lifts its Base.
func MigrateArc(ctx context.Context, old v4.Arc) (Arc, error) {
	return autoMigrateArc(ctx, old)
}

// Migration lifts each stored Arc to v5.
var Migration = migrate.NewEntry("v63_arc_compile_keys", MigrateArc)
