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
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// MatchPaths returns a filter for the channels at paths. A path is the names of the
// folders from the top, then the name of the channel, joined by validate.PathSeparator,
// in any normal form. A path that no channel is at, or that passes a folder that
// Retrieve.AllowFolders refuses, matches nothing. Each folder costs one probe of the
// names in a folder, and the channel one probe and one keyed read, so a channel that a
// commit in flight renames or moves shows as storage holds it. A folder that a commit
// in flight renames shows under its old name until the commit flushes.
func MatchPaths(paths ...string) Filter {
	return func(r Retrieve) gorp.Filter[Key, Channel] {
		return gorp.MatchKeysFunc[Key, Channel](func(ctx gorp.Context) ([]Key, error) {
			keys := make([]Key, 0, len(paths))
			for _, p := range paths {
				key, found, err := r.keyAt(ctx, ctx.Snapshot, validate.NormalizeName(p))
				if err != nil {
					return nil, err
				}
				if found {
					keys = append(keys, key)
				}
			}
			return keys, nil
		})
	}
}

// keyAt returns the key of the channel at path, as seen by snap, and false when no
// channel is at it or the path passes a folder that r does not allow.
func (r Retrieve) keyAt(ctx context.Context, snap gorp.Snapshot, path string) (
	Key,
	bool,
	error,
) {
	var parent group.Key
	for {
		segment, rest := path, ""
		i := strings.IndexByte(path, validate.PathSeparator)
		if i >= 0 {
			segment, rest = path[:i], path[i+1:]
		}
		sibling := group.Sibling{Parent: parent, Name: segment}
		holder, found, err := r.indexes.sibling.Holder(ctx, snap, sibling)
		if err != nil {
			return 0, false, err
		}
		if i < 0 {
			candidates := r.indexes.sibling.Committing(sibling)
			if found && holder.Table == tableName {
				key, err := ParseKey(holder.Key)
				if err != nil {
					return 0, false, err
				}
				candidates = append(candidates, key)
			}
			return channelAt(ctx, snap, sibling, candidates)
		}
		if !found {
			return 0, false, nil
		}
		switch holder.Table {
		case tableName:
			return 0, false, nil
		case string(resource.TypeGroup):
			id := resource.ID{Type: resource.TypeGroup, Key: holder.Key}
			if parent, err = group.KeyFromResourceID(id); err != nil {
				return 0, false, err
			}
			if r.folderAllowed != nil && !r.folderAllowed(id) {
				return 0, false, nil
			}
		default:
			return 0, false, errors.Newf(
				"%s %s holds a name among channels",
				holder.Table,
				holder.Key,
			)
		}
		path = rest
	}
}

// channelAt returns the first of keys whose channel, as stored in snap, sits at s. The
// read sees a commit in flight, which the index shows only once the commit flushes.
func channelAt(
	ctx context.Context,
	snap gorp.Snapshot,
	s group.Sibling,
	keys []Key,
) (Key, bool, error) {
	for _, k := range keys {
		var ch Channel
		err := gorp.NewRetrieve[Key, Channel]().
			Where(gorp.MatchKeys[Key, Channel](k)).
			Entry(&ch).
			Exec(ctx, snap)
		if errors.Is(err, query.ErrNotFound) {
			continue
		}
		if err != nil {
			return 0, false, err
		}
		var parent group.Key
		if ch.Group != nil {
			parent = *ch.Group
		}
		if parent == s.Parent && validate.NormalizeName(ch.Name) == s.Name {
			return k, true, nil
		}
	}
	return 0, false, nil
}
