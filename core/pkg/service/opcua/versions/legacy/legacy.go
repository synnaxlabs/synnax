// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy converts the OPC UA config shapes released Consoles wrote: read
// channels named the index flag for the act of using the node rather than the state.
// Write channels carried cmdChannel from the first release, so they need only era
// normalization.
package legacy

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// LastVersion is the newest legacy OPC UA shape. The typed shape sits directly
// above it.
const LastVersion imex.Version = 0

// Read converts the released read shape. A config that already holds sampling keeps
// it: the rewrite writes it as the array_mode and array_size that the first typed shape
// lifts back into the same sampling. Only a config without sampling keeps its own
// array_mode and array_size.
var Read = legacy.Rewrite{Post: func(config msgpack.EncodedJSON) {
	legacy.EachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		legacy.RenameKey(ch, "use_as_index", "is_index")
	})
	sampling, ok := config["sampling"].(map[string]any)
	if !ok {
		return
	}
	delete(config, "sampling")
	delete(config, "array_size")
	config["array_mode"] = sampling["type"] == "array"
	if size, ok := sampling["size"]; ok {
		config["array_size"] = size
	}
}}

// Scan converts the stored driver scan form.
var Scan = legacy.Scan
