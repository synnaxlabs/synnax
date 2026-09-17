// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

// Version is the format of the files stored in a channel. A channel opened at a lower
// version is migrated up and its metadata rewritten.
type Version = uint8

const (
	Version1 Version = 1
	Version2 Version = 2
	// Version3 renames the metadata's is_index member, which earlier versions stored
	// under the Go field name. Opening at this version rewrites the file, so the
	// stored form catches up with the tag.
	Version3       Version = 3
	VersionCurrent         = Version3
)
