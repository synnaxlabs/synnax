// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/gorp"
)

// WhereRacks returns a filter for tasks whose rack key matches any of the provided keys.
func WhereRacks(keys ...rack.Key) gorp.Filter[Key, Task] {
	return gorp.Match(func(_ gorp.Context, t *Task) (bool, error) {
		return lo.Contains(keys, t.Rack()), nil
	})
}
