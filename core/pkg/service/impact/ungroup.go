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
	"context"
	"maps"
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// ungrouping is the plan of an ungroup and what applying it moves.
type ungrouping struct {
	// plan lists each tombstone that drops, with no token.
	plan Plan
	// groups are the groups that the ungroup deletes, sorted by key.
	groups []group.Group
	// moves maps each of groups to its parent group, or to nil.
	moves map[group.Key]*group.Key
}

// planUngroup returns the plan of c, the ungroup of the groups with keys, as Plan
// describes, with no token.
func (s *Service) planUngroup(
	ctx context.Context,
	snap gorp.Snapshot,
	c Change,
	keys []group.Key,
	opts PlanOptions,
) (ungrouping, error) {
	chosen, picked, err := groupPicks(opts)
	if err != nil {
		return ungrouping{}, err
	}
	u := ungrouping{plan: Plan{Change: c}}
	names, err := s.ungrouped(ctx, snap, keys, &u)
	if err != nil {
		return ungrouping{}, err
	}
	dropped, err := s.cfg.Channel.DroppedTombstones(ctx, snap, u.moves)
	if err != nil {
		return ungrouping{}, err
	}
	if len(dropped) > 0 {
		u.plan.Entries = make([]Entry, len(dropped))
		for i, t := range dropped {
			path := entryPath(names[*t.Group], t.Name)
			u.plan.Entries[i] = Entry{
				Record: channel.ResourceID(t.Key),
				Effect: EffectCascade,
				Path:   &path,
			}
		}
		slices.SortFunc(u.plan.Entries, compareEntries)
	}
	if picked {
		if err = refusePicks(chosen, u.plan.Entries); err != nil {
			return ungrouping{}, err
		}
	}
	u.plan = withHidden(u.plan, opts.Visible)
	return u, nil
}

// ungrouped fills the groups and moves of u and returns each group's name by key. A
// group whose parent is among keys fails with a validation error.
func (s *Service) ungrouped(
	ctx context.Context,
	snap gorp.Snapshot,
	keys []group.Key,
	u *ungrouping,
) (map[group.Key]string, error) {
	if err := s.cfg.Group.NewRetrieve().
		Where(group.MatchKeys(keys...)).
		Entries(&u.groups).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	slices.SortFunc(u.groups, func(a, b group.Group) int {
		return compareGroups(a.Key, b.Key)
	})
	u.moves = make(map[group.Key]*group.Key, len(u.groups))
	names := make(map[group.Key]string, len(u.groups))
	for _, g := range u.groups {
		u.moves[g.Key] = nil
		names[g.Key] = g.Name
	}
	for _, g := range u.groups {
		if g.Parent == nil || g.Parent.Type != resource.TypeGroup {
			continue
		}
		parent, err := group.KeyFromResourceID(*g.Parent)
		if err != nil {
			return nil, err
		}
		if _, ok := names[parent]; ok {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"cannot ungroup %s and %s, which holds it, in one change",
				g.Name,
				names[parent],
			)
		}
		u.moves[g.Key] = &parent
	}
	return names, nil
}

// ungroupGroups applies c, the ungroup of the groups with keys, as Apply describes.
func (s *Service) ungroupGroups(
	ctx context.Context,
	tx gorp.Tx,
	c Change,
	keys []group.Key,
	opts ApplyOptions,
) (Plan, error) {
	u, err := s.planUngroup(ctx, tx, c, keys, PlanOptions{
		Picks:   opts.Picks,
		Hidden:  opts.Hidden,
		Visible: opts.Visible,
	})
	if err != nil {
		return Plan{}, err
	}
	if opts.Token != "" {
		// A change of its own keeps c off the heap when no token comes.
		now := u.plan
		now.Change = Change{Variant: UngroupGroupsChange{Keys: keys}}
		sum := groupsSum(UngroupGroupsChangeType, keys, now)
		if err = checkToken(opts.Token, sum, now); err != nil {
			return Plan{}, err
		}
		u.plan.Token = opts.Token
	}
	for _, g := range u.groups {
		if err = s.moveChildren(ctx, tx, g, u.moves[g.Key]); err != nil {
			return Plan{}, err
		}
	}
	if _, err = s.cfg.Channel.NewWriter(tx).MoveTombstones(ctx, u.moves); err != nil {
		return Plan{}, err
	}
	if err = s.cfg.Group.NewWriter(tx).Delete(ctx, keys...); err != nil {
		return Plan{}, err
	}
	return u.plan, nil
}

// moveChildren moves the groups in g to the parent of g, and its other members except
// tombstones into the group to, or out of any group for nil.
func (s *Service) moveChildren(
	ctx context.Context,
	tx gorp.Tx,
	g group.Group,
	to *group.Key,
) error {
	children, err := s.cfg.Group.Children(ctx, tx, g.Key)
	if err != nil {
		return err
	}
	var (
		subgroups []group.Key
		members   = make(map[resource.Type][]resource.ID)
	)
	for _, id := range children {
		if id.Type == resource.TypeGroup {
			k, err := group.KeyFromResourceID(id)
			if err != nil {
				return err
			}
			subgroups = append(subgroups, k)
		} else if !channel.IsTombstone(id) {
			members[id.Type] = append(members[id.Type], id)
		}
	}
	if len(subgroups) > 0 {
		w := s.cfg.Group.NewWriter(tx)
		if err = w.SetParent(ctx, g.Parent, subgroups...); err != nil {
			return err
		}
	}
	for _, t := range slices.Sorted(maps.Keys(members)) {
		move, ok := s.cfg.Moves[t]
		if !ok {
			return errors.Newf("[impact] - no move for a %s", t)
		}
		if err = move(ctx, tx, members[t], to); err != nil {
			return err
		}
	}
	return nil
}
