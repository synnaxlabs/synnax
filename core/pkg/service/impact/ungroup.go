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
	// moves maps the key of each of groups to the key of its parent, or to nil when its
	// parent is not a group.
	moves map[group.Key]*group.Key
}

// planUngroup returns the plan of c, the ungroup of the groups with keys, as snap sees
// it, with no token: each tombstone that drops because its new path is taken, as a
// cascade entry with its path from its group. A group whose parent is among keys fails
// with a validation error, and a pick fails as it fails Plan.
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
			path := names[*t.Group] + "." + t.Name
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
	return u, nil
}

// ungrouped sets the groups and moves of u for the ungroup of the groups with keys, as
// snap sees them, and returns the name of each group by key. A group whose parent is
// among keys fails with a validation error.
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

// ungroupGroups moves the members, child groups, and tombstones of the groups with
// keys, the groups of c, to the parent of each, and then deletes the groups, as Apply
// describes.
func (s *Service) ungroupGroups(
	ctx context.Context,
	tx gorp.Tx,
	c Change,
	keys []group.Key,
	opts ApplyOptions,
) (Plan, error) {
	u, err := s.planUngroup(ctx, tx, c, keys, PlanOptions{
		Picks:  opts.Picks,
		Hidden: opts.Hidden,
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

// moveChildren moves each group in g to the parent of g, and each other member of g
// except its tombstones into the group to, or out of any group for nil.
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
		members   []resource.ID
	)
	for _, id := range children {
		switch id.Type {
		case resource.TypeGroup:
			k, err := group.KeyFromResourceID(id)
			if err != nil {
				return err
			}
			subgroups = append(subgroups, k)
		case resource.Type(channel.TombstoneTable):
		default:
			members = append(members, id)
		}
	}
	if len(subgroups) > 0 {
		w := s.cfg.Group.NewWriter(tx)
		if err = w.SetParent(ctx, g.Parent, subgroups...); err != nil {
			return err
		}
	}
	if len(members) == 0 {
		return nil
	}
	return s.cfg.Move(ctx, tx, members, to)
}
