// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of users.
func (s *Service) Resources() *resource.TableProvider[Key, User] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, User]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    KeyFromResourceID,
		Name:   func(u *User) string { return u.Name },
	})
}

// ResourceID returns the resource ID of the user with key.
func ResourceID(key Key) resource.ID {
	return resource.ID{Type: resource.TypeUser, Key: key.String()}
}

// ResourceIDsFromKeys returns a slice of unique identifiers from a slice of keys
func ResourceIDsFromKeys(keys []Key) []resource.ID {
	return lo.Map(keys, func(key Key, _ int) resource.ID { return ResourceID(key) })
}

// ResourceIDsFromUsers returns the resource IDs of users.
func ResourceIDsFromUsers(users []User) []resource.ID {
	return lo.Map(users, func(u User, _ int) resource.ID { return u.ResourceID() })
}

func KeyFromResourceID(id resource.ID) (Key, error) { return resource.ParseUUIDKey(id) }
