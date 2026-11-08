// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of panels.
func (s *Service) Resources() *resource.TableProvider[Key, Panel] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Panel]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    resource.ParseUUIDKey,
		Name:   func(p *Panel) string { return p.Name },
	})
}

// ResourceID returns the resource ID of the panel with key k.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypePanel, Key: k.String()}
}

// ResourceIDs returns the resource ID of the panel with each key.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

// ResourceIDsFromPanels returns the resource ID of each panel.
func ResourceIDsFromPanels(panels []Panel) []resource.ID {
	return lo.Map(panels, func(p Panel, _ int) resource.ID { return p.ResourceID() })
}

// KeysFromResourceIDs returns the key of the panel that each ID names. A malformed key
// is a validation error.
func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	keys := make([]Key, len(ids))
	var err error
	for i, id := range ids {
		if keys[i], err = resource.ParseUUIDKey(id); err != nil {
			return nil, err
		}
	}
	return keys, nil
}
