// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alias

import "github.com/synnaxlabs/synnax/pkg/service/resource"

// Resources returns the resource.Provider of range aliases.
func (s *Service) Resources() *resource.TableProvider[string, Alias] {
	return resource.NewTableProvider(resource.TableProviderConfig[string, Alias]{
		Table:  s.table,
		Search: s.search,
		ID:     resourceID,
		Key: func(id resource.ID) (string, error) {
			_, _, err := parseGorpKey(id.Key)
			return id.Key, err
		},
		Name: func(a *Alias) string { return a.Alias },
	})
}

// resourceID returns the resource ID of the alias with the given gorp key.
func resourceID(key string) resource.ID {
	return resource.ID{Type: resource.TypeRangeAlias, Key: key}
}
