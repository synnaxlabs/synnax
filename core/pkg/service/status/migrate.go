// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"

	statusv54 "github.com/synnaxlabs/synnax/pkg/service/status/migrations/v54"
	"github.com/synnaxlabs/x/telem"
)

// MigrateStatus migrates a persisted v54 status into the current server-side Status
// entity. Labels are intentionally dropped: they are no longer persisted on the status
// and are instead resolved at read time from the label relationship. All other fields
// are copied across unchanged.
func MigrateStatus[Details any](
	_ context.Context,
	old statusv54.Status[Details],
) (Status[Details], error) {
	return Status[Details]{
		Key:         old.Key,
		Name:        old.Name,
		Variant:     Variant(old.Variant),
		Message:     old.Message,
		Description: old.Description,
		Time:        telem.TimeStamp(old.Time),
		Details:     old.Details,
		Labels:      nil,
	}, nil
}
