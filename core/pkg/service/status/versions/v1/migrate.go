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
	"encoding/json"

	"github.com/synnaxlabs/gorp/migrate"
)

// Migration lifts stored statuses from v0 to v1. Labels are intentionally dropped: they
// are no longer persisted on the status and are instead resolved at read time from the
// label relationship. Details pass through as raw JSON, because decoding them as any
// rounds integers past 2^53.
var Migration = migrate.NewEntry("v54_drop_labels", autoMigrateStatus[json.RawMessage])
