// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
)

// ByKey executes r through snap and returns the ICDs that it finds, by key.
func (r Retrieve) ByKey(ctx context.Context, snap gorp.Snapshot) (map[Key]ICD, error) {
	var found []ICD
	if err := r.Entries(&found).Exec(ctx, snap); err != nil {
		return nil, err
	}
	return lo.KeyBy(found, func(ic ICD) Key { return ic.Key }), nil
}
