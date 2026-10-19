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

	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
	"github.com/synnaxlabs/x/gorp"
)

// MigrateArc lifts a v3 Arc to v4. Every stored Arc predates modules, so it is an
// automation.
func MigrateArc(ctx context.Context, old v3.Arc) (Arc, error) {
	a, err := autoMigrateArc(ctx, old)
	if err != nil {
		return Arc{}, err
	}
	a.Type = TypeAutomation
	return a, nil
}

// Migration lifts stored Arcs from v3 to v4.
var Migration = gorp.NewEntryMigration("v60_type", MigrateArc)
