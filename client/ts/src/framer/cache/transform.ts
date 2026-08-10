// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type DataType, type math, type Series } from "@synnaxlabs/x";

/**
 * Converts series into the representation the caller renders from before they enter
 * cache buffers. Injected at construction; the cache applies it to every write.
 */
export interface Transform {
  /** @returns the buffer data type used to hold series of the given type. */
  resolveDataType: (dt: DataType) => DataType;
  /**
   * Converts the series to the resolved data type.
   * @param series - The series to convert.
   * @param offset - Subtracted from each sample before narrowing conversions to
   * preserve precision. Omitted when the buffer anchors off the first sample.
   */
  convert: (series: Series, offset?: math.Numeric) => Series;
}

/** A transform that stores every series exactly as it arrives. */
export const IDENTITY_TRANSFORM: Transform = {
  resolveDataType: (dt) => dt,
  convert: (series) => series,
};
