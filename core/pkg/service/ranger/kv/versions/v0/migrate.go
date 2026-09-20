// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "github.com/synnaxlabs/x/gorp"

// Migration re-encodes stored entries from MessagePack to Orc.
var Migration = gorp.CodecMigration[string, Pair]("msgpack_to_orc")

// NormalizeKeys re-keys Pair rows stored under the pre-v0.54 key format. v0.53 and
// earlier stored them under the type name "Pair"; CustomTypeName renamed it to
// "KVPair" in v0.54.
var NormalizeKeys = gorp.NormalizeKeysMigration[string, Pair]("Pair")
