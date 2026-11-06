// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Resources returns the resource.Provider of racks.
func (s *Service) Resources() *resource.TableProvider[Key, Rack] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Rack]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     Key.ResourceID,
		Key:    KeyFromResourceID,
		Name:   func(r *Rack) string { return r.Name },
	})
}

// ResourceIDs returns the resource ID of the rack with each key.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(key Key, _ int) resource.ID { return key.ResourceID() })
}

// ResourceIDsFromRacks returns the resource ID of each rack.
func ResourceIDsFromRacks(racks []Rack) []resource.ID {
	return lo.Map(racks, func(r Rack, _ int) resource.ID { return r.ResourceID() })
}

// KeyFromResourceID parses the key of a rack resource ID. A malformed key is a
// validation error.
func KeyFromResourceID(id resource.ID) (Key, error) {
	k, err := strconv.ParseUint(id.Key, 10, 32)
	if err != nil {
		return 0, errors.Wrapf(validate.ErrValidation, "invalid rack key %q", id.Key)
	}
	return Key(k), nil
}

// KeysFromResourceIDs returns the key of the rack that each ID names. A malformed key
// is a validation error.
func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return KeyFromResourceID(id)
	})
}
