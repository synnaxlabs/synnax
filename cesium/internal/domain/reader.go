// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/telem"
)

// Reader is a readable domain of telemetry within the DB implementing the io.ReaderAt
// and io.Closer interfaces.
type Reader struct {
	xio.ReaderAtCloser
	ptr pointer
}

func (db *DB) newReader(ctx context.Context, ptr pointer) (*Reader, error) {
	internal, err := db.fc.acquireReader(ctx, ptr.fileKey)
	if err != nil {
		return nil, err
	}
	reader := xio.NewSectionReaderAtCloser(internal, int64(ptr.offset), int64(ptr.size))
	return &Reader{ptr: ptr, ReaderAtCloser: reader}, nil
}

// Size returns the number of bytes in the entire domain.
func (r *Reader) Size() telem.Size { return telem.Size(r.ptr.size) }
