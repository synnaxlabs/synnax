// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group

import (
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Resources returns the resource.Provider of groups.
func (s *Service) Resources() *resource.TableProvider[Key, Group] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Group]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    KeyFromResourceID,
		Name:   func(g *Group) string { return g.Name },
	})
}

func ResourceID(key Key) resource.ID {
	return resource.ID{Type: resource.TypeGroup, Key: key.String()}
}

func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

func ResourceIDsFromGroups(groups []Group) []resource.ID {
	return lo.Map(groups, func(g Group, _ int) resource.ID { return g.ResourceID() })
}

// KeysFromGroups returns the key of each group.
func KeysFromGroups(groups []Group) []Key {
	return lo.Map(groups, func(g Group, _ int) Key { return g.Key })
}

// KeyFromResourceID converts id to the Key of the group it identifies. It returns an
// error wrapping validate.ErrValidation if id is not a group or its key is not a valid
// UUID. Callers that resolve a caller-supplied field scope the returned error to that
// field with validate.PathedError.
func KeyFromResourceID(id resource.ID) (Key, error) {
	if id.Type != resource.TypeGroup {
		return uuid.Nil(), errors.Wrapf(
			validate.ErrValidation, "must be a group, got %q", id.Type,
		)
	}
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return uuid.Nil(), errors.Wrapf(
			validate.ErrValidation, "invalid group key %q: %v", id.Key, err,
		)
	}
	return key, nil
}
