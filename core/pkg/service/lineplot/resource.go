// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of line plots.
func (s *Service) Resources() *resource.TableProvider[Key, LinePlot] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, LinePlot]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    resource.ParseUUIDKey,
		Name:   func(l *LinePlot) string { return l.Name },
	})
}

// ResourceID returns the resource ID of the line plot with key.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeLineplot, Key: k.String()}
}

// ResourceIDs returns the resource IDs of the line plots with keys.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

// ResourceIDsFromLinePlots returns the resource IDs of the schematics.
func ResourceIDsFromLinePlots(linePlots []LinePlot) []resource.ID {
	return lo.Map(linePlots, func(linePlot LinePlot, _ int) resource.ID {
		return linePlot.ResourceID()
	})
}
