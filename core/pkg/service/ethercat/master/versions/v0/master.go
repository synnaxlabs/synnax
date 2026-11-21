// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ gorp.Entry[Key] = Master{}

// CustomTypeName gives masters a table prefix that cannot collide with another
// integration's masters.
func (Master) CustomTypeName() string { return "ethercat_master" }

// GorpKey returns the key of the master.
func (m Master) GorpKey() Key { return m.Key }

// ResourceID returns the resource ID of the master.
func (m Master) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypeEthercatMaster, Key: m.Key.String()}
}
