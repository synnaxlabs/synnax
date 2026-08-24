// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type framer } from "@synnaxlabs/client";
import { DataType, type math, type Series } from "@synnaxlabs/x";

/**
 * Converts the given series to a supported data type for pluto WebGL rendered
 * components (such as lines). If the series is a uint8 or has a variable data type, it
 * is returned as is. If the series of any other type, it is converted to a float32
 * series with the given offset applied.
 * @param offset - An optional offset to apply to the series. If the series uses bigint
 * storage (timestamp, int64, uint64) and no offset is provided, the first sample is
 * used as the default offset. This preserves precision when narrowing 64-bit integers
 * to float32, which would otherwise quantize values above 2^53 to multiples of the
 * float32 ULP at that magnitude.
 * @returns The converted series.
 */
export const convertSeriesToSupportedGL = (
  series: Series,
  offset?: math.Numeric,
): Series => {
  if (series.dataType.isVariable || series.dataType.equals(DataType.UINT8))
    return series;
  if (offset == null && series.dataType.usesBigInt && series.length > 0)
    offset = BigInt(series.data[0]);
  return series.convert(DataType.FLOAT32, offset);
};

/**
 * Resolves the data type to a supported data type for pluto WebGL rendered components
 * (such as lines). If the data type is variable density or uint8, it is returned as is.
 * If the data type is any other type, float32 is returned.
 * @returns The resolved data type.
 */
export const resolveGLDataType = (dt: DataType): DataType => {
  if (dt.isVariable || dt.equals(DataType.UINT8)) return dt;
  return DataType.FLOAT32;
};

/** Stores cached series in a WebGL-uploadable representation. */
export const GL_TRANSFORM: framer.Transform = {
  resolveDataType: resolveGLDataType,
  convert: convertSeriesToSupportedGL,
};
