// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy converts the one EtherCAT config shape released Consoles wrote. Its
// PDO sub-index key is the all-lowercase "subindex", which the generic camelCase pass
// cannot convert, so it needs an explicit rename.
package legacy

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// LastVersion is the newest legacy EtherCAT shape. The typed shape sits directly
// above it.
const LastVersion imex.Version = 0

// Read converts the released read shape.
var Read = legacy.Rewrite{Post: subIndex}

// Write converts the released write shape.
var Write = legacy.Rewrite{Post: subIndex}

// Scan converts the stored driver scan form.
var Scan = legacy.Scan

func subIndex(config msgpack.EncodedJSON) {
	legacy.EachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		legacy.RenameKey(ch, "subindex", "sub_index")
		if addr, ok := ch["address"].(map[string]any); ok {
			legacy.RenameKey(addr, "subindex", "sub_index")
		}
	})
}
