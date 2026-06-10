// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"strings"
	"unicode"

	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/set"
)

// opaqueConfigFields names element config fields whose contents carry semantic
// keys (telem pipeline segment names, custom symbol state shapes) and must not
// be case-converted.
var opaqueConfigFields = set.New("props", "state_overrides")

// normalizeConfigKeys converts a config payload's field keys from the
// camelCase the Console writes verbatim to the snake_case wire form of the
// element config union. Already snake_case keys pass through unchanged, so the
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

// extractTelemArgs rewrites a config's stored telem pipeline specs into the
// semantic arguments the schema now declares, in place on the normalized wire
// map. Channel keys are read from the pipeline's well-known segments; a zero
// channel (the legacy unset default) produces no argument. Spec fields are
// removed regardless so the entry decodes under the args-based schema.
func extractTelemArgs(cfg map[string]any) {
	variant, _ := cfg["variant"].(string)
	switch variant {
	case "value", "gauge":
		if ch, ok := segProp(cfg["telem"], "valueStream", "channel"); ok {
			cfg["channel"] = ch
		}
		if w, ok := segProp(cfg["telem"], "rollingAverage", "windowSize"); ok {
			cfg["rolling_average"] = w
		}
		delete(cfg, "telem")
		delete(cfg, "background_telem")
	case "light":
		if ch, ok := segProp(cfg["source"], "valueStream", "channel"); ok {
			cfg["channel"] = ch
		}
		if b, ok := segProp(cfg["source"], "threshold", "trueBound"); ok {
			cfg["threshold"] = b
		}
		delete(cfg, "source")
	case "stateIndicator":
		if ch, ok := segProp(cfg["source"], "valueStream", "channel"); ok {
			cfg["channel"] = ch
		}
		delete(cfg, "source")
	case "setpoint":
		if ch, ok := segProp(cfg["source"], "valueStream", "channel"); ok {
			cfg["state_channel"] = ch
		}
		if ch, ok := segProp(cfg["sink"], "setter", "channel"); ok {
			cfg["command_channel"] = ch
		}
		delete(cfg, "source")
		delete(cfg, "sink")
	case "button", "select", "input":
		if ch, ok := segProp(cfg["sink"], "setter", "channel"); ok {
			cfg["command_channel"] = ch
		}
		delete(cfg, "sink")
	default:
		if _, ok := cfg["source"]; ok {
			if ch, k := segProp(cfg["source"], "valueStream", "channel"); k {
				cfg["state_channel"] = ch
			}
			delete(cfg, "source")
		}
		if _, ok := cfg["sink"]; ok {
			if ch, k := segProp(cfg["sink"], "setter", "channel"); k {
				cfg["command_channel"] = ch
			}
			delete(cfg, "sink")
		}
	}
	if ctl, ok := cfg["control"].(map[string]any); ok {
		if chip, ok := ctl["chip"].(map[string]any); ok {
			if sink, ok := chip["sink"].(map[string]any); ok {
				if props, ok := sink["props"].(map[string]any); ok {
					if a, ok := props["authority"]; ok {
						ctl["authority"] = a
					}
				}
			}
		}
		delete(ctl, "chip")
		delete(ctl, "indicator")
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
