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

// transform rewrites a legacy config blob in place toward the shape the task type's
// config record decodes.
type transform func(config msgpack.EncodedJSON)

// transforms maps task types to their legacy-shape transform. Types absent from the
// map need only the common transforms.
var transforms = map[string]transform{
	"ni_analog_read":  transformNIRead,
	"ni_counter_read": transformNIRead,
	"opc_write":       transformOPCWrite,
	"labjack_write":   transformLabJackWrite,
	"ethercat_read":   transformEtherCAT,
	"ethercat_write":  transformEtherCAT,
	"http_read":       transformHTTPRead,
	"http_write":      transformHTTPWrite,
	"pagerduty_alert": transformPagerDuty,
	"opc_scan":        transformScan,
	"labjack_scan":    transformScan,
	"modbus_scan":     transformScan,
	"ethercat_scan":   transformScan,
	"http_scan":       transformScan,
	"ni_scanner":      transformScan,
}

// Transform returns config normalized for the given task type: common legacy keys
// are rewritten first, then the type's own transform runs. The top-level map is
// copied; nested structures are rewritten in place.
func Transform(taskType string, config msgpack.EncodedJSON) msgpack.EncodedJSON {
	out := make(msgpack.EncodedJSON, len(config))
	for k, v := range config {
		out[k] = v
	}
	transformCommon(out)
	if f, ok := transforms[taskType]; ok {
		f(out)
	}
	return out
}

// transformCommon rewrites the legacy keys every integration shared: camelCase base
// fields, the data_saving polarity flip, and the per-channel enabled polarity flip.
func transformCommon(config msgpack.EncodedJSON) {
	renameKey(config, "autoStart", "auto_start")
	renameKey(config, "dataSaving", "data_saving")
	renameKey(config, "sampleRate", "sample_rate")
	renameKey(config, "streamRate", "stream_rate")
	renameKey(config, "stateRate", "state_rate")
	flipBool(config, "data_saving", "data_saving_disabled")
	eachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		flipBool(ch, "enabled", "disabled")
	})
}

// renameKey moves the value under from to under to. An existing value under to
// wins; the legacy key is dropped either way.
func renameKey(m msgpack.EncodedJSON, from, to string) {
	v, ok := m[from]
	if !ok {
		return
	}
	delete(m, from)
	if _, taken := m[to]; !taken {
		m[to] = v
	}
}

// flipBool replaces the boolean under from with its negation under to. A non-bool
// value under from is dropped.
func flipBool(m msgpack.EncodedJSON, from, to string) {
	v, ok := m[from]
	if !ok {
		return
	}
	delete(m, from)
	b, ok := v.(bool)
	if !ok {
		return
	}
	if _, taken := m[to]; !taken {
		m[to] = !b
	}
}

// eachChild applies f to every object element of the list stored under key.
func eachChild(m msgpack.EncodedJSON, key string, f func(msgpack.EncodedJSON)) {
	list, ok := m[key].([]any)
	if !ok {
		return
	}
	for _, el := range list {
		if child, ok := el.(map[string]any); ok {
			f(child)
		}
	}
}

// recordToList converts the record stored under key into a list of objects, one per
// record entry, with the record key under keyField and its value under valueField.
func recordToList(m msgpack.EncodedJSON, key, keyField, valueField string) {
	rec, ok := m[key].(map[string]any)
	if !ok {
		return
	}
	list := make([]any, 0, len(rec))
	for k, v := range rec {
		list = append(list, map[string]any{keyField: k, valueField: v})
	}
	m[key] = list
}
