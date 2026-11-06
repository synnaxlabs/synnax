// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view

import (
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of views.
func (s *Service) Resources() *resource.TableProvider[Key, View] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, View]{
		Table:  s.table,
		Search: s.search,
		ID:     ResourceID,
		Key:    resource.ParseUUIDKey,
		Name:   func(v *View) string { return v.Base().Name },
	})
}
