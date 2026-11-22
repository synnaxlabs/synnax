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
	"context"
	"encoding/binary"
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/set"
)

// planGroups returns the plan of c, the delete of the groups with keys, as snap sees
// it, with no token: each tombstone in the groups, as a cascade entry with its path. A
// pick fails as it fails Plan.
func (s *Service) planGroups(
	ctx context.Context,
	snap gorp.Snapshot,
	c Change,
	keys []group.Key,
	opts PlanOptions,
) (Plan, error) {
	var (
		chosen picks
		err    error
	)
	picked := len(opts.Picks) > 0 || opts.Hidden != nil
	if picked {
		if chosen, err = newPicks(opts); err != nil {
			return Plan{}, err
		}
	}
	dead, err := s.cfg.Channel.Tombstones(ctx, snap, keys...)
	if err != nil {
		return Plan{}, err
	}
	p := Plan{Change: c}
	if len(dead) > 0 {
		paths, err := s.tombstonePaths(ctx, snap, keys, dead)
		if err != nil {
			return Plan{}, err
		}
		p.Entries = make([]Entry, len(dead))
		for i, t := range dead {
			p.Entries[i] = Entry{
				Record: channel.ResourceID(t.Key),
				Effect: EffectCascade,
				Path:   &paths[i],
			}
		}
		slices.SortFunc(p.Entries, compareEntries)
	}
	if picked {
		if err = refusePicks(chosen, p.Entries); err != nil {
			return Plan{}, err
		}
	}
	return p, nil
}

// refusePicks returns the validation error of the first pick of chosen that its entry
// among entries does not allow, or then of the smallest pick that names no entry.
func refusePicks(chosen picks, entries []Entry) error {
	named := make(set.Set[resource.ID], len(entries))
	for _, e := range entries {
		if _, err := chosen.pickOf(e); err != nil {
			return err
		}
		named.Add(e.Record)
	}
	return pickWithoutEntry(chosen.named, named)
}

// groupsSum returns the plan sum of p, the plan of the change of type t to the groups
// with keys, with the keys in sorted order.
func groupsSum(t ChangeType, keys []group.Key, p Plan) uint64 {
	sorted := slices.Compact(slices.SortedFunc(slices.Values(keys), compareGroups))
	buf := appendField(nil, string(t))
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(sorted)))
	for _, k := range sorted {
		buf = append(buf, k[:]...)
	}
	return planSum(buf, p)
}

func compareGroups(a, b group.Key) int { return bytes.Compare(a[:], b[:]) }
