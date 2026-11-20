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
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"go.uber.org/zap"
)

// Drop returns a migration identified by migrationKey that deletes the table whose
// entry type is named typeName and whose key type is K: every entry in the current and
// the pre-v0.54 key formats, and the record of the migrations that the table applied.
// It logs a warning with the key of each entry it deletes. It decodes no entry, so it
// can drop a table whose entry type no longer exists. The name of no other table's
// entry type may start with typeName.
func Drop[K gorp.Key](migrationKey, typeName string) gorp.Migration {
	return New(
		migrationKey,
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			codec := key.NewCodec[K](key.NamedPrefix(typeName))
			keys, err := deletePrefix(
				ctx,
				tx,
				codec.Prefix(),
				func(b []byte) (K, error) { return codec.Decode(b), nil },
			)
			if err != nil {
				return err
			}
			legacy, err := key.LegacyPrefix(typeName)
			if err != nil {
				return err
			}
			legacyKeys, err := deletePrefix(
				ctx,
				tx,
				legacy,
				func(b []byte) (k K, err error) {
					err = msgpack.Codec.Decode(b[len(legacy):], &k)
					return k, err
				},
			)
			if err != nil {
				return err
			}
			for _, k := range append(keys, legacyKeys...) {
				ins.L.Warn(
					"deleted an entry",
					zap.String("type", typeName),
					zap.String("key", key.Format(k)),
				)
			}
			return tx.Delete(ctx, key.Migrations(typeName))
		},
	)
}

// deletePrefix deletes every entry whose key starts with prefix, and returns the key
// that decode reads from each.
func deletePrefix[K gorp.Key](
	ctx context.Context,
	tx gorp.Tx,
	prefix []byte,
	decode func(b []byte) (K, error),
) (keys []K, err error) {
	itr, err := tx.OpenIterator(kv.IterPrefix(prefix))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, itr.Close()) }()
	for itr.First(); itr.Valid(); itr.Next() {
		var k K
		if k, err = decode(itr.Key()); err != nil {
			return nil, err
		}
		if err = tx.Delete(ctx, itr.Key()); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}
