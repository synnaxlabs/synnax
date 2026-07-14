// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v56

import (
	"context"

	v54 "github.com/synnaxlabs/synnax/pkg/service/task/migrations/v54"
)

// MigrateTask drops the embedded status from a v54 task record. Statuses are stored
// separately from v56 onward.
func MigrateTask(_ context.Context, old v54.Task) (Task, error) {
	return Task{
		Key:      Key(old.Key),
		Name:     old.Name,
		Type:     old.Type,
		Config:   old.Config,
		Internal: old.Internal,
		Snapshot: old.Snapshot,
	}, nil
}
