// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of Arcs.
func (s *Service) Resources() *resource.TableProvider[Key, Arc] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Arc]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    resource.ParseUUIDKey,
		Name:   func(a *Arc) string { return a.Base().Name },
	})
}

// DirectoryResources returns the resource.Provider of Arc directories.
func (s *Service) DirectoryResources() *resource.TableProvider[
	directory.Key,
	directory.Directory,
] {
	return s.directories.Resources()
}

// ResourceID returns the resource ID of the Arc with key.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeArc, Key: k.String()}
}

// ResourceIDs returns the resource IDs of the Arcs with keys.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(key Key, _ int) resource.ID { return ResourceID(key) })
}

// KeysFromResourceIDs extracts the keys of the arcs from the resource IDs.
func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return resource.ParseUUIDKey(id)
	})
}

// ResourceIDsFromArcs returns the resource IDs of the arcs.
func ResourceIDsFromArcs(arcs []Arc) []resource.ID {
	return lo.Map(arcs, func(a Arc, _ int) resource.ID { return a.ResourceID() })
}
