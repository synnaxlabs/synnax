// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package storage

import (
	"github.com/cockroachdb/pebble/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
)

// Keys of the digest and the version that Aspen stored for each write.
const (
	aspenDigestPrefix = "--dig/"
	aspenVersionKey   = "ver"
)

// deleteAspenKeys deletes the digests and the version counter that Aspen stored. It
// writes nothing when none remain.
func deleteAspenKeys(db *pebble.DB) error {
	digests := kv.IterPrefix([]byte(aspenDigestPrefix))
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: digests.LowerBound,
		UpperBound: digests.UpperBound,
	})
	if err != nil {
		return err
	}
	hasDigests := iter.First()
	if err = iter.Close(); err != nil {
		return err
	}
	_, closer, err := db.Get([]byte(aspenVersionKey))
	hasVersion := err == nil
	if hasVersion {
		if err = closer.Close(); err != nil {
			return err
		}
	} else if !errors.Is(err, pebble.ErrNotFound) {
		return err
	}
	if !hasDigests && !hasVersion {
		return nil
	}
	b := db.NewBatch()
	if err = b.DeleteRange(digests.LowerBound, digests.UpperBound, nil); err != nil {
		return errors.Combine(err, b.Close())
	}
	if err = b.Delete([]byte(aspenVersionKey), nil); err != nil {
		return errors.Combine(err, b.Close())
	}
	return errors.Combine(b.Commit(pebble.Sync), b.Close())
}
