// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"strings"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// MatchPaths returns a filter for the channels at paths. A path is the names of the
// folders from the top, then the name of the channel, joined by validate.PathSeparator,
// in any normal form. A path that no channel is at, or that passes a folder that
// Retrieve.AllowFolders refuses, matches nothing. Each segment costs one probe of the
// names in a folder, and the filter reads no record to find the keys. It checks the
// folder and the name of each channel it decodes, so a channel that a commit in flight
// renames or moves shows as storage holds it. A folder that a commit in flight renames
// shows under its old name until the commit flushes.
func MatchPaths(paths ...string) Filter {
	return func(r Retrieve) gorp.Filter[Key, Channel] {
		return gorp.MatchKeysChecked[Key, Channel](func(ctx gorp.Context) (
			[]Key,
			func(*Channel) bool,
			error,
		) {
			var (
				keys   = make([]Key, 0, len(paths))
				wanted = make(set.Set[group.Sibling], len(paths))
			)
			for _, p := range paths {
				s, found, err := r.siblingAt(
					ctx,
					ctx.Snapshot,
					validate.NormalizeName(p),
				)
				if err != nil {
					return nil, nil, err
				}
				if !found {
					continue
				}
				wanted.Add(s)
				key, held, err := r.indexes.sibling.Key(ctx, ctx.Snapshot, s)
				if err != nil {
					return nil, nil, err
				}
				if held {
					keys = append(keys, key)
				}
				keys = append(keys, r.indexes.sibling.Committing(s)...)
			}
			return keys, func(ch *Channel) bool { return wanted.Contains(siblingOf(ch)) }, nil
		})
	}
}

// MatchPathSet returns the filter of the path line of a Where: the channels at the path
// set.Eq and at a path in set.In, for each operator that is set.
func MatchPathSet(set filter.Set[string]) Filter {
	var fs []Filter
	if set.Eq != nil {
		fs = append(fs, MatchPaths(*set.Eq))
	}
	if set.In != nil {
		fs = append(fs, MatchPaths(set.In...))
	}
	return And(fs...)
}

// siblingOf returns the folder and the name in NFC of ch.
func siblingOf(ch *Channel) group.Sibling {
	s := group.Sibling{Name: validate.NormalizeName(ch.Name)}
	if ch.Group != nil {
		s.Parent = *ch.Group
	}
	return s
}

// siblingAt returns the folder of the last segment of path and that segment, as seen
// by snap. It returns false when a folder on the path does not exist or r does not
// allow
// it.
func (r Retrieve) siblingAt(ctx context.Context, snap gorp.Snapshot, path string) (
	group.Sibling,
	bool,
	error,
) {
	var parent group.Key
	for {
		i := strings.IndexByte(path, validate.PathSeparator)
		if i < 0 {
			return group.Sibling{Parent: parent, Name: path}, true, nil
		}
		folder := group.Sibling{Parent: parent, Name: path[:i]}
		holder, found, err := r.indexes.sibling.Holder(ctx, snap, folder)
		if err != nil || !found {
			return group.Sibling{}, false, err
		}
		switch holder.Table {
		case tableName:
			return group.Sibling{}, false, nil
		case string(resource.TypeGroup):
			id := resource.ID{Type: resource.TypeGroup, Key: holder.Key}
			if parent, err = group.KeyFromResourceID(id); err != nil {
				return group.Sibling{}, false, err
			}
			if r.folderAllowed != nil && !r.folderAllowed(id) {
				return group.Sibling{}, false, nil
			}
		default:
			return group.Sibling{}, false, errors.Newf(
				"%s %s holds a name among channels",
				holder.Table,
				holder.Key,
			)
		}
		path = path[i+1:]
	}
}
