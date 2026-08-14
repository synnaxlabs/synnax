// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy converts the NI config shapes released Consoles wrote: v0 stored the
// analog read device at the config level, v1 moved it onto each channel. Both eras
// are camelCase; shapes are discriminated structurally because released task files
// carry no version stamp.
package legacy

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/task/config/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// LastVersion is the newest legacy NI shape. The typed shape sits directly above it.
const LastVersion imex.Version = 1

// AnalogRead converts both released analog read shapes.
var AnalogRead = legacy.Rewrite{Post: analogRead}

// CounterRead converts the released counter read shape: an optional task-level
// device the schema no longer stores, and a measurement method spelling the Python
// client wrote on frequency and period channels that the Driver's own table never
// knew.
var CounterRead = legacy.Rewrite{Post: func(config msgpack.EncodedJSON) {
	pushDownDevice(config)
	legacy.EachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		t := ch["type"]
		if (t == "ci_frequency" || t == "ci_period") && ch["meas_method"] == "DynAvg" {
			ch["meas_method"] = "DynamicAvg"
		}
	})
}}

// Digital converts released digital read and write shapes: channels carried a type
// tag with one possible value, which the schema no longer stores.
var Digital = legacy.Rewrite{Post: func(config msgpack.EncodedJSON) {
	legacy.EachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		delete(ch, "type")
	})
}}

// Scanner converts the stored driver scan form.
var Scanner = legacy.Scan

// analogRead copies the config-level device onto channels, replaces the renamed AI
// type alias, rewrites the charge units, and collapses the flat cold-junction fields
// into their union.
func analogRead(config msgpack.EncodedJSON) {
	pushDownDevice(config)
	legacy.EachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		if ch["type"] == "ai_frequency_voltage" {
			ch["type"] = "ai_freq_voltage"
		}
		rewriteChargeUnits(ch)
		rewriteStrainValues(ch)
		collapseCJC(ch)
	})
}

// pushDownDevice moves a config-level device onto every channel without one. The
// released Python client serialized device as "" on channels it left unset, so an
// empty string counts as missing.
func pushDownDevice(config msgpack.EncodedJSON) {
	device, hasDevice := config["device"]
	delete(config, "device")
	if !hasDevice || device == "" {
		return
	}
	legacy.EachChild(config, "channels", func(ch msgpack.EncodedJSON) {
		if v, ok := ch["device"]; !ok || v == "" {
			ch["device"] = device
		}
	})
}

// strainConfigs maps the kebab-case bridge names Consoles 0.36 through 0.44 wrote to
// the DAQmx names. Console 0.44.6 flipped the constants in place with no migration,
// so stored rows keep the kebab spelling until redeployed.
var strainConfigs = map[string]string{
	"full-bridge-I":     "FullBridgeI",
	"full-bridge-II":    "FullBridgeII",
	"full-bridge-III":   "FullBridgeIII",
	"half-bridge-I":     "HalfBridgeI",
	"half-bridge-II":    "HalfBridgeII",
	"quarter-bridge-I":  "QuarterBridgeI",
	"quarter-bridge-II": "QuarterBridgeII",
}

// rewriteStrainValues replaces a strain gauge channel's kebab-case bridge names and
// Console 0.36's lowercase units with the DAQmx spellings.
func rewriteStrainValues(ch msgpack.EncodedJSON) {
	if ch["type"] != "ai_strain_gauge" {
		return
	}
	legacy.RemapValue(ch, "strain_config", strainConfigs)
	if ch["units"] == "strain" {
		ch["units"] = "Strain"
	}
}

// rewriteChargeUnits replaces a charge channel's released unit strings with the DAQmx
// names. "C" resolved to Celsius in the Driver's unit table, so a released charge
// channel measured temperature; "uC" resolved to nothing and DAQmx has no
// micro-coulomb unit, so PicoCoulombs is the nearest real one.
func rewriteChargeUnits(ch msgpack.EncodedJSON) {
	if ch["type"] != "ai_charge" {
		return
	}
	switch ch["units"] {
	case "C":
		ch["units"] = "Coulombs"
	case "uC":
		ch["units"] = "PicoCoulombs"
	}
}

// collapseCJC replaces a thermocouple channel's flat cjc_source, cjc_val, and
// cjc_port with the nested cjc union, keeping only the value the source selects. An
// unrecognized source falls back to the built-in sensor, which is what the Driver did
// with it before.
func collapseCJC(ch msgpack.EncodedJSON) {
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
