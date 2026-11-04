// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import "github.com/synnaxlabs/gorp"

// MatchNested returns a filter for devices held by another device when nested is true,
// and for devices that sit directly on their rack when nested is false.
func MatchNested(nested bool) Filter {
	return Match(func(_ gorp.Context, _ Retrieve, e *Device) (bool, error) {
		return (e.Parent != nil) == nested, nil
	})
}
