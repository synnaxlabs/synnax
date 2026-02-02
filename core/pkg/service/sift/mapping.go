// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift

import (
	"github.com/samber/lo"
	typev1 "github.com/sift-stack/sift/go/gen/sift/common/type/v1"
	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// MapDataType converts a Synnax data type to a Sift channel data type.
func MapDataType(dt telem.DataType) (typev1.ChannelDataType, error) {
	switch dt {
	case telem.Float64T:
		return typev1.ChannelDataType_CHANNEL_DATA_TYPE_DOUBLE, nil
	case telem.Float32T:
		return typev1.ChannelDataType_CHANNEL_DATA_TYPE_FLOAT, nil
	case telem.Int64T:
		return typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_64, nil
	case telem.Int32T:
		return typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32, nil
	case telem.Uint64T:
		return typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_64, nil
	case telem.Uint32T:
		return typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32, nil
	case telem.StringT:
		return typev1.ChannelDataType_CHANNEL_DATA_TYPE_STRING, nil
	case telem.TimeStampT:
		// Timestamps are sent as int64 nanoseconds
		return typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_64, nil
	case telem.Int8T:
		return typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32, nil
	case telem.Int16T:
		return typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32, nil
	case telem.Uint8T:
		return typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32, nil
	case telem.Uint16T:
		return typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32, nil
	case telem.BytesT:
		return typev1.ChannelDataType_CHANNEL_DATA_TYPE_BYTES, nil
	default:
		return 0, errors.Newf("unsupported data type for Sift: %s", dt)
	}
}

// ConvertSeriesToProtoValues converts a Synnax series to Sift proto channel values.
func ConvertSeriesToProtoValues(
	series telem.Series,
) ([]*ingestv1.IngestWithConfigDataChannelValue, error) {
	switch series.DataType {
	case telem.Float64T:
		data := telem.UnmarshalSlice[float64](series.Data, series.DataType)
		result2 := lo.Map(data, func(v float64, _ int) *ingestv1.IngestWithConfigDataChannelValue {
			return &ingestv1.IngestWithConfigDataChannelValue{
				Type: &ingestv1.IngestWithConfigDataChannelValue_Double{Double: v},
			}
		})
		return result2, nil
	case telem.Float32T:
		data := telem.UnmarshalSlice[float32](series.Data, series.DataType)
		result := make([]*ingestv1.IngestWithConfigDataChannelValue, len(data))
		for i, v := range data {
			result[i] = &ingestv1.IngestWithConfigDataChannelValue{
				Type: &ingestv1.IngestWithConfigDataChannelValue_Float{Float: v},
			}
		}
		return result, nil
	case telem.Int64T:
		data := telem.UnmarshalSlice[int64](series.Data, series.DataType)
		result := make([]*ingestv1.IngestWithConfigDataChannelValue, len(data))
		for i, v := range data {
			result[i] = &ingestv1.IngestWithConfigDataChannelValue{
				Type: &ingestv1.IngestWithConfigDataChannelValue_Int64{Int64: v},
			}
		}
		return result, nil
	case telem.Int32T:
		data := telem.UnmarshalSlice[int32](series.Data, series.DataType)
		result := make([]*ingestv1.IngestWithConfigDataChannelValue, len(data))
		for i, v := range data {
			result[i] = &ingestv1.IngestWithConfigDataChannelValue{
				Type: &ingestv1.IngestWithConfigDataChannelValue_Int32{Int32: v},
			}
		}
		return result, nil
	case telem.Uint64T:
		data := telem.UnmarshalSlice[uint64](series.Data, series.DataType)
		result := make([]*ingestv1.IngestWithConfigDataChannelValue, len(data))
		for i, v := range data {
			result[i] = &ingestv1.IngestWithConfigDataChannelValue{
				Type: &ingestv1.IngestWithConfigDataChannelValue_Uint64{Uint64: v},
			}
		}
		return result, nil
	case telem.Uint32T:
		data := telem.UnmarshalSlice[uint32](series.Data, series.DataType)
		result := make([]*ingestv1.IngestWithConfigDataChannelValue, len(data))
		for i, v := range data {
			result[i] = &ingestv1.IngestWithConfigDataChannelValue{
				Type: &ingestv1.IngestWithConfigDataChannelValue_Uint32{Uint32: v},
			}
		}
		return result, nil
	case telem.TimeStampT:
		data := telem.UnmarshalSlice[telem.TimeStamp](series.Data, series.DataType)
		result := make([]*ingestv1.IngestWithConfigDataChannelValue, len(data))
		for i, v := range data {
			result[i] = &ingestv1.IngestWithConfigDataChannelValue{
				Type: &ingestv1.IngestWithConfigDataChannelValue_Int64{Int64: int64(v)},
			}
		}
		return result, nil
	case telem.Int8T:
		data := telem.UnmarshalSlice[int8](series.Data, series.DataType)
		result := make([]*ingestv1.IngestWithConfigDataChannelValue, len(data))
		for i, v := range data {
			result[i] = &ingestv1.IngestWithConfigDataChannelValue{
				Type: &ingestv1.IngestWithConfigDataChannelValue_Int32{Int32: int32(v)},
			}
		}
		return result, nil
	case telem.Int16T:
		data := telem.UnmarshalSlice[int16](series.Data, series.DataType)
		result := make([]*ingestv1.IngestWithConfigDataChannelValue, len(data))
		for i, v := range data {
			result[i] = &ingestv1.IngestWithConfigDataChannelValue{
				Type: &ingestv1.IngestWithConfigDataChannelValue_Int32{Int32: int32(v)},
			}
		}
		return result, nil
	case telem.Uint8T:
		data := telem.UnmarshalSlice[uint8](series.Data, series.DataType)
		result := make([]*ingestv1.IngestWithConfigDataChannelValue, len(data))
		for i, v := range data {
			result[i] = &ingestv1.IngestWithConfigDataChannelValue{
				Type: &ingestv1.IngestWithConfigDataChannelValue_Uint32{Uint32: uint32(v)},
			}
		}
		return result, nil
	case telem.Uint16T:
		data := telem.UnmarshalSlice[uint16](series.Data, series.DataType)
		result := make([]*ingestv1.IngestWithConfigDataChannelValue, len(data))
		for i, v := range data {
			result[i] = &ingestv1.IngestWithConfigDataChannelValue{
				Type: &ingestv1.IngestWithConfigDataChannelValue_Uint32{Uint32: uint32(v)},
			}
		}
		return result, nil
	default:
		return nil,
			errors.Newf("unsupported data type for proto conversion: %s", series.DataType)
	}
}
