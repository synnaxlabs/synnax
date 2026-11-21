// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package master

import (
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Resources returns the resource.Provider of EtherCAT masters.
func (s *Service) Resources() *resource.TableProvider[Key, Master] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Master]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    keyFromResourceID,
		Name:   func(m *Master) string { return m.Name },
	})
}

// ResourceID returns the resource ID of the master under key.
func ResourceID(key Key) resource.ID {
	return resource.ID{Type: resource.TypeEthercatMaster, Key: key.String()}
}

func keyFromResourceID(id resource.ID) (Key, error) {
	k, err := uuid.Parse(id.Key)
	if err != nil {
		return Key{}, errors.Wrapf(
			validate.ErrValidation,
			"invalid master key %q",
			id.Key,
		)
	}
	return k, nil
}
