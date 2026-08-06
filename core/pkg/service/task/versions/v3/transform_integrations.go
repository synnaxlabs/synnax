// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import "github.com/synnaxlabs/x/encoding/msgpack"

// transformNIRead rewrites legacy NI analog and counter read configs: the v0
// config-level device is copied onto every channel missing one, camelCase channel
// keys become snake_case, and the renamed AI type alias is replaced.
func transformNIRead(config msgpack.EncodedJSON) {
	device, hasDevice := config["device"]
	delete(config, "device")
	eachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		if hasDevice {
			if _, ok := ch["device"]; !ok {
				ch["device"] = device
			}
		}
		renameKey(ch, "terminalA", "terminal_a")
		renameKey(ch, "terminalB", "terminal_b")
		renameKey(ch, "terminalZ", "terminal_z")
		renameKey(ch, "activeEdge", "active_edge")
		if ch["type"] == "ai_frequency_voltage" {
			ch["type"] = "ai_freq_voltage"
		}
	})
}

// transformOPCWrite renames the legacy v0 output channel key "channel" to
// "cmd_channel".
func transformOPCWrite(config msgpack.EncodedJSON) {
	eachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		renameKey(ch, "channel", "cmd_channel")
	})
}

// transformLabJackWrite renames the legacy output channel key "cmd_key" to
// "cmd_channel".
func transformLabJackWrite(config msgpack.EncodedJSON) {
	eachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		renameKey(ch, "cmd_key", "cmd_channel")
	})
}

// transformEtherCAT normalizes the PDO sub-index key, which legacy clients wrote as
// "subindex" or "subIndex", on channels and their nested addresses.
func transformEtherCAT(config msgpack.EncodedJSON) {
	normalize := func(m msgpack.EncodedJSON) {
		renameKey(m, "subindex", "sub_index")
		renameKey(m, "subIndex", "sub_index")
	}
	eachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		normalize(ch)
		if addr, ok := ch["address"].(map[string]any); ok {
			normalize(addr)
		}
	})
}

// transformHTTPRead converts the legacy v0 record-shaped headers, query params, and
// enum values into their list shapes and flips field polarity.
func transformHTTPRead(config msgpack.EncodedJSON) {
	eachChild(config, "endpoints", func(ep msgpack.EncodedJSON) {
		transformHTTPRequest(ep)
		eachChild(ep, "fields", func(f msgpack.EncodedJSON) {
			flipBool(f, "enabled", "disabled")
			renameKey(f, "enumValues", "enum_values")
			recordToList(f, "enum_values", "label", "value")
		})
	})
}

// transformHTTPWrite converts the legacy v0 record-shaped headers and query params
// into their list shapes and flips endpoint polarity.
func transformHTTPWrite(config msgpack.EncodedJSON) {
	eachChild(config, "endpoints", func(ep msgpack.EncodedJSON) {
		flipBool(ep, "enabled", "disabled")
		transformHTTPRequest(ep)
	})
}

// transformHTTPRequest rewrites the request-shaping keys shared by read and write
// endpoints.
func transformHTTPRequest(ep msgpack.EncodedJSON) {
	renameKey(ep, "queryParams", "query_params")
	recordToList(ep, "headers", "name", "value")
	recordToList(ep, "query_params", "parameter", "value")
}

// transformScan renames the legacy "scan_rate" key to "rate" and flips the enabled
// polarity.
func transformScan(config msgpack.EncodedJSON) {
	renameKey(config, "scan_rate", "rate")
	flipBool(config, "enabled", "disabled")
}

// transformPagerDuty flips the legacy per-alert enabled polarity.
func transformPagerDuty(config msgpack.EncodedJSON) {
	eachChild(config, "alerts", func(a msgpack.EncodedJSON) {
		flipBool(a, "enabled", "disabled")
	})
}
