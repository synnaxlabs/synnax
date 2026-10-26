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
	"context"

	"github.com/synnaxlabs/gorp/migrate"
)

// Migration re-encodes stored roles from MessagePack to Orc.
var Migration = migrate.Codec[Key, Role]("msgpack_to_orc")

// NormalizeKeys re-keys Role rows stored under the pre-v0.54 key format.
var NormalizeKeys = migrate.NormalizeKeys[Key, Role]("Role")

// NameMigration names each stored role that has an empty name "Role <key>",
// where <key> is the first eight characters of its key.
var NameMigration = migrate.NewEntry(
	"v60_require_name",
	func(_ context.Context, r Role) (Role, error) {
		if r.Name == "" {
			r.Name = "Role " + r.Key.String()[:8]
		}
		return r, nil
	},
)
