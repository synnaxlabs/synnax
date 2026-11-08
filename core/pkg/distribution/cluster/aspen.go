// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster

import (
	"context"
	"slices"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
)

// Keys of the digest and the version that Aspen stored for each write.
var (
	aspenDigestPrefix = []byte("--dig/")
	aspenVersionKey   = []byte("ver")
)

// deleteAspenKeys deletes the digests and the version counter that Aspen stored. It
// writes nothing when none remain.
func deleteAspenKeys(ctx context.Context, db kv.DB) error {
	iter, err := db.OpenIterator(kv.IterPrefix(aspenDigestPrefix))
	if err != nil {
		return err
	}
	var keys [][]byte
	for iter.First(); iter.Valid(); iter.Next() {
		keys = append(keys, slices.Clone(iter.Key()))
	}
	if err = iter.Close(); err != nil {
		return err
	}
	_, closer, err := db.Get(ctx, aspenVersionKey)
	if errors.Is(err, query.ErrNotFound) {
		if len(keys) == 0 {
			return nil
		}
	} else if err != nil {
		return err
	} else if err = closer.Close(); err != nil {
		return err
	}
	tx := db.OpenTx()
	for _, k := range append(keys, aspenVersionKey) {
		if err = tx.Delete(ctx, k); err != nil {
			return errors.Combine(err, tx.Close())
		}
	}
	return errors.Combine(tx.Commit(ctx), tx.Close())
}
