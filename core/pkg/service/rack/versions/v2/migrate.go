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

	v1 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v1"
	"github.com/synnaxlabs/x/gorp"
)

// migrateRack lifts a v1 rack into the v2 shape, dropping the local task-key counter
// made obsolete by UUID task keys.
func migrateRack(_ context.Context, old v1.Rack) (Rack, error) {
	return Rack{
		Key:          old.Key,
		Name:         old.Name,
		Embedded:     old.Embedded,
		Integrations: old.Integrations,
	}, nil
}

// Migration lifts stored racks from v1 to v2, dropping the task-key counter.
var Migration = gorp.NewEntryMigration("v56_drop_task_counter", migrateRack)
