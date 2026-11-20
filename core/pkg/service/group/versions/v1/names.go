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
	"strings"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/resource/rename"
	versions "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// NameRuleMigration renames each stored channel group whose name breaks the name rule
// (validate.IsPathSegment), is not in NFC, or is the name of a group that comes first
// by
// key in the same parent, and stages each rename (rename.Stage). A rewritten name drops
// what the rule refuses (validate.SanitizePathSegment), becomes "folder" when nothing
// is left, and takes the first suffix free among its siblings. A legal name that no
// earlier sibling holds never moves.
var NameRuleMigration = migrate.New("v63_folder_name_rule", migrateFolderNames)

// folderSibling is a name in a parent, or at the top when parent is zero.
type folderSibling struct {
	parent versions.ID
	name   string
}

func migrateFolderNames(
	ctx context.Context,
	tx gorp.Tx,
	_ alamos.Instrumentation,
) error {
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
		return strings.Compare(a.Key.String(), b.Key.String())
	})
	taken := make(set.Set[folderSibling], len(groups))
	for _, g := range groups {
		if validate.IsPathSegment(g.Name) && validate.IsNormalName(g.Name) {
			taken.Add(folderSibling{parent: parentOf(g), name: g.Name})
		}
	}
	var (
		held    = make(set.Set[folderSibling], len(groups))
		renamed []Group
		staged  []rename.Rename
	)
	for _, g := range groups {
		at := folderSibling{parent: parentOf(g), name: validate.NormalizeName(g.Name)}
		if validate.IsPathSegment(g.Name) && validate.IsNormalName(g.Name) &&
			!held.Contains(at) {
			held.Add(at)
			continue
		}
		base := validate.SanitizePathSegment(g.Name)
		if base == "" {
			base = "folder"
		}
		name := rename.Free(base, func(n string) bool {
			return taken.Contains(folderSibling{parent: at.parent, name: n})
		})
		next := folderSibling{parent: at.parent, name: name}
		taken.Add(next)
		held.Add(next)
		staged = append(staged, rename.Rename{
			Resource: resource.ID{Type: resource.TypeGroup, Key: g.Key.String()},
			From:     g.Name,
			To:       name,
		})
		g.Name = name
		renamed = append(renamed, g)
	}
	if err := gorp.WrapWriter[Key, Group](tx).Set(ctx, renamed...); err != nil {
		return err
	}
	return rename.Stage(ctx, tx, staged...)
}

func parentOf(g Group) versions.ID {
	if g.Parent == nil {
		return versions.ID{}
	}
	return *g.Parent
}
