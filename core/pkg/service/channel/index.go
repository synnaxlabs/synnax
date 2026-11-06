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

// indexReference returns the reference from a data channel to its index channel. A
// delete of an index that a data channel uses fails.
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

// MatchIndexes returns a filter for the data channels whose index is any of indexes. An
// index channel does not match its own key.
func (s *Service) MatchIndexes(indexes ...Key) Filter {
	return func(Retrieve) gorp.Filter[Key, Channel] { return s.index.Filter(indexes...) }
}
