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
	"encoding/json"

	"github.com/synnaxlabs/gorp/migrate"
)

// Migration lifts stored statuses from v1 to v2, which adds the problem start and clear
// times. Both stay empty until the next write of each status. Details pass through as
// raw JSON, because decoding them as any rounds integers past 2^53.
var Migration = migrate.NewEntry(
	"v61_add_since_and_cleared",
	autoMigrateStatus[json.RawMessage],
)
