// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of ICDs.
func (s *Service) Resources() *resource.TableProvider[Key, ICD] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, ICD]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    resource.ParseUUIDKey,
		Name:   func(ic *ICD) string { return ic.Name },
	})
}

// ResourceID returns the resource ID of the ICD with the given key.
func ResourceID(key Key) resource.ID {
	return resource.ID{Type: resource.TypeICD, Key: key.String()}
}

// ResourceIDs returns the resource IDs of the ICDs with the given keys.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

// ResourceIDsFromICDs returns the resource IDs of the given ICDs.
func ResourceIDsFromICDs(ics []ICD) []resource.ID {
	return lo.Map(ics, func(ic ICD, _ int) resource.ID { return ic.ResourceID() })
}
