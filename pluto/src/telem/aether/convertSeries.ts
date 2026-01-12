// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, type math, type Series } from "@synnaxlabs/x";

/**
 * Converts the given series to a supported data type for pluto WebGL rendered components
 * (such as lines). If the series is a uint8 or has a variable data type, it is returned
 * as is. If the series of any other type, it is converted to a float32 series with the given
 * offset applied.
 *
 * @param series - The series to convert.
 * @param offset - An optional offset to apply to the series. If the series is a timestamp
 * series, the default offset is applied to the first value in the series. This helps fix
 * issues with reducing precision from uint64s to float32s at high nanosecond values.
 * @returns The converted series.
 */
export const convertSeriesToSupportedGL = (
  series: Series,
  offset?: math.Numeric,
): Series => {
  if (series.dataType.isVariable || series.dataType.equals(DataType.UINT8))
    return series;
  if (offset == null && series.dataType.equals(DataType.TIMESTAMP))
    offset = BigInt(series.data[0]);
  return series.convert(DataType.FLOAT32, offset);
};

/**
 * Resolves the data type to a supported data type for pluto WebGL rendered components
 * (such as lines). If the data type is variable density or uint8, it is returned as is.
 * If the data type is any other type, float32 is returned.
 *
 * @param dt - The data type to resolve.
 * @returns The resolved data type.
 */
export const resolveGLDataType = (dt: DataType): DataType => {
  if (dt.isVariable || dt.equals(DataType.UINT8)) return dt;
  return DataType.FLOAT32;
};
