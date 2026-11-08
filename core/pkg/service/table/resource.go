// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of tables.
func (s *Service) Resources() *resource.TableProvider[Key, Table] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Table]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    resource.ParseUUIDKey,
		Name:   func(t *Table) string { return t.Name },
	})
}

// ResourceID returns the resource ID of the table with key.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeTable, Key: k.String()}
}

// ResourceIDs returns the resource IDs of the tables with keys.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(key Key, _ int) resource.ID { return ResourceID(key) })
}

// ResourceIDsFromTables returns the resource IDs of the tables.
func ResourceIDsFromTables(tables []Table) []resource.ID {
	return lo.Map(tables, func(t Table, _ int) resource.ID { return t.ResourceID() })
}
