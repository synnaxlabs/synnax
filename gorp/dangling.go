// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"context"

	"github.com/synnaxlabs/x/errors"
)

// DropDangling removes from e each target of refs that does not exist in tx, through
// the Detach of the reference that holds it. A target whose key is not in the text
// form of its table's keys does not exist. DropDangling reports whether it removed a
// target. It returns an error when a target's table is not open in db, or when a
// missing target's reference has no Detach.
func DropDangling[K Key, E Entry[K]](
	ctx context.Context,
	tx Tx,
	db *DB,
	refs []Reference[K, E],
	e *E,
) (bool, error) {
	dropped := false
	for _, ref := range refs {
		for _, target := range ref.targets(e) {
			table, err := db.registry.table(target.Table)
			if err != nil {
				return false, err
			}
			if table.canonical(target.Key) {
				ok, err := table.exists(ctx, tx, target.Key)
				if err != nil {
					return false, err
				}
				if ok {
					continue
				}
			}
			if !ref.detachTarget(e, target) {
				return false, errors.Newf(
					"[gorp] - %s %s does not exist, and %s cannot drop it",
					target.Table,
					target.Key,
					ref.referenceName(),
				)
			}
			dropped = true
		}
	}
	return dropped, nil
}
