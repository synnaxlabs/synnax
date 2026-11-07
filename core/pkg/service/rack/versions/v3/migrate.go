// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import "github.com/synnaxlabs/gorp/migrate"

// Migration lifts stored racks from v2 to v3, which adds an optional Driver version.
var Migration = migrate.NewEntry("v61_add_driver_version", autoMigrateRack)
