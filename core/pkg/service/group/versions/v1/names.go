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
	"context"
	"slices"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/set"
)

// NameRuleMigration renames each stored channel folder whose name a path cannot hold
// (resource.NeedsRename), or whose name in NFC an earlier folder in the same parent
// holds, and stages each rename (resource.StageRename). Names in NFC come first, then
// the rest, each by key, which is a random order for v4 UUIDs. A new name is
// resource.SanitizeName of the old one, or "folder" when that is empty, with the first
// suffix (_2, _3) that no other folder in the same parent holds. It runs after the step
// that sets each parent.
var NameRuleMigration = migrate.New("v63_folder_name_rule", migrateNameRule)

// sibling is a name in a parent, or at the top when parent is zero.
type sibling struct {
	parent resource.ID
	name   string
}

func migrateNameRule(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	var groups []Group
	if err := gorp.NewRetrieve[Key, Group]().
		Where(gorp.Match(func(_ gorp.Context, g *Group) (bool, error) {
			return g.Scope == resource.TypeChannel, nil
		})).
		Entries(&groups).
		Exec(ctx, tx); err != nil {
		return err
	}
	slices.SortFunc(groups, func(a, b Group) int {
		// On a tie in NFC, the name that is in NFC keeps it.
		if aNFC, bNFC := resource.InNFC(a.Name), resource.InNFC(b.Name); aNFC != bNFC {
			if aNFC {
				return -1
			}
			return 1
		}
		return a.Key.Compare(b.Key)
	})
	var (
		claimed = make(set.Set[sibling], len(groups))
		moves   = make([]bool, len(groups))
		taken   = make(set.Set[sibling], len(groups))
	)
	for i, g := range groups {
		if resource.NeedsRename(g.Name) {
			moves[i] = true
			continue
		}
		at := sibling{parent: parentOf(g), name: resource.NormalizeName(g.Name)}
		moves[i] = claimed.Contains(at)
		claimed.Add(at)
		if !moves[i] {
			taken.Add(at)
		}
	}
	var renamed []Group
	for i, g := range groups {
		if !moves[i] {
			continue
		}
		parent := parentOf(g)
		base := resource.SanitizeName(g.Name)
		if base == "" {
			base = "folder"
		}
		name := resource.FreeName(base, func(name string) bool {
			return taken.Contains(sibling{parent: parent, name: name})
		})
		taken.Add(sibling{parent: parent, name: name})
		if err := resource.StageRename(ctx, tx, g.ResourceID(), g.Name); err != nil {
			return err
		}
		g.Name = name
		renamed = append(renamed, g)
	}
	return gorp.WrapWriter[Key, Group](tx).Set(ctx, renamed...)
}

func parentOf(g Group) resource.ID {
	if g.Parent == nil {
		return resource.ID{}
	}
	return *g.Parent
}
