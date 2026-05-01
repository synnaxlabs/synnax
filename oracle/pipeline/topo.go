// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pipeline

import (
	"sort"

	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/x/set"
)

// topoLevels groups plugins into levels where every plugin in level N is
// satisfied by levels < N. Plugins within a level run concurrently and are
// sorted by name to keep generation order deterministic. Cyclic or unknown
// dependencies fall through to the final level so the regular Check error
// path can surface the actual failure.
func topoLevels(registry *plugin.Registry) [][]plugin.Plugin {
	remaining := make(map[string]plugin.Plugin)
	for _, p := range registry.All() {
		remaining[p.Name()] = p
	}
	var levels [][]plugin.Plugin
	placed := set.New[string]()
	for len(remaining) > 0 {
		var level []plugin.Plugin
		for _, p := range remaining {
			satisfied := true
			for _, dep := range p.Requires() {
				if placed.Contains(dep) {
					continue
				}
				if _, exists := remaining[dep]; exists {
					satisfied = false
					break
				}
			}
			if satisfied {
				level = append(level, p)
			}
		}
		if len(level) == 0 {
			for _, p := range remaining {
				level = append(level, p)
			}
		}
		sort.Slice(level, func(i, j int) bool { return level[i].Name() < level[j].Name() })
		for _, p := range level {
			placed.Add(p.Name())
			delete(remaining, p.Name())
		}
		levels = append(levels, level)
	}
	return levels
}
