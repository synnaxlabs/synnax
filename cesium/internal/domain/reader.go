// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain

import (
	"context"

	xio "github.com/synnaxlabs/x/io"
)

// Reader is a readable domain of telemetry within the DB implementing the io.ReaderAt
// and io.Closer interfaces.
type Reader struct {
	ptr pointer
	xio.ReaderAtCloser
}

func (db *DB) newReader(ctx context.Context, ptr pointer) (*Reader, error) {
	internal, err := db.fc.acquireReader(ctx, ptr.fileKey)
	if err != nil {
		return nil, err
	}
	reader := xio.NewSectionReaderAtCloser(internal, int64(ptr.offset), int64(ptr.length))
	return &Reader{ptr: ptr, ReaderAtCloser: reader}, nil
}

// Len returns the number of bytes in the entire domain.
func (r *Reader) Len() int64 { return int64(r.ptr.length) }
