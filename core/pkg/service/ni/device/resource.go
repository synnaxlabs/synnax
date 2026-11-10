// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Resources returns the resource.Provider of NI devices.
func (s *Service) Resources() *resource.TableProvider[Key, Device] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Device]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    KeyFromResourceID,
		Name:   func(d *Device) string { return d.Name },
	})
}

// ResourceID returns the resource ID of the device under key.
func ResourceID(key Key) resource.ID {
	return resource.ID{Type: resource.TypeNiDevice, Key: key.String()}
}

// KeyFromResourceID parses the key of a device resource ID. A malformed key is a
// validation error.
func KeyFromResourceID(id resource.ID) (Key, error) {
	k, err := uuid.Parse(id.Key)
	if err != nil {
		return Key{}, errors.Wrapf(
			validate.ErrValidation,
			"invalid device key %q",
			id.Key,
		)
	}
	return k, nil
}
