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

// CodecMigrationKey names the codec migration the project-layout adoption
// migration depends on.
const CodecMigrationKey = "msgpack_to_orc"

// CodecMigration re-encodes stored panels from msgpack to orc.
var CodecMigration = gorp.CodecMigration[Key, Panel](CodecMigrationKey)
