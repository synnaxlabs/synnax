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
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/set"
)

// FolderNameRuleMigration renames each stored channel folder whose name a path cannot
// hold (resource.NeedsRename), or whose name in NFC an earlier folder in the same
// parent holds, and stages each rename (resource.StageRename). Names in NFC come first,
// then the rest, each by key, which is a random order for v4 UUIDs. A new name is
// resource.SanitizeName of the old one, or "folder" when that is empty, with the first
// suffix (_2, _3) that no other folder in the same parent holds. It writes each name
// that it keeps in NFC, with no staged rename. It leaves each top-level MetricsFolder
// to the clash pass, which knows the one that the Core uses, and each folder of another
// scope as it is. It runs after the step of the group table that sets each parent.
var FolderNameRuleMigration = migrate.New("v63_folder_name_rule", migrateFolderNameRule)

// folderSibling is a name in a parent, or at the top when parent is zero.
type folderSibling struct {
	parent resource.ID
	name   string
}

func migrateFolderNameRule(
	ctx context.Context,
	tx gorp.Tx,
	_ alamos.Instrumentation,
) error {
	var groups []groupv1.Group
	if err := gorp.NewRetrieve[groupv1.Key, groupv1.Group]().
		Where(gorp.Match(func(_ gorp.Context, g *groupv1.Group) (bool, error) {
			return g.Scope == resource.TypeChannel, nil
		})).
		Entries(&groups).
		Exec(ctx, tx); err != nil {
		return err
	}
	slices.SortFunc(groups, claimFolderOrder)
	moves, taken := markFolderMoves(groups)
	var changed []groupv1.Group
	for i, g := range groups {
		if !moves[i] {
			if nfc := resource.NormalizeName(g.Name); nfc != g.Name {
				g.Name = nfc
				changed = append(changed, g)
			}
			continue
		}
		parent := folderParent(g)
		base := resource.SanitizeName(g.Name)
		if base == "" {
			base = "folder"
		}
		name := resource.FreeName(base, func(name string) bool {
			return taken.Contains(folderSibling{parent: parent, name: name})
		})
		taken.Add(folderSibling{parent: parent, name: name})
		if err := resource.StageRename(
			ctx,
			tx,
			g.ResourceID(),
			g.Name,
			resource.RenameReason(g.Name),
		); err != nil {
			return err
		}
		g.Name = name
		changed = append(changed, g)
	}
	return gorp.WrapWriter[groupv1.Key, groupv1.Group](tx).Set(ctx, changed...)
}

// claimFolderOrder puts names in NFC first, then each by key, so each folder claims its
// name before the folders after it.
func claimFolderOrder(a, b groupv1.Group) int {
	// On a tie in NFC, the name that is in NFC keeps it.
	if aNFC, bNFC := resource.InNFC(a.Name), resource.InNFC(b.Name); aNFC != bNFC {
		if aNFC {
			return -1
		}
		return 1
	}
	return a.Key.Compare(b.Key)
}

// markFolderMoves reports which of groups, in claim order, take a new name, and returns
// the names that the others keep in each parent.
func markFolderMoves(groups []groupv1.Group) ([]bool, set.Set[folderSibling]) {
	var (
		claimed = make(set.Set[folderSibling], len(groups))
		moves   = make([]bool, len(groups))
		taken   = make(set.Set[folderSibling], len(groups))
		metrics = folderSibling{name: MetricsFolder}
	)
	for i, g := range groups {
		if resource.NeedsRename(g.Name) {
			moves[i] = true
			continue
		}
		at := folderSibling{
			parent: folderParent(g),
			name:   resource.NormalizeName(g.Name),
		}
		if at == metrics {
			taken.Add(at)
			continue
		}
		moves[i] = claimed.Contains(at)
		claimed.Add(at)
		if !moves[i] {
			taken.Add(at)
		}
	}
	return moves, taken
}

func folderParent(g groupv1.Group) resource.ID {
	if g.Parent == nil {
		return resource.ID{}
	}
	return *g.Parent
}
