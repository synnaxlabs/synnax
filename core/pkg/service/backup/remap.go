// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package backup

import (
	"encoding/json"

	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
)

// remapChannelKeys walks a JSON blob and replaces any numeric value that matches
// an entry in the remap table with its new value. Channel keys are uint32 values
// typically > 1,000,000 (due to node_key << 20), so collisions with regular
// numeric values (pixel positions, colors, etc.) are extremely unlikely.
func remapChannelKeys(data json.RawMessage, remap map[distchannel.Key]distchannel.Key) (json.RawMessage, error) {
	if len(remap) == 0 || len(data) == 0 {
		return data, nil
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	walked := walk(v, remap)
	return json.Marshal(walked)
}

func walk(v any, remap map[distchannel.Key]distchannel.Key) any {
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			val[k] = walk(child, remap)
		}
		return val
	case []any:
		for i, child := range val {
			val[i] = walk(child, remap)
		}
		return val
	case float64:
		key := distchannel.Key(val)
		if float64(key) == val {
			if newKey, ok := remap[key]; ok {
				return float64(newKey)
			}
		}
		return val
	default:
		return val
	}
}
