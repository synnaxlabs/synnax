// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import "github.com/synnaxlabs/gorp"

// nameSpace is the space of the index that keeps channel names unique across the Core.
const nameSpace = "channel.name"

// newNameIndex returns the index that keeps channel names unique across the Core. A
// write that keeps the name its channel holds passes, so channels with one name from
// before the index stay writable.
func newNameIndex() *gorp.UniqueIndex[Key, Channel, string] {
	return gorp.NewUniqueIndex(
		"unique_name",
		nameSpace,
		func(c Channel) (string, bool) { return c.Name, true },
	)
}
