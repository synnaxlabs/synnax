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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Retrieve is a builder for querying channels.
type Retrieve struct {
	gorp    gorp.Retrieve[Key, Channel]
	indexes indexes
	// folders is shared by each copy of the Retrieve, so a filter that Where added
	// reads the folders that AllowFolders allows when the query runs.
	folders *folderAccess
}

// folderAccess holds the folders that a path can pass.
type folderAccess struct {
	// allowed reports whether a path can pass the folder with an ID, or is nil when a
	// path can pass any folder.
	allowed func(resource.ID) bool
}

// AllowFolders makes a path that MatchPaths matches pass only folders that allowed
// accepts, whether it comes before or after Where.
func (r Retrieve) AllowFolders(allowed func(resource.ID) bool) Retrieve {
	r.folders.allowed = allowed
	return r
}

// MatchVirtual returns a filter for channels that are virtual if virtual is true, or
// are not virtual if virtual is false. Calculated channels are excluded from the
// virtual bucket even though they are stored with Virtual=true.
func MatchVirtual(virtual bool) Filter {
	return Match(func(_ gorp.Context, _ Retrieve, ch *Channel) (bool, error) {
		isVirtual := ch.Virtual && !ch.IsCalculated()
		return isVirtual == virtual, nil
	})
}

// MatchCalculated returns a filter for channels that have a non-empty Expression field.
func MatchCalculated() Filter {
	return Match(func(_ gorp.Context, _ Retrieve, ch *Channel) (bool, error) {
		return ch.IsCalculated(), nil
	})
}
