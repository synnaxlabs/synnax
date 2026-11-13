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
	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
)

// Drop returns a migration identified by migrationKey that deletes the table whose
// entry type is named typeName: every entry in the current and the pre-v0.54 key
// formats, and the record of the migrations that the table applied. It decodes no
// entry, so it can drop a table whose entry type no longer exists. The name of no other
// table's entry type may start with typeName.
func Drop(migrationKey, typeName string) gorp.Migration {
	return New(
		migrationKey,
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			legacy, err := key.LegacyPrefix(ctx, typeName)
			if err != nil {
				return err
			}
			for _, prefix := range [][]byte{key.NamedPrefix(typeName), legacy} {
				if err = deletePrefix(ctx, tx, prefix); err != nil {
					return err
				}
			}
			return tx.Delete(ctx, key.Migrations(typeName))
		},
	)
}

func deletePrefix(ctx context.Context, tx gorp.Tx, prefix []byte) (err error) {
	itr, err := tx.OpenIterator(kv.IterPrefix(prefix))
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, itr.Close()) }()
	for itr.First(); itr.Valid(); itr.Next() {
		if err = tx.Delete(ctx, itr.Key()); err != nil {
			return err
		}
	}
	return nil
}
