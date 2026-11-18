// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
)

// MatchRacks returns a filter for the file certificates on any of the racks with keys.
func MatchRacks(keys ...rack.Key) Filter {
	return func(r Retrieve) gorp.Filter[Key, Certificate] {
		return r.indexes.rack.Filter(keys...)
	}
}
