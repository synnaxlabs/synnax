// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of schematic symbols.
func (s *Service) Resources() *resource.TableProvider[Key, Symbol] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Symbol]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    resource.ParseUUIDKey,
		Name:   func(sym *Symbol) string { return sym.Name },
	})
}

// ResourceID returns the resource ID of the symbol with key.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeSchematicSymbol, Key: k.String()}
}

// ResourceIDs returns the resource IDs of the symbols with keys.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(key Key, _ int) resource.ID { return ResourceID(key) })
}

// KeysFromResourceIDs extracts the keys of the symbols from the resource IDs.
func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return resource.ParseUUIDKey(id)
	})
}

// ResourceIDsFromSymbols returns the resource IDs of the symbols.
func ResourceIDsFromSymbols(symbols []Symbol) []resource.ID {
	return lo.Map(symbols, func(s Symbol, _ int) resource.ID { return s.ResourceID() })
}
