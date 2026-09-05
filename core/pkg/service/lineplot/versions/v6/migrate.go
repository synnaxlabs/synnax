// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v6

import (
	"context"

	v5 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v5"
	"github.com/synnaxlabs/x/gorp"
)

func MigrateLinePlot(ctx context.Context, old v5.LinePlot) (LinePlot, error) {
	return autoMigrateLinePlot(ctx, old)
}

func MigrateRanges(ctx context.Context, old v5.Ranges) (Ranges, error) {
	return autoMigrateRanges(ctx, old)
}

var Migration = gorp.NewEntryMigration("v58_custom_range", MigrateLinePlot)
