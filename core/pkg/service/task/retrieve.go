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

func (r Retrieve) WhereRacks(key ...rack.Key) Retrieve {
	r.gorp = r.gorp.Where(func(_ gorp.Context, t *Task) (bool, error) {
		return lo.Contains(key, t.Rack()), nil
	}, gorp.Required())
	return r
}
