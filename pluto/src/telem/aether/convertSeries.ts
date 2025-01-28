// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, type math, type Series } from "@synnaxlabs/x";

/**
 * Converts the given serie
 *
 * @param s
 * @param offset
 * @returns
 */
export const convertSeries = (s: Series, offset?: math.Numeric): Series => {
  if (s.dataType.isVariable || s.dataType.equals(DataType.UINT8)) return s;
  if (offset == null && s.dataType.equals(DataType.TIMESTAMP))
    offset = BigInt(s.data[0]);
  return s.convert(DataType.FLOAT32, offset);
};

export const resolveDataType = (dt: DataType): DataType => {
  if (dt.isVariable || dt.equals(DataType.UINT8)) return dt;
  return DataType.FLOAT32;
};
