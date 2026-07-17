// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"fmt"
	"sort"
	"strings"

	"github.com/synnaxlabs/arc/types"
)

// treePrefix returns the prefix for a tree item. If last is true, returns "└── ",
// otherwise "├── ".
func treePrefix(last bool) string {
	if last {
		return "└── "
	}
	return "├── "
}

func treeIndent(last bool) string {
	if last {
		return "    "
	}
	return "│   "
}

// formatParams formats a slice of Params as "name (type), name (type), ..."
func formatParams(params types.Params) string {
	if len(params) == 0 {
		return "(none)"
	}
	parts := make([]string, len(params))
	for i, p := range params {
		if p.Value != nil {
			parts[i] = fmt.Sprintf("%s (%s) = %v", p.Name, p.Type.String(), p.Value)
		} else {
			parts[i] = fmt.Sprintf("%s (%s)", p.Name, p.Type.String())
		}
	}
	return strings.Join(parts, ", ")
}

// formatChannels formats Channels as "read [id: name, ...], write [id: name, ...]"
func formatChannels(ch types.Channels) string {
	if len(ch.Read) == 0 && len(ch.Write) == 0 {
		return "(none)"
	}
	var parts []string
	if len(ch.Read) > 0 {
		readParts := make([]string, 0, len(ch.Read))
		for id, name := range ch.Read {
			readParts = append(readParts, fmt.Sprintf("%d: %s", id, name))
		}
		sort.Strings(readParts)
		parts = append(parts, "read ["+strings.Join(readParts, ", ")+"]")
	}
	if len(ch.Write) > 0 {
		writeParts := make([]string, 0, len(ch.Write))
		for id, name := range ch.Write {
			writeParts = append(writeParts, fmt.Sprintf("%d: %s", id, name))
		}
		sort.Strings(writeParts)
		parts = append(parts, "write ["+strings.Join(writeParts, ", ")+"]")
	}
	return strings.Join(parts, ", ")
}
