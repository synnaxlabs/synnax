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

	v0 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v0"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

// Migration lifts stored statuses from v0 to v1. Labels are intentionally dropped: they
// are no longer persisted on the status and are instead resolved at read time from the
// label relationship.
var Migration = gorp.NewEntryMigration(
	"v54_drop_labels",
	func(_ context.Context, old v0.Status[any]) (Status[any], error) {
		return Status[any]{
			Key:         old.Key,
			Name:        old.Name,
			Variant:     Variant(old.Variant),
			Message:     old.Message,
			Description: old.Description,
			Time:        telem.TimeStamp(old.Time),
			Details:     old.Details,
			Labels:      nil,
		}, nil
	},
)
