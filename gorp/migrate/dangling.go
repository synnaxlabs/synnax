// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
)

// DropDangling returns a migration identified by key that drops from every entry of
// type E each target of refs that does not exist, through gorp.DropDangling. It
// decodes entries as E, so it runs after the migrations that convert them to E.
func DropDangling[K gorp.Key, E gorp.Entry[K]](
	key string,
	db *gorp.DB,
	refs []gorp.Reference[K, E],
) gorp.Migration {
	return New(
		key,
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var entries []E
			if err := gorp.NewRetrieve[K, E]().Entries(&entries).
				Exec(ctx, tx); err != nil {
				return err
			}
			changed := make([]E, 0, len(entries))
			for i := range entries {
				dropped, err := gorp.DropDangling(ctx, tx, db, refs, &entries[i])
				if err != nil {
					return err
				}
				if dropped {
					changed = append(changed, entries[i])
				}
			}
			return gorp.NewCreate[K, E]().Entries(&changed).Exec(ctx, tx)
		},
	)
}
