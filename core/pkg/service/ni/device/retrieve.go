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
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ni/chassis/versions"
)

// MatchChassis returns a filter for devices held by any of the given chassis.
func MatchChassis(keys ...versions.Key) Filter {
	return func(Retrieve) gorp.Filter[Key, Device] {
		return gorp.Match(func(_ gorp.Context, d *Device) (bool, error) {
			return d.Chassis != nil && slices.Contains(keys, *d.Chassis), nil
		})
	}
}
