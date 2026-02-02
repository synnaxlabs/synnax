// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb

import (
	"context"

	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	xunsafe "github.com/synnaxlabs/x/unsafe"
)

// FrameToPB converts telem.Frame to protobuf Frame using provided key converter.
func FrameToPB[Key types.SizedNumeric](
	ctx context.Context,
	r telem.Frame[Key],
) (*Frame, error) {
	seriesVal, err := SeriessToPB(ctx, r.SeriesSlice())
	if err != nil {
		return nil, err
	}
	pb := &Frame{
		Keys:   xunsafe.CastSlice[Key, uint32](r.KeysSlice()),
		Series: seriesVal,
	}
	return pb, nil
}

// FrameFromPB converts protobuf Frame to telem.Frame using provided key converter.
func FrameFromPB[Key types.SizedNumeric](
	ctx context.Context,
	pb *Frame,
) (telem.Frame[Key], error) {
	if pb == nil {
		return telem.Frame[Key]{}, nil
	}
	series, err := SeriessFromPB(ctx, pb.Series)
	if err != nil {
		return telem.Frame[Key]{}, err
	}
	return telem.MultiFrame(xunsafe.CastSlice[uint32, Key](pb.Keys), series), nil
}

// SeriesToPB converts Series to Series.
func SeriesToPB(ctx context.Context, r telem.Series) (*Series, error) {
	timeRangeVal, err := TimeRangeToPB(ctx, r.TimeRange)
	if err != nil {
		return nil, err
	}
	pb := &Series{
		DataType:  string(r.DataType),
		Data:      r.Data,
		Alignment: uint64(r.Alignment),
		TimeRange: timeRangeVal,
	}
	return pb, nil
}

// SeriesFromPB converts Series to Series.
func SeriesFromPB(ctx context.Context, pb *Series) (telem.Series, error) {
	var r telem.Series
	if pb == nil {
		return r, nil
	}
	var err error
	r.TimeRange, err = TimeRangeFromPB(ctx, pb.TimeRange)
	if err != nil {
		return r, err
	}
	r.DataType = telem.DataType(pb.DataType)
	r.Data = pb.Data
	r.Alignment = telem.Alignment(pb.Alignment)
	return r, nil
}

// SeriessToPB converts a slice of Series to Series.
func SeriessToPB(ctx context.Context, rs []telem.Series) ([]*Series, error) {
	result := make([]*Series, len(rs))
	for i := range rs {
		var err error
		result[i], err = SeriesToPB(ctx, rs[i])
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// SeriessFromPB converts a slice of Series to Series.
func SeriessFromPB(ctx context.Context, pbs []*Series) ([]telem.Series, error) {
	result := make([]telem.Series, len(pbs))
	for i, pb := range pbs {
		var err error
		result[i], err = SeriesFromPB(ctx, pb)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
