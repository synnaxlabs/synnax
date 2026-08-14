// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy converts the LabJack config shapes released Consoles wrote: v0 keyed
// write channels with cmdKey and stateKey, v1 renamed them to cmdChannel and
// stateChannel, and channel type tags were port-mode acronyms.
package legacy

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// LastVersion is the newest legacy LabJack shape. The typed shape sits directly
// above it.
const LastVersion imex.Version = 1

// readTypes maps released read channel type tags to their current labels.
var readTypes = map[string]string{
	"AI": "analog",
	"DI": "digital",
	"TC": "thermocouple",
}

// writeTypes maps released write channel type tags to their current labels.
var writeTypes = map[string]string{
	"AO": "analog",
	"DO": "digital",
}

// Read converts the released read shape.
var Read = legacy.Rewrite{Post: func(config msgpack.EncodedJSON) {
	legacy.EachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		legacy.RemapValue(ch, "type", readTypes)
	})
}}

// Write converts both released write shapes.
var Write = legacy.Rewrite{Post: write}

// Scan converts the stored driver scan form.
var Scan = legacy.Scan

func write(config msgpack.EncodedJSON) {
	legacy.EachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		legacy.RenameKey(ch, "cmd_key", "cmd_channel")
		legacy.RenameKey(ch, "state_key", "state_channel")
		legacy.RemapValue(ch, "type", writeTypes)
	})
}
