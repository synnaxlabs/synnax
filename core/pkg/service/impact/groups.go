// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact

import (
	"bytes"
	"encoding/binary"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/group"
)

// groupsPlan returns the plan of c, a delete of groups, with no token. It has no
// entries, so a pick has no effect. A pick that is not a choice or a record fails as it
// fails Plan.
func groupsPlan(c Change, picks map[string]Choice, hidden *Choice) (Plan, error) {
	if len(picks) > 0 || hidden != nil {
		if _, err := newPicks(PlanOptions{Picks: picks, Hidden: hidden}); err != nil {
			return Plan{}, err
		}
	}
	return Plan{Change: c}, nil
}

// groupsSum returns the plan sum of p, the plan of the delete of the groups with keys,
// with the keys in sorted order.
func groupsSum(keys []group.Key, p Plan) uint64 {
	sorted := slices.Compact(slices.SortedFunc(slices.Values(keys), compareGroups))
	buf := appendField(nil, string(DeleteGroupsChangeType))
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(sorted)))
	for _, k := range sorted {
		buf = append(buf, k[:]...)
	}
	return planSum(buf, p)
}

func compareGroups(a, b group.Key) int { return bytes.Compare(a[:], b[:]) }
