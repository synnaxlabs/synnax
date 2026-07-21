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
	v0 "github.com/synnaxlabs/synnax/pkg/service/project/types/v0"
	"github.com/synnaxlabs/x/gorp"
)

// CodecMigrationKey names the codec migration the workspace-to-project
// migration depends on.
const CodecMigrationKey = "msgpack_to_orc"

// CodecMigration re-encodes stored workspaces from msgpack to orc. It is
// pinned to the v0 shape so its output stays stable as Project evolves.
var CodecMigration = gorp.CodecMigration[Key, v0.Workspace](CodecMigrationKey)
