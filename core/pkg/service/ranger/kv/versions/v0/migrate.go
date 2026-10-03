// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
)

// Migration re-encodes stored entries from MessagePack to Orc.
var Migration = gorp.CodecMigration[string, Pair]("msgpack_to_orc")

// NormalizeKeys re-keys Pair rows that v0.52 and v0.53 stored under the pre-v0.54 key
// format with the type name "Pair".
var NormalizeKeys = gorp.NormalizeKeysMigration[string, Pair]("Pair")

// RecoverKVPairKeys re-keys Pair rows that v0.51 and earlier stored under the pre-v0.54
// key format with the type name "KVPair". A pair whose key already holds a value keeps
// that newer value.
var RecoverKVPairKeys = gorp.NewMigration(
	"v59_recover_kvpair_keys",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) (err error) {
		prefix, err := msgpack.Codec.Encode(ctx, "KVPair")
		if err != nil {
			return err
		}
		itr, err := tx.OpenIterator(kv.IterPrefix(prefix))
		if err != nil {
			return err
		}
		defer func() { err = errors.Combine(err, itr.Close()) }()
		reader := gorp.WrapReader[string, Pair](tx)
		writer := gorp.WrapWriter[string, Pair](tx)
		for itr.First(); itr.Valid(); itr.Next() {
			var p Pair
			if err = tx.Decode(ctx, itr.Value(), &p); err != nil {
				return errors.Wrapf(err, "failed to decode KVPair row %x", itr.Key())
			}
			_, err = reader.Get(ctx, p.GorpKey())
			if errors.Is(err, query.ErrNotFound) {
				err = writer.Set(ctx, p)
			}
			if err != nil {
				return err
			}
			if err = tx.Delete(ctx, itr.Key()); err != nil {
				return err
			}
		}
		return nil
	},
)
