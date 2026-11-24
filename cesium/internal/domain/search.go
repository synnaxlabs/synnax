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
	"encoding/binary"
	"io"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// SearchStamps returns the position of stamp among the rising index stamps of size
// bytes in r: exactly the first position that holds stamp, or the positions before and
// after it. It reads one stamp at each step of a binary search.
func SearchStamps(
	r io.ReaderAt,
	size telem.Size,
	stamp telem.TimeStamp,
) (Approximation[int64], error) {
	var (
		start int64
		end   = telem.Bit64.SampleCount(size) - 1
		buf   = make([]byte, telem.Bit64)
		found bool
	)
	for start <= end {
		mid := (start + end) / 2
		n, err := r.ReadAt(buf, int64(telem.Bit64.Size(mid)))
		if err != nil {
			return Exactly[int64](0), err
		}
		if n != len(buf) {
			return Exactly[int64](0), errors.Newf(
				"read of a stamp at position %d returned %d bytes", mid, n,
			)
		}
		if midTs := telem.TimeStamp(binary.LittleEndian.Uint64(buf)); midTs < stamp {
			start = mid + 1
		} else {
			found = found || midTs == stamp
			end = mid - 1
		}
	}
	if found {
		return Exactly(start), nil
	}
	return Between(end, end+1), nil
}
