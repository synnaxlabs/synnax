// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, type Series, TimeStamp } from "@synnaxlabs/x";

import { type framer } from "@/framer";

export const secondsLinspace = (start: number, n: number): TimeStamp[] =>
  Array.from({ length: n }, (_, i) => start + i).map((n) => TimeStamp.seconds(n));

const isRaw = (dt: DataType): boolean => dt.isVariable || dt.equals(DataType.UINT8);

/** Narrows every numeric series to float32, anchoring bigints on an offset. Mirrors
 * the transform the visualization layer injects for WebGL rendering. */
export const glTransform: framer.Transform = {
  resolveDataType: (dt: DataType) => (isRaw(dt) ? dt : DataType.FLOAT32),
  convert: (series: Series, offset) => {
    if (isRaw(series.dataType)) return series;
    if (offset == null && series.dataType.usesBigInt && series.length > 0)
      offset = BigInt(series.data[0]);
    return series.convert(DataType.FLOAT32, offset);
  },
};
