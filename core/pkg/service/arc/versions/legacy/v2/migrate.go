// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import v1 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy/v1"

// Migrate transforms v1 arc state into v2 by renaming the deprecated set_status
// node props to status.set. The bare set_status STL symbol was removed; saved
// graphs must remap the props.
func Migrate(old v1.Data) Data {
	old.Graph.Props = rewriteSetStatus(old.Graph.Props)
	return old
}

func rewriteSetStatus(
	props map[string]map[string]any,
) map[string]map[string]any {
	out := make(map[string]map[string]any, len(props))
	for k, p := range props {
		if p["key"] != "set_status" {
			out[k] = p
			continue
		}
		out[k] = map[string]any{
			"key":         "status.set",
			"key_or_name": stringOr(p["statusKey"], ""),
			"variant":     stringOr(p["variant"], "success"),
			"message":     stringOr(p["message"], ""),
		}
	}
	return out
}

func stringOr(v any, def string) string {
	if s, ok := v.(string); ok {
		return s
	}
	return def
}
