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

// indexReference returns the config of the reference from a data channel to the index
// channel that holds its timestamps. A delete of an index fails while it keeps a data
// channel that uses the index. Opening the service clears a missing index, and a
// calculated channel whose index was deleted gets a new one on its next write.
func indexReference() *gorp.KeyReference[Key, Channel, Key] {
	return gorp.NewKeyReference(gorp.KeyReferenceConfig[Key, Channel, Key]{
		Name:     "local_index",
		Table:    tableName,
		OnDelete: gorp.Restrict,
		Targets: func(c *Channel) []Key {
			if c.IsIndex || c.LocalIndex == 0 {
				return nil
			}
			return []Key{c.Index()}
		},
		Detach: func(c *Channel, _ Key) { c.LocalIndex = 0 },
	})
}
