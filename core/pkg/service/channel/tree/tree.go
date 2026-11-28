// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package tree makes the edits of the channel tree: channels and the channel folders
// that hold them.
package tree

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a Service.
type ServiceConfig struct {
	// Channel is the service of the channels in the tree.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Group is the service of the channel folders in the tree.
	//
	// [REQUIRED]
	Group *group.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("channel.tree")
	v.NotNil("channel", c.Channel)
	v.NotNil("group", c.Group)
	return v.Error()
}

// Service makes the edits of the channel tree.
type Service struct{ cfg ServiceConfig }

// OpenService opens a Service with the provided configurations.
func OpenService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

// DeleteMembers deletes the tombstones of the channel folders groups in tx, so a delete
// of the folders sees only their live channels. It is the group.DeleteMembers hook of
// the channel scope.
func (s *Service) DeleteMembers(
	ctx context.Context,
	tx gorp.Tx,
	groups []group.Key,
) error {
	return s.cfg.Channel.NewWriter(tx).DeleteTombstones(ctx, groups...)
}

// PlanDelete returns the plan of the delete of the channel folders groups, as seen by
// snap, and the token that ApplyDelete checks. Each tombstone in the folders is an
// entry
// with the effect delete and its path, and offers no choice. An entry whose record
// visible hides counts in Plan.Hidden, as impact.Seen does. A group that is not a
// channel folder fails with a validation error.
func (s *Service) PlanDelete(
	ctx context.Context,
	snap gorp.Snapshot,
	groups []group.Key,
	visible func(resource.ID) bool,
) (impact.Plan, error) {
	p, folders, err := s.planDelete(ctx, snap, groups, visible)
	if err != nil {
		return impact.Plan{}, err
	}
	p.Token = impact.Token(folders, p)
	return p, nil
}

// ApplyDelete deletes the channel folders groups and their tombstones in tx. It plans
// the delete in tx, as PlanDelete does, and fails with an impact.PlanChangedError
// before any write when token names another plan. An empty token skips the check, and
// the plan it returns then holds no entries. A folder that still holds a channel or a
// folder that groups does not name fails, as group.Writer.Delete does.
func (s *Service) ApplyDelete(
	ctx context.Context,
	tx gorp.Tx,
	groups []group.Key,
	token string,
	visible func(resource.ID) bool,
) (impact.Plan, error) {
	p, folders, err := s.planDelete(ctx, tx, groups, visible)
	if err != nil {
		return impact.Plan{}, err
	}
	if token != "" {
		if t := impact.Token(folders, p); t != token {
			p.Token = t
			return impact.Plan{}, impact.PlanChangedError{Plan: p}
		}
		p.Token = token
	}
	if err = s.cfg.Group.NewWriter(tx).Delete(ctx, groups...); err != nil {
		return impact.Plan{}, err
	}
	if token == "" {
		return impact.Plan{Records: folders}, nil
	}
	return p, nil
}

// ReplacedEntries returns an entry with the effect delete for each tombstone that the
// delete of the channels removed replaces, with the effects that its plan returned. It
// is the impact.Deleter.Entries of channels.
func (s *Service) ReplacedEntries(
	ctx context.Context,
	snap gorp.Snapshot,
	removed []resource.ID,
	effects gorp.DeleteEffects,
) ([]impact.Entry, error) {
	keys, err := channel.KeysFromResourceIDs(removed)
	if err != nil {
		return nil, err
	}
	replaced, err := s.cfg.Channel.Replaced(ctx, snap, keys, effects.Weak)
	if err != nil {
		return nil, err
	}
	return s.entries(ctx, snap, replaced)
}

// planDelete returns the plan of the delete of groups with no token, and the IDs of
// the folders.
func (s *Service) planDelete(
	ctx context.Context,
	snap gorp.Snapshot,
	groups []group.Key,
	visible func(resource.ID) bool,
) (impact.Plan, []resource.ID, error) {
	var folders []group.Group
	if err := s.cfg.Group.NewRetrieve().
		Where(group.MatchKeys(groups...)).
		Entries(&folders).
		Exec(ctx, snap); err != nil {
		return impact.Plan{}, nil, err
	}
	ids := make([]resource.ID, len(folders))
	for i, f := range folders {
		if f.Scope != resource.TypeChannel {
			return impact.Plan{}, nil, errors.Wrapf(
				validate.ErrValidation,
				"%s is not a channel folder",
				f.ResourceID(),
			)
		}
		ids[i] = f.ResourceID()
	}
	tombstones, err := s.cfg.Channel.Tombstones(ctx, snap, groups...)
	if err != nil {
		return impact.Plan{}, nil, err
	}
	entries, err := s.entries(ctx, snap, tombstones)
	if err != nil {
		return impact.Plan{}, nil, err
	}
	return impact.Seen(impact.Plan{Entries: entries}, ids, visible), ids, nil
}

// entries returns an entry with the effect delete and its path for each tombstone of
// tombstones, sorted by record.
func (s *Service) entries(
	ctx context.Context,
	snap gorp.Snapshot,
	tombstones []channel.Tombstone,
) ([]impact.Entry, error) {
	if len(tombstones) == 0 {
		return nil, nil
	}
	paths, err := s.paths(ctx, snap, tombstones)
	if err != nil {
		return nil, err
	}
	entries := make([]impact.Entry, len(tombstones))
	for i, t := range tombstones {
		entries[i] = impact.Entry{
			Record: channel.ResourceID(t.Key),
			Effect: impact.EffectDelete,
			Path:   &paths[i],
		}
	}
	slices.SortFunc(entries, func(a, b impact.Entry) int {
		return cmp.Compare(a.Record.Key, b.Record.Key)
	})
	return entries, nil
}

// paths returns the path of each tombstone of tombstones: the names of its folders from
// the top, then its own name, joined by dots.
func (s *Service) paths(
	ctx context.Context,
	snap gorp.Snapshot,
	tombstones []channel.Tombstone,
) ([]string, error) {
	keys := make(set.Set[group.Key])
	for _, t := range tombstones {
		if t.Group != nil {
			keys.Add(*t.Group)
		}
	}
	byKey := make(map[group.Key]group.Group, len(keys))
	if len(keys) > 0 {
		holding := keys.Slice()
		var folders []group.Group
		if err := s.cfg.Group.NewRetrieve().
			Where(group.MatchKeys(holding...)).
			Entries(&folders).
			Exec(ctx, snap); err != nil {
			return nil, err
		}
		above, err := s.cfg.Group.Ancestors(ctx, snap, holding...)
		if err != nil {
			return nil, err
		}
		for _, f := range append(folders, above...) {
			byKey[f.Key] = f
		}
	}
	paths := make([]string, len(tombstones))
	for i, t := range tombstones {
		segments := []string{t.Name}
		for g := t.Group; g != nil; {
			f := byKey[*g]
			segments = append(segments, f.Name)
			if f.Parent == nil || f.Parent.Type != resource.TypeGroup {
				break
			}
			parent, err := group.KeyFromResourceID(*f.Parent)
			if err != nil {
				return nil, err
			}
			g = &parent
		}
		slices.Reverse(segments)
		paths[i] = strings.Join(segments, ".")
	}
	return paths, nil
}
