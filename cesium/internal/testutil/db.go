// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"context"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// WriteDomain writes data to db as one domain that spans tr. It returns an error if tr
// overlaps a domain that db already holds.
func WriteDomain(
	ctx context.Context,
	db *domain.DB,
	tr telem.TimeRange,
	data []byte,
) (err error) {
	w, err := db.OpenWriter(ctx, domain.WriterConfig{Start: tr.Start, End: tr.End})
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, w.Close()) }()
	if _, err = w.Write(data); err != nil {
		return err
	}
	return w.Commit(ctx, tr.End)
}

// ReadDomain returns the data of every domain in db that overlaps tr, in time order.
func ReadDomain(
	ctx context.Context,
	db *domain.DB,
	tr telem.TimeRange,
) (b []byte, err error) {
	i := db.OpenIterator(domain.IterRange(tr))
	defer func() { err = errors.Combine(err, i.Close()) }()
	for i.SeekFirst(ctx); i.Valid(); i.Next() {
		r, err := i.OpenReader(ctx)
		if err != nil {
			return nil, err
		}
		chunk := make([]byte, r.Size())
		if _, err = r.ReadAt(chunk, 0); err != nil {
			return nil, errors.Combine(err, r.Close())
		}
		b = append(b, chunk...)
		if err = r.Close(); err != nil {
			return nil, err
		}
	}
	return b, nil
}

// ReadUnary returns a frame that holds the data of db that overlaps tr.
func ReadUnary(
	ctx context.Context,
	db *unary.DB,
	tr telem.TimeRange,
) (frame channel.Frame, err error) {
	iter, err := db.OpenIterator(unary.IterRange(tr))
	if err != nil {
		return frame, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	if !iter.SeekFirst(ctx) {
		return frame, nil
	}
	for iter.Next(ctx, telem.TimeSpanMax) {
		frame = frame.Extend(iter.Value())
	}
	return frame, nil
}
