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

	"github.com/synnaxlabs/x/gorp"
)

// Migration re-encodes stored entries from MessagePack to Orc.
var Migration = gorp.CodecMigration[Key, User]("msgpack_to_orc")

// NormalizeKeys re-keys User rows stored under the pre-v0.54 key format.
var NormalizeKeys = gorp.NormalizeKeysMigration[Key, User]("User")

// KeysByUsername returns the key of every user in tx, indexed by username.
func KeysByUsername(ctx context.Context, tx gorp.Tx) (map[string]Key, error) {
	var users []User
	if err := gorp.NewRetrieve[Key, User]().Entries(&users).Exec(ctx, tx); err != nil {
		return nil, err
	}
	keys := make(map[string]Key, len(users))
	for _, u := range users {
		keys[u.Username] = u.Key
	}
	return keys, nil
}
