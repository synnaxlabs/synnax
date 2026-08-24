// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import "github.com/synnaxlabs/x/gorp"

// Migration lifts stored racks from v1 to v2, dropping the task-key counter.
var Migration = gorp.NewEntryMigration("v56_drop_task_counter", autoMigrateRack)
