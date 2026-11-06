// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of schematics.
func (s *Service) Resources() *resource.TableProvider[Key, Schematic] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Schematic]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    func(id resource.ID) (Key, error) { return uuid.Parse(id.Key) },
		Name:   func(sc *Schematic) string { return sc.Name },
	})
}
