// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import "github.com/synnaxlabs/synnax/pkg/service/resource"

// Resources returns the resource.Provider of projects.
func (s *Service) Resources() *resource.TableProvider[Key, Project] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Project]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    KeyFromResourceID,
		Name:   func(p *Project) string { return p.Name },
	})
}
