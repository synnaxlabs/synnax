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
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of channels. Searches and lists leave out
// internal channels.
func (s *Service) Resources() *resource.TableProvider[Key, Channel] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Channel]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    func(id resource.ID) (Key, error) { return ParseKey(id.Key) },
		Name:   func(c *Channel) string { return c.Name },
		Hidden: func(c *Channel) bool { return c.Internal },
	})
}

// ResourceID returns the resource ID of the channel with key k.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeChannel, Key: k.String()}
}

// ResourceIDsFromKeys returns the resource.ID for each key.
func ResourceIDsFromKeys(keys Keys) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

// KeysFromResourceIDs returns a slice of Keys from a slice of resource.IDs.
func KeysFromResourceIDs(ids []resource.ID) (Keys, error) {
	return lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return ParseKey(id.Key)
	})
}

// ResourceIDsFromChannels returns the resource.ID for each channel.
func ResourceIDsFromChannels(chs []Channel) []resource.ID {
	return lo.Map(chs, func(item Channel, _ int) resource.ID {
		return ResourceID(item.Key())
	})
}
