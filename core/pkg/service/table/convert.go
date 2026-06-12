// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"strings"
	"unicode"

	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/set"
)

// opaqueConfigFields names cell config fields whose contents carry semantic
// keys (telem pipeline segment names) and must not be case-converted.
var opaqueConfigFields = set.New("props")

// normalizeConfigKeys converts a cell config payload's field keys from the
// camelCase the Console wrote verbatim to the snake_case wire form of the cell
// config union. Already snake_case input passes through unchanged, so the
// conversion is idempotent. Values under opaque fields are left untouched.
func normalizeConfigKeys(raw msgpack.EncodedJSON) msgpack.EncodedJSON {
	if raw == nil {
		return nil
	}
	out, _ := normalizeConfigValue(map[string]any(raw)).(map[string]any)
	return out
}

func normalizeConfigValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			nk := camelToSnakeKey(k)
			if opaqueConfigFields.Contains(nk) {
				out[nk] = val
				continue
			}
			out[nk] = normalizeConfigValue(val)
		}
		return out
	case []any:
		for i := range t {
			t[i] = normalizeConfigValue(t[i])
		}
		return t
	default:
		return v
	}
}

// camelToSnakeKey converts a camelCase identifier to snake_case, leaving
// already snake_case identifiers unchanged.
func camelToSnakeKey(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// legacyAligns maps the x-location alignment values the pre-typed text cell
// schema declared onto the flex alignments the alignment form control has
// always written.
var legacyAligns = map[string]string{"left": "start", "right": "end"}

// legacyWeights maps named CSS font weights onto the numeric weights the
// weight form control writes.
var legacyWeights = map[string]float64{
	"lighter": 300,
	"normal":  400,
	"bold":    700,
	"bolder":  700,
}

// extractLegacyArgs rewrites a stored cell config's legacy fields into the
// semantic arguments the schema now declares, in place on the normalized wire
// map. Value cells extract their channel, rolling average, precision, and
// notation from the stored telem pipeline spec; a zero channel (the legacy
// unset default) produces no argument. Spec fields are removed regardless so
// the entry decodes under the args-based schema. Text cells map legacy
// x-location alignments and named font weights onto their typed counterparts.
func extractLegacyArgs(cfg map[string]any) {
	variant, _ := cfg["variant"].(string)
	switch variant {
	case "value":
		if ch, ok := segProp(cfg["telem"], "valueStream", "channel"); ok {
			cfg["channel"] = ch
		}
		if w, ok := segProp(cfg["telem"], "rollingAverage", "windowSize"); ok {
			cfg["rolling_average"] = w
		}
		if p, ok := segProp(cfg["telem"], "stringifier", "precision"); ok {
			cfg["precision"] = p
		}
		if n, ok := segProp(cfg["telem"], "stringifier", "notation"); ok {
			cfg["notation"] = n
		}
		delete(cfg, "telem")
	case "text":
		if a, ok := cfg["align"].(string); ok {
			if mapped, ok := legacyAligns[a]; ok {
				cfg["align"] = mapped
			}
		}
		if w, ok := cfg["weight"].(string); ok {
			if mapped, ok := legacyWeights[w]; ok {
				cfg["weight"] = mapped
			}
		}
	}
}

// segProp reads a property from a named segment of a stored pipeline spec,
// reporting false when any layer is missing or the value is a zero channel.
func segProp(spec any, segment, prop string) (any, bool) {
	m, ok := spec.(map[string]any)
	if !ok {
		return nil, false
	}
	props, ok := m["props"].(map[string]any)
	if !ok {
		return nil, false
	}
	segments, ok := props["segments"].(map[string]any)
	if !ok {
		// Single-segment pipelines store the spec at the top level.
		segments = map[string]any{segment: m}
	}
	seg, ok := segments[segment].(map[string]any)
	if !ok {
		return nil, false
	}
	segProps, ok := seg["props"].(map[string]any)
	if !ok {
		return nil, false
	}
	v, ok := segProps[prop]
	if !ok {
		return nil, false
	}
	if prop == "channel" {
		if n, isNum := v.(float64); isNum && n == 0 {
			return nil, false
		}
	}
	return v, true
}
