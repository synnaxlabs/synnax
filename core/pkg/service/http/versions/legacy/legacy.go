// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy converts the HTTP config shapes released Consoles wrote: v0 held
// headers, query params, and enum values as records, v1 holds them as entry lists.
package legacy

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// LastVersion is the newest legacy HTTP shape. The typed shape sits directly above it.
const LastVersion imex.Version = 1

// Read converts both released read shapes.
var Read = legacy.Rewrite{Pre: listify, Post: read}

// Write converts both released write shapes.
var Write = legacy.Rewrite{Pre: listify, Post: write}

// Scan converts the stored driver scan form.
var Scan = legacy.Scan

// listify converts the v0 record-shaped headers, query params, and enum values into
// their list shapes. It runs before the snake_case key pass because the record keys
// are data (header names, enum labels) that must not be converted.
func listify(config msgpack.EncodedJSON) {
	legacy.EachChild(config, "endpoints", func(ep msgpack.EncodedJSON) {
		legacy.RecordToList(ep, "headers", "name", "value")
		legacy.RecordToList(ep, "queryParams", "parameter", "value")
		legacy.RecordToList(ep, "query_params", "parameter", "value")
		legacy.EachChild(ep, "fields", func(f msgpack.EncodedJSON) {
			legacy.RecordToList(f, "enumValues", "label", "value")
			legacy.RecordToList(f, "enum_values", "label", "value")
		})
	})
}

// read flips the legacy field polarity on read endpoints and renames the timestamp
// format, which the write side always spelled time_format.
func read(config msgpack.EncodedJSON) {
	legacy.EachChild(config, "endpoints", func(ep msgpack.EncodedJSON) {
		legacy.EachChild(ep, "fields", func(f msgpack.EncodedJSON) {
			legacy.FlipBool(f, "enabled", "disabled")
			legacy.RenameKey(f, "timestamp_format", "time_format")
		})
	})
}

// write flips the legacy endpoint polarity on write endpoints.
func write(config msgpack.EncodedJSON) {
	legacy.EachChild(config, "endpoints", func(ep msgpack.EncodedJSON) {
		legacy.FlipBool(ep, "enabled", "disabled")
	})
}
