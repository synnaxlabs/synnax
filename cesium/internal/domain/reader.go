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
	"io"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// reader is a readable domain of telemetry within the DB implementing the io.ReaderAt
// and io.Closer interfaces.
type reader struct {
	// section limits reads to the domain's bytes in the file.
	section io.SectionReader
	// internal is the file handle, returned to the DB on Close.
	internal *controlledReader
	// ptr locates the domain in its file.
	ptr pointer
}

// newReader opens a reader on ptr, which the index held when the GC rewrite count of
// its file was rewrites. It follows ptr to its new offset if GC moved it, and returns
// errChanged if a delete also changed it. With no rewrite, the bytes of ptr are in
// place even if a delete removed them.
func (db *DB) newReader(
	ctx context.Context,
	ptr pointer,
	rewrites uint64,
) (*reader, error) {
	internal, err := db.fc.acquireReader(ctx, ptr.fileKey)
	if err != nil {
		return nil, err
	}
	// The handle keeps GC off the file from here on.
	if current, count, found := db.idx.locate(ptr); count != rewrites {
		if !found {
			return nil, errors.Combine(errChanged, internal.Close())
		}
		ptr.offset = current.offset
	}
	r := &reader{internal: internal, ptr: ptr}
	r.section = *io.NewSectionReader(internal, int64(ptr.offset), int64(ptr.size))
	return r, nil
}

// ReadAt reads len(p) bytes of the domain starting at offset off.
func (r *reader) ReadAt(p []byte, off int64) (int, error) {
	return r.section.ReadAt(p, off)
}

// Close releases the reader's file handle back to the DB.
func (r *reader) Close() error { return r.internal.Close() }

// Size returns the number of bytes in the entire domain.
func (r *reader) Size() telem.Size { return telem.Size(r.ptr.size) }
