// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of statuses.
func (s *Service) Resources() *resource.TableProvider[Key, Status[any]] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Status[any]]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    func(id resource.ID) (Key, error) { return id.Key, nil },
		Name:   func(st *Status[any]) string { return st.Name },
	})
}

// ResourceID returns the resource ID of the status with key.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeStatus, Key: k}
}

// ResourceIDs converts a slice of keys to a slice of resource IDs.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

// KeysFromResourceIDs converts a slice of resource IDs to a slice of keys.
func KeysFromResourceIDs(ids []resource.ID) []Key {
	return lo.Map(ids, func(id resource.ID, _ int) Key { return id.Key })
}
