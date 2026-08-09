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
// config-level device is copied onto every channel missing one, the renamed AI type
// alias is replaced, and the flat cold-junction fields collapse into their union.
func transformNIRead(config msgpack.EncodedJSON) {
	device, hasDevice := config["device"]
	delete(config, "device")
	eachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		if hasDevice {
			if _, ok := ch["device"]; !ok {
				ch["device"] = device
			}
		}
		if ch["type"] == "ai_frequency_voltage" {
			ch["type"] = "ai_freq_voltage"
		}
		collapseNICJC(ch)
	})
}

// collapseNICJC replaces a thermocouple channel's flat cjc_source, cjc_val, and
// cjc_port with the nested cjc union, keeping only the value the source selects. An
// unrecognized source falls back to the built-in sensor, which is what the Driver did
// with it before.
func collapseNICJC(ch msgpack.EncodedJSON) {
	source, hadSource := ch["cjc_source"]
	if !hadSource && ch["type"] != "ai_thermocouple" {
		return
	}
	val, port := ch["cjc_val"], ch["cjc_port"]
	delete(ch, "cjc_source")
	delete(ch, "cjc_val")
	delete(ch, "cjc_port")
	if _, taken := ch["cjc"]; taken {
		return
	}
	switch source {
	case "ConstVal":
		ch["cjc"] = map[string]any{"source": "const_val", "val": zeroIfNil(val)}
	case "Chan":
		ch["cjc"] = map[string]any{"source": "chan", "port": zeroIfNil(port)}
	default:
		ch["cjc"] = map[string]any{"source": "built_in"}
	}
}

func zeroIfNil(v any) any {
	if v == nil {
		return 0
	}
	return v
}

// transformOPCWrite renames the legacy v0 output channel key "channel" to
// "cmd_channel".
func transformOPCWrite(config msgpack.EncodedJSON) {
	eachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		renameKey(ch, "channel", "cmd_channel")
	})
}

// transformLabJackWrite renames the legacy output channel keys "cmd_key" and
// "state_key" to "cmd_channel" and "state_channel".
func transformLabJackWrite(config msgpack.EncodedJSON) {
	eachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		renameKey(ch, "cmd_key", "cmd_channel")
		renameKey(ch, "state_key", "state_channel")
	})
}

// transformEtherCAT normalizes the all-lowercase PDO sub-index spelling some legacy
// clients wrote, on channels and their nested addresses.
func transformEtherCAT(config msgpack.EncodedJSON) {
	normalize := func(m msgpack.EncodedJSON) {
		renameKey(m, "subindex", "sub_index")
	}
	eachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		normalize(ch)
		if addr, ok := ch["address"].(map[string]any); ok {
			normalize(addr)
		}
	})
}

// preTransformHTTP converts the legacy v0 record-shaped headers, query params, and
// enum values into their list shapes. It runs before the snake_case key pass because
// the record keys are data (header names, enum labels) that must not be converted.
func preTransformHTTP(config msgpack.EncodedJSON) {
	eachChild(config, "endpoints", func(ep msgpack.EncodedJSON) {
		recordToList(ep, "headers", "name", "value")
		recordToList(ep, "queryParams", "parameter", "value")
		recordToList(ep, "query_params", "parameter", "value")
		eachChild(ep, "fields", func(f msgpack.EncodedJSON) {
			recordToList(f, "enumValues", "label", "value")
			recordToList(f, "enum_values", "label", "value")
		})
	})
}

// transformHTTPRead flips the legacy field polarity on read endpoints.
func transformHTTPRead(config msgpack.EncodedJSON) {
	eachChild(config, "endpoints", func(ep msgpack.EncodedJSON) {
		eachChild(ep, "fields", func(f msgpack.EncodedJSON) {
			flipBool(f, "enabled", "disabled")
		})
	})
}

// transformHTTPWrite flips the legacy endpoint polarity on write endpoints.
func transformHTTPWrite(config msgpack.EncodedJSON) {
	eachChild(config, "endpoints", func(ep msgpack.EncodedJSON) {
		flipBool(ep, "enabled", "disabled")
	})
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
