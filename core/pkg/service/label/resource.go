// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of labels.
func (s *Service) Resources() *resource.TableProvider[Key, Label] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Label]{
		Table:  s.table,
		Search: s.search,
		ID:     ResourceID,
		Key:    resource.ParseUUIDKey,
		Name:   func(l *Label) string { return l.Name },
	})
}

// ResourceID constructs a unique resource.ID for the label with the given key.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeLabel, Key: k.String()}
}

// ResourceIDs constructs a slice of unique resource.IDs for the labels with the given
// keys.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

func newSearchIndex() *gorp.SearchIndex[Key, Label] {
	return gorp.NewSearchIndex("search", func(e *Label) []string {
		return []string{e.Name}
	})
}
