// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Resources returns the resource.Provider of policies.
func (s *Service) Resources() *resource.TableProvider[Key, Policy] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Policy]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    resource.ParseUUIDKey,
		Name:   func(p *Policy) string { return p.Name },
	})
}

// ResourceID constructs a unique resource.ID for the Policy with the given key.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypePolicy, Key: k.String()}
}

// ResourceIDs constructs a slice of unique resource.IDs for the Policys with the given
// keys.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

// ResourceIDsFromPolicies constructs a slice of unique resource.IDs for the given
// Policys.
func ResourceIDsFromPolicies(policies []Policy) []resource.ID {
	return lo.Map(policies, func(p Policy, _ int) resource.ID { return p.ResourceID() })
}

// KeysFromResourceIDs extracts the Policy keys from the given resource.IDs.
func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return resource.ParseUUIDKey(id)
	})
}
