// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"bytes"
	"context"
	"strconv"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
)

// textRenamePrefix starts the key of each stored text rename. The rest of the key is
// the key of the renamed channel, and the value is the path that text names it by.
const textRenamePrefix = "sy_channel_text_rename/"

// TextRename is a rename of a channel whose text pass has not run.
type TextRename struct {
	// Key is the key of the renamed channel.
	Key Key
	// From is the path that text names the channel by.
	From string
}

func textRenameKey(k Key) []byte {
	return []byte(textRenamePrefix + strconv.FormatUint(uint64(k), 10))
}

// storeTextRename records that the channel with key k was renamed from the path from.
// A later rename of k keeps the first from, since text still names that path.
func (w Writer) storeTextRename(ctx context.Context, k Key, from string) error {
	key := textRenameKey(k)
	_, closer, err := w.tx.Get(key)
	if err == nil {
		return closer.Close()
	}
	if !errors.Is(err, query.ErrNotFound) {
		return err
	}
	return w.tx.Set(ctx, key, []byte(from))
}

// TextRenames returns each rename whose text pass has not run, in key order.
func (w Writer) TextRenames(context.Context) (renames []TextRename, err error) {
	iter, err := w.tx.OpenIterator(kv.IterPrefix([]byte(textRenamePrefix)))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		raw := bytes.TrimPrefix(iter.Key(), []byte(textRenamePrefix))
		k, err := strconv.ParseUint(string(raw), 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "text rename has no channel key: %q", raw)
		}
		renames = append(renames, TextRename{Key: Key(k), From: string(iter.Value())})
	}
	return renames, iter.Error()
}

// TakeTextRenames returns each rename whose text pass has not run, in key order, and
// deletes them in the transaction of w.
func (w Writer) TakeTextRenames(ctx context.Context) ([]TextRename, error) {
	renames, err := w.TextRenames(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range renames {
		if err = w.tx.Delete(ctx, textRenameKey(r.Key)); err != nil {
			return nil, err
		}
	}
	return renames, nil
}
