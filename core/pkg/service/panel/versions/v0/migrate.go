// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
)

// codecMigration re-encodes stored panels from MessagePack to Orc.
var codecMigration = gorp.CodecMigration[Key, Panel]("msgpack_to_orc")

// Migrations is the ordered set of migrations introduced at this version. The
// migrations that adopt other services' staged legacy data (project layouts, task
// re-key mapping) live in the parent versions package's CompositionMigrations: they
// cannot run in this chain because the producing services open after the panel table.
var Migrations = []migrate.Migration{codecMigration}
