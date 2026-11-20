// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package rename stages the renames that an upgrade makes to stored records, so that a
// later upgrade step can report each one and rewrite the text that names it.
package rename

import (
	"context"
	"strconv"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
)

// Rename is a new name that an upgrade gave a stored record.
type Rename struct {
	// Resource is the record that the upgrade renamed.
	Resource resource.ID `msgpack:"resource"`
	// From is the name before the upgrade.
	From string `msgpack:"from"`
	// To is the name after the upgrade.
	To string `msgpack:"to"`
}

// prefix starts the key of each staged rename.
const prefix = "sy_v63_staged_rename/"

func keyOf(id resource.ID) []byte { return []byte(prefix + id.String()) }

// Stage records renames in tx. A later rename of the same record replaces it.
func Stage(ctx context.Context, tx gorp.Tx, renames ...Rename) error {
	for _, r := range renames {
		data, err := msgpack.Codec.Encode(r)
		if err != nil {
			return err
		}
		if err = tx.Set(ctx, keyOf(r.Resource), data); err != nil {
			return err
		}
	}
	return nil
}

// Staged returns every rename that Stage recorded, in the order of their resource IDs.
func Staged(snap gorp.Snapshot) (renames []Rename, err error) {
	iter, err := snap.OpenIterator(kv.IterPrefix([]byte(prefix)))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		var r Rename
		if err = msgpack.Codec.Decode(iter.Value(), &r); err != nil {
			return nil, err
		}
		renames = append(renames, r)
	}
	return renames, nil
}

// Clear deletes every staged rename in tx.
func Clear(ctx context.Context, tx gorp.Tx) error {
	renames, err := Staged(tx)
	if err != nil {
		return err
	}
	for _, r := range renames {
		if err = tx.Delete(ctx, keyOf(r.Resource)); err != nil {
			return err
		}
	}
	return nil
}

// Free returns base when taken reports false for it, and else base with the first
// suffix (_2, _3) for which taken reports false.
func Free(base string, taken func(string) bool) string {
	if !taken(base) {
		return base
	}
	for i := 2; ; i++ {
		if candidate := base + "_" + strconv.Itoa(i); !taken(candidate) {
			return candidate
		}
	}
}
