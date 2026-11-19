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
	v0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
)

// MigrateICD lifts a v0 ICD to v1.
func MigrateICD(ctx context.Context, old v0.ICD) (ICD, error) {
	return autoMigrateICD(ctx, old)
}

// MigrateMessage lifts a v0 message to v1, with no cycle time.
func MigrateMessage(ctx context.Context, old v0.Message) (Message, error) {
	return autoMigrateMessage(ctx, old)
}

// Migration lifts stored ICDs from v0 to v1.
var Migration = migrate.NewEntry("v63_message_cycle_time", MigrateICD)
