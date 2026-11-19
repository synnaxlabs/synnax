// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import "github.com/synnaxlabs/gorp/migrate"

// Migration lifts stored grants from v0 to v1. A v0 grant gets an empty command, so no
// task can spend it.
var Migration = migrate.NewEntry("v63_grant_command", autoMigrateGrant)
