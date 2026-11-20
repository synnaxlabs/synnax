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
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource/rename"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// NameRuleMigration renames each stored channel whose name breaks the name rule
// (validate.IsPathSegment), is not in NFC, or is the name of a channel with a lower key
// in the same group, and stages each rename (rename.Stage). A rewritten name drops what
// the rule refuses (validate.SanitizePathSegment), becomes "channel_<key>" when nothing
// is left, and takes the first free suffix when another channel in the Core holds it. A
// built-in channel keeps its name, so a channel that shares it in the same group moves.
// A legal name that no lower key holds never moves.
var NameRuleMigration = migrate.New("v63_channel_name_rule", migrateNameRule)

// sibling is a name in a group, or at the top when group is the zero key.
type sibling struct {
	group groupv1.Key
	name  string
}

func migrateNameRule(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	var channels []Channel
	if err := gorp.NewRetrieve[Key, Channel]().
		Entries(&channels).
		Exec(ctx, tx); err != nil {
		return err
	}
	// Built-ins claim their names first, then each channel in key order.
	slices.SortFunc(channels, func(a, b Channel) int {
		if a.Internal != b.Internal {
			if a.Internal {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.Key(), b.Key())
	})
	taken := make(set.Set[string], len(channels))
	for _, c := range channels {
		if isLegalName(c.Name) {
			taken.Add(c.Name)
		}
	}
	var (
		held    = make(set.Set[sibling], len(channels))
		renamed []Channel
		staged  []rename.Rename
	)
	for _, c := range channels {
		at := sibling{name: validate.NormalizeName(c.Name)}
		if c.Group != nil {
			at.group = *c.Group
		}
		if c.Internal || (isLegalName(c.Name) && !held.Contains(at)) {
			held.Add(at)
			continue
		}
		base := validate.SanitizePathSegment(c.Name)
		if base == "" {
			base = "channel_" + c.Key().String()
		}
		name := rename.Free(base, taken.Contains)
		taken.Add(name)
		held.Add(sibling{group: at.group, name: name})
		staged = append(staged, rename.Rename{
			Resource: c.ResourceID(),
			From:     c.Name,
			To:       name,
		})
		c.Name = name
		renamed = append(renamed, c)
	}
	if err := gorp.WrapWriter[Key, Channel](tx).Set(ctx, renamed...); err != nil {
		return err
	}
	return rename.Stage(ctx, tx, staged...)
}

// isLegalName reports whether name passes the name rule and is in NFC.
func isLegalName(name string) bool {
	return validate.IsPathSegment(name) && validate.IsNormalName(name)
}
