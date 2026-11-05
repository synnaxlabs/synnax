// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of ranges.
func (s *Service) Resources() *resource.TableProvider[Key, Range] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Range]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    KeyFromResourceID,
		Name:   func(r *Range) string { return r.Name },
	})
}

// ResourceID returns the resource ID of the range with key.
func ResourceID(key Key) resource.ID {
	return resource.ID{Type: resource.TypeRange, Key: key.String()}
}

// ResourceIDs converts a slice of keys to a slice of resource IDs.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

func KeyFromResourceID(id resource.ID) (Key, error) { return resource.ParseUUIDKey(id) }

// KeysFromResourceIDs converts a slice of resource IDs to a slice of keys, returning an
// error if any of the IDs are invalid.
func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return KeyFromResourceID(id)
	})
}
