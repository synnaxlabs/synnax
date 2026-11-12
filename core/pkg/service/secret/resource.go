// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of secrets.
func (s *Service) Resources() *resource.TableProvider[Key, Secret] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Secret]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    resource.ParseUUIDKey,
		Name:   func(s *Secret) string { return s.Base().Name },
	})
}

// ResourceID returns the resource ID of the secret with the given key.
func ResourceID(key Key) resource.ID {
	return resource.ID{Type: resource.TypeSecret, Key: key.String()}
}

// ResourceIDs returns the resource IDs of the secrets with the given keys.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

// ResourceIDsFromSecrets returns the resource IDs of the given secrets.
func ResourceIDsFromSecrets(secrets []Secret) []resource.ID {
	return lo.Map(secrets, func(s Secret, _ int) resource.ID { return s.ResourceID() })
}
