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
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// pathSeparator joins the segments of a path.
const pathSeparator = "."

// MatchPaths returns a filter for the channels at paths. A path is the names of the
// folders from the top, then the name of the channel, joined by '.'. A path that no
// channel is at matches nothing. Each segment costs one probe of the names in a folder,
// and the filter reads no record to find the keys.
func MatchPaths(paths ...string) Filter {
	return func(r Retrieve) gorp.Filter[Key, Channel] {
		return gorp.MatchKeysFunc[Key, Channel](func(ctx gorp.Context) ([]Key, error) {
			keys := make([]Key, 0, len(paths))
			for _, p := range paths {
				key, found, err := r.indexes.keyAt(ctx, ctx.Snapshot, p)
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
// channel is at it.
func (i indexes) keyAt(ctx context.Context, snap gorp.Snapshot, path string) (
	Key,
	bool,
	error,
) {
	var parent group.Key
	for {
		segment, rest, inFolder := strings.Cut(path, pathSeparator)
		holder, found, err := i.sibling.Holder(
			ctx,
			snap,
			group.Sibling{Parent: parent, Name: segment},
		)
		if err != nil || !found {
			return 0, false, err
		}
		if !inFolder {
			if holder.Table != tableName {
				return 0, false, nil
			}
			key, err := ParseKey(holder.Key)
			return key, err == nil, err
		}
		if holder.Table != string(resource.TypeGroup) {
			return 0, false, nil
		}
		if parent, err = uuid.Parse(holder.Key); err != nil {
			return 0, false, err
		}
		path = rest
	}
}
