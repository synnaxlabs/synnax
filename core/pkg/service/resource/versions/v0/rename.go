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
	"bytes"
	"context"
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
)

// renamePrefix starts the key of each staged rename. The rest of the key is the
// resource ID, and the value is the name before the first rename.
const renamePrefix = "sy_v63_staged_rename/"

// Rename is a stored resource that the upgrade renamed.
type Rename struct {
	// ID is the renamed resource.
	ID ID
	// From is the name that the resource held before the upgrade.
	From string
}

func renameKey(id ID) []byte { return []byte(renamePrefix + id.String()) }

// StageRename records that the upgrade renamed the resource id from the name from. A
// later rename of id keeps the first from.
func StageRename(ctx context.Context, tx kv.ReadWriter, id ID, from string) error {
	key := renameKey(id)
	_, closer, err := tx.Get(key)
	if err == nil {
		return closer.Close()
	}
	if !errors.Is(err, query.ErrNotFound) {
		return err
	}
	return tx.Set(ctx, key, []byte(from))
}

// StagedRenames returns each rename that StageRename recorded, in the order of their
// resource IDs.
func StagedRenames(r kv.Reader) (renames []Rename, err error) {
	iter, err := r.OpenIterator(kv.IterPrefix([]byte(renamePrefix)))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		id := string(bytes.TrimPrefix(iter.Key(), []byte(renamePrefix)))
		typ, key, ok := strings.Cut(id, ":")
		if !ok {
			return nil, errors.Newf("staged rename has no resource ID: %q", id)
		}
		renames = append(renames, Rename{
			ID:   ID{Type: Type(typ), Key: key},
			From: string(iter.Value()),
		})
	}
	return renames, iter.Error()
}

// ClearRenames deletes every staged rename.
func ClearRenames(ctx context.Context, tx kv.ReadWriter) error {
	renames, err := StagedRenames(tx)
	if err != nil {
		return err
	}
	for _, r := range renames {
		if err = tx.Delete(ctx, renameKey(r.ID)); err != nil {
			return err
		}
	}
	return nil
}

// FreeName returns base when taken reports false for it, and else base with the first
// suffix (_2, _3) for which taken reports false.
func FreeName(base string, taken func(string) bool) string {
	if !taken(base) {
		return base
	}
	for i := 2; ; i++ {
		if name := base + "_" + strconv.Itoa(i); !taken(name) {
			return name
		}
	}
}
