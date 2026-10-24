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

// normalizeKeysKey identifies every key-normalization migration in the applied set,
// which OpenTable scopes per entry type. Released stores already record it.
const normalizeKeysKey = "normalize_keys"

// NormalizeKeys returns a migration that moves entries stored under the pre-v0.54 key
// prefix msgpack(legacyTypeName) to the current prefix and key encoding, decoding each
// row as E to recover its key. E is the frozen shape that wrote those rows and
// legacyTypeName the name the writing release reported for it; state both, never
// derive them from the head type. Tables first stored by v0.54 or later hold no such
// rows and must not run it.
func NormalizeKeys[K gorp.Key, E gorp.Entry[K]](legacyTypeName string) gorp.Migration {
	return New(
		normalizeKeysKey,
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) (err error) {
			codec := key.NewCodec[K](key.Prefix[E]())
			oldPrefix, err := key.LegacyPrefix(ctx, legacyTypeName)
			if err != nil {
				return err
			}
			if string(oldPrefix) == string(codec.Prefix()) {
				return nil
			}
			itr, err := tx.OpenIterator(kv.IterPrefix(oldPrefix))
			if err != nil {
				return err
			}
			defer func() { err = errors.Combine(err, itr.Close()) }()
			for itr.First(); itr.Valid(); itr.Next() {
				rawValue := itr.Value()
				var entry E
				if err = tx.Decode(ctx, rawValue, &entry); err != nil {
					return errors.Wrapf(
						err,
						"%s: failed to decode entry at old prefix key %x",
						normalizeKeysKey,
						itr.Key(),
					)
				}
				if err = tx.Delete(ctx, itr.Key()); err != nil {
					return err
				}
				if err = tx.Set(
					ctx,
					codec.Encode(entry.GorpKey()),
					rawValue,
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
