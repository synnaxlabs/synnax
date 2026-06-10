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
