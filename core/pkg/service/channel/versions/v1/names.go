// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"cmp"
	"context"
	"slices"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	group "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/set"
)

// NameRuleMigration renames each stored channel whose name a path cannot hold
// (resource.NeedsRename), or whose name an earlier channel in the same group holds, and
// stages each rename (resource.StageRename). Built-ins come first and keep their names,
// then the rest go by key. A new name is resource.SanitizeName of the old one, or
// "channel_<key>" when that is empty, with the first suffix (_2, _3) that no other
// channel holds. It runs after the step that names each unnamed channel.
var NameRuleMigration = migrate.New("v63_channel_name_rule", migrateNameRule)

// sibling is a name in a group, or in no group when group is zero.
type sibling struct {
	group group.Key
	name  string
}

func migrateNameRule(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	var channels []Channel
	if err := gorp.NewRetrieve[Key, Channel]().
		Entries(&channels).
		Exec(ctx, tx); err != nil {
		return err
	}
	slices.SortFunc(channels, func(a, b Channel) int {
		if a.Internal != b.Internal {
			if a.Internal {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.Key(), b.Key())
	})
	var (
		claimed = make(set.Set[sibling], len(channels))
		moves   = make([]bool, len(channels))
		taken   = make(set.Set[string], len(channels))
	)
	for i, c := range channels {
		if !c.Internal && resource.NeedsRename(c.Name) {
			moves[i] = true
			continue
		}
		at := sibling{name: resource.NormalizeName(c.Name)}
		if c.Group != nil {
			at.group = *c.Group
		}
		moves[i] = !c.Internal && claimed.Contains(at)
		claimed.Add(at)
		if !moves[i] {
			taken.Add(at.name)
		}
	}
	var renamed []Channel
	for i, c := range channels {
		if !moves[i] {
			continue
		}
		base := resource.SanitizeName(c.Name)
		if base == "" {
			base = "channel_" + c.Key().String()
		}
		name := resource.FreeName(base, taken.Contains)
		taken.Add(name)
		if err := resource.StageRename(ctx, tx, c.ResourceID(), c.Name); err != nil {
			return err
		}
		c.Name = name
		renamed = append(renamed, c)
	}
	return gorp.WrapWriter[Key, Channel](tx).Set(ctx, renamed...)
}
