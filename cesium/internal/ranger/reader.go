// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"github.com/synnaxlabs/x/telem"
	"io"
)

// Reader is a readable range of telemetry within the DB implementing the io.ReaderAt
// and io.Closer interfaces.
type Reader struct {
	ptr pointer
	io.ReaderAt
	io.Closer
}

// Len returns the number of bytes in the entire range.
func (r *Reader) Len() int64 { return int64(r.ptr.length) }

// Range returns the time interval occupied by the range.
func (r *Reader) Range() telem.TimeRange { return r.ptr.TimeRange }
