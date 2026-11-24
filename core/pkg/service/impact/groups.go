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
)

// planGroups returns the plan of c, the delete of the groups with keys, as snap sees
// it, with no token: each tombstone in the groups, as a cascade entry with its path,
// with those that opts.Visible hides counted in Hidden. Picks act as they act in Plan.
func (s *Service) planGroups(
	ctx context.Context,
	snap gorp.Snapshot,
	c Change,
	keys []group.Key,
	opts PlanOptions,
) (Plan, error) {
	chosen, picked, err := groupPicks(opts)
	if err != nil {
		return Plan{}, err
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
	return withHidden(p, opts.Visible), nil
}

// groupPicks parses the picks of opts for a change of groups, and reports whether opts
// holds a pick.
func groupPicks(opts PlanOptions) (picks, bool, error) {
	if len(opts.Picks) == 0 && opts.Hidden == nil {
		return picks{}, false, nil
	}
	chosen, err := newPicks(opts)
	return chosen, true, err
}

// refusePicks returns the validation error of the first pick of chosen that its entry
// among entries does not allow.
func refusePicks(chosen picks, entries []Entry) error {
	for _, e := range entries {
		if _, err := chosen.pickOf(e); err != nil {
			return err
		}
	}
	return nil
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
