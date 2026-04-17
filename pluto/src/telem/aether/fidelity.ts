// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Rate, type TimeSpan } from "@synnaxlabs/x";

import { NATIVE_FIDELITY } from "@/telem/client/cache/unary";

export { NATIVE_FIDELITY };

export interface FidelityForProps {
  /**
   * The time range the plot is rendering. For a static plot this is the
   * configured timeRange; for a dynamic plot it's the current live window.
   */
  timeSpan: TimeSpan;
  /**
   * The channel's sample rate in Hz. Used to convert timeSpan into a native
   * sample count.
   */
  rate: Rate;
  /**
   * The pixel width of the plotting region. Fewer pixels mean coarser
   * fidelity is acceptable.
   */
  pixelWidth: number;
  /**
   * Oversampling factor: how many server-returned samples per output pixel.
   * Default 2 hedges against aliasing for every-Nth aggregation. Min/max and
   * M4 reducers work well at k=1.
   */
  oversampleFactor?: number;
}

/**
 * Computes the coarsest acceptable fidelity (as a bigint alignmentMultiple)
 * for rendering `timeSpan` of data from a channel at the given `rate` across
 * `pixelWidth` pixels. Result is clamped to >= 1.
 *
 * The formula is `sampleCount / (pixelWidth * oversampleFactor)`, where
 * `sampleCount` is derived from `timeSpan * rate`. A 3-day time span at 1kHz
 * rendered at 2000px with k=2 yields fidelity ~= 64800 (one output per ~65s
 * of raw data). For short ranges or wide plots where the raw sample count is
 * already comparable to the pixel count, the result clamps down to 1 (raw).
 */
export const fidelityFor = ({
  timeSpan,
  rate,
  pixelWidth,
  oversampleFactor = 2,
}: FidelityForProps): bigint => {
  if (pixelWidth <= 0 || oversampleFactor <= 0) return NATIVE_FIDELITY;
  const sampleCount = Number(timeSpan.seconds) * Number(rate.valueOf());
  if (!isFinite(sampleCount) || sampleCount <= 0) return NATIVE_FIDELITY;
  const raw = Math.floor(sampleCount / (pixelWidth * oversampleFactor));
  if (raw <= 1) return NATIVE_FIDELITY;
  return BigInt(raw);
};
