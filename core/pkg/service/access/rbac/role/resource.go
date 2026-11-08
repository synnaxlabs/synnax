// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of roles.
func (s *Service) Resources() *resource.TableProvider[Key, Role] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Role]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    KeyFromResourceID,
		Name:   func(r *Role) string { return r.Name },
	})
}

// ResourceID constructs a unique resource.ID for the Role with the given key.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeRole, Key: k.String()}
}

// ResourceIDs constructs a slice of unique resource.IDs for the Roles with the given
// keys.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

// ResourceIDsFromRoles constructs a slice of unique resource.IDs for the given Roles.
func ResourceIDsFromRoles(roles []Role) []resource.ID {
	return lo.Map(roles, func(r Role, _ int) resource.ID { return r.ResourceID() })
}

// KeyFromResourceID parses the role key from the given resource.ID.
func KeyFromResourceID(id resource.ID) (Key, error) { return resource.ParseUUIDKey(id) }

// KeysFromResourceIDs converts a slice of resource IDs to a slice of keys, returning an
// error if any of the IDs are invalid.
func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return KeyFromResourceID(id)
	})
}
