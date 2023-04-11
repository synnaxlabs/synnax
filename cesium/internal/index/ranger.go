// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index

import (
	"context"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/ranger"
	"github.com/synnaxlabs/x/telem"
	"io"
)

type Ranger struct {
	alamos.Instrumentation
	DB *ranger.DB
}

var _ Index = (*Ranger)(nil)

// Distance implements Index.
func (i *Ranger) Distance(ctx context.Context, tr telem.TimeRange, continuous bool) (approx DistanceApproximation, err error) {
	var startApprox, endApprox DistanceApproximation
	ctx, span := i.T.Trace(ctx, "distance", alamos.DebugLevel)
	defer func() { span.EndWith(err, ErrDiscontinuous) }()

	iter := i.DB.NewIterator(ranger.IteratorConfig{Bounds: tr})

	if !iter.SeekFirst(ctx) || (!iter.Range().ContainsRange(tr) && continuous) {
		err = ErrDiscontinuous
		return
	}

	if tr.IsZero() {
		return
	}

	r, err := iter.NewReader(ctx)
	if err != nil {
		return
	}

	startApprox, err = i.search(tr.Start, r)
	if err != nil {
		return
	}

	if iter.Range().ContainsStamp(tr.End) || tr.End == iter.Range().End {
		endApprox, err = i.search(tr.End, r)
		approx = Between(
			endApprox.Lower-startApprox.Upper,
			endApprox.Upper-startApprox.Lower,
		)
		return
	} else if continuous {
		err = ErrDiscontinuous
		return
	}

	l := r.Len() / 8
	startToFirstEnd := Between(l-startApprox.Upper, l-startApprox.Lower)
	var gap int64 = 0

	for {
		if !iter.Next() {
			if continuous {
				err = ErrDiscontinuous
				return
			}
			approx = Between(
				startToFirstEnd.Lower+(iter.Len()/8)+gap,
				startToFirstEnd.Lower+(iter.Len()/8)+gap,
			)
			return
		}
		if iter.Range().ContainsStamp(tr.End) {
			r, err = iter.NewReader(ctx)
			if err != nil {
				return
			}
			endApprox, err = i.search(tr.End, r)
			if err != nil {
				return
			}
			approx = Between(
				startToFirstEnd.Lower+gap+endApprox.Lower,
				startToFirstEnd.Upper+gap+endApprox.Upper,
			)
			return
		}
		gap += iter.Len()
	}
}

// Stamp implements Index.
func (i *Ranger) Stamp(
	ctx context.Context,
	ref telem.TimeStamp,
	offset int64,
	continuous bool,
) (approx TimeStampApproximation, err error) {
	ctx, span := i.T.Trace(ctx, "stamp", alamos.DebugLevel)
	defer func() { span.EndWith(err, ErrDiscontinuous) }()

	iter := i.DB.NewIterator(ranger.IterRange(ref.SpanRange(telem.TimeSpanMax)))

	if !iter.SeekFirst(ctx) ||
		!iter.Range().ContainsStamp(ref) ||
		(offset >= iter.Len()/8 && continuous) {
		err = ErrDiscontinuous
		return
	}

	if offset == 0 {
		approx = Exactly(ref)
		return
	}

	r, err := iter.NewReader(ctx)
	if err != nil {
		return
	}
	startApprox, err := i.search(ref, r)
	if err != nil {
		return
	}

	endOffset := startApprox.Upper + offset
	gap := iter.Len() / 8
	if endOffset >= iter.Len()/8 {
		for {
			if !iter.Next() {
				if continuous {
					err = ErrDiscontinuous
					return
				}
				approx = Between(iter.Range().End, telem.TimeStampMax)
				return
			}
			gap += iter.Len() / 8
			if endOffset < gap {
				r, err = iter.NewReader(ctx)
				if err != nil {
					return
				}
				endOffset -= gap - iter.Len()/8
				break
			}
		}

	}

	lowerTs, err := readStamp(r, (endOffset-(startApprox.Upper-startApprox.Lower))*8, make([]byte, 8))
	if err != nil {
		return
	}
	upperTs, err := readStamp(r, (endOffset)*8, make([]byte, 8))
	if err != nil {
		return
	}
	return Between(lowerTs, upperTs), nil
}

func (i *Ranger) search(ts telem.TimeStamp, r *ranger.Reader) (DistanceApproximation, error) {
	var (
		start int64 = 0
		end         = (r.Len() / 8) - 1
		buf         = make([]byte, 8)
	)
	for start <= end {
		mid := (start + end) / 2
		midTs, err := readStamp(r, mid*8, buf)
		if err != nil {
			return Exactly[int64](0), err
		}
		if midTs == ts {
			return Exactly(mid), nil
		} else if midTs < ts {
			start = mid + 1
		} else {
			end = mid - 1
		}
	}
	return Between(end, end+1), nil
}

func readStamp(r io.ReaderAt, offset int64, buf []byte) (telem.TimeStamp, error) {
	_, err := r.ReadAt(buf, offset)
	return telem.UnmarshalF[telem.TimeStamp](telem.TimeStampT)(buf), err
}
