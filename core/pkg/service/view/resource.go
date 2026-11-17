// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of views.
func (s *Service) Resources() *resource.TableProvider[Key, View] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, View]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    resource.ParseUUIDKey,
		Name:   func(v *View) string { return v.Base().Name },
	})
}

// ResourceID returns the resource ID of the view with the given key.
func ResourceID(keys Key) resource.ID {
	return resource.ID{Type: resource.TypeView, Key: keys.String()}
}

// KeysFromResourceIDs returns the keys of the views for the given resource IDs.
func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return resource.ParseUUIDKey(id)
	})
}
