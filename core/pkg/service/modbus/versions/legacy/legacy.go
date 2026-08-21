// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy converts the Modbus config shapes released Consoles wrote: channel
// type tags carried a direction suffix the union name now owns, and the register byte
// and word order flags were named for the swap rather than the resulting state.
package legacy

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// LastVersion is the newest legacy Modbus shape. The typed shape sits directly
// above it.
const LastVersion imex.Version = 0

// readTypes maps released read channel type tags to their current labels.
var readTypes = map[string]string{
	"coil_input":             "coil",
	"holding_register_input": "holding_register",
	"register_input":         "input_register",
}

// writeTypes maps released write channel type tags to their current labels.
var writeTypes = map[string]string{
	"coil_output":             "coil",
	"holding_register_output": "holding_register",
}

// Read converts the released read shape.
var Read = legacy.Rewrite{Post: func(config msgpack.EncodedJSON) {
	legacy.EachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		legacy.RemapValue(ch, "type", readTypes)
		renameSwaps(ch)
	})
}}

// Write converts the released write shape.
var Write = legacy.Rewrite{Post: func(config msgpack.EncodedJSON) {
	legacy.EachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		legacy.RemapValue(ch, "type", writeTypes)
		renameSwaps(ch)
	})
}}

func renameSwaps(ch msgpack.EncodedJSON) {
	legacy.RenameKey(ch, "swap_bytes", "bytes_swapped")
	legacy.RenameKey(ch, "swap_words", "words_swapped")
}

// Scan converts the stored driver scan form.
var Scan = legacy.Scan
