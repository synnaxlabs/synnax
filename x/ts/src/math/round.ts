// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds } from "@/spatial/bounds";

const LARGE_SPAN_DECIMAL_PLACES = 2;
const MEDIUM_SPAN_DECIMAL_PLACES = 3;

// The number of additional decimal places to show past the precision of the span.
const EXTRA_DECIMAL_PLACES = 2;

const SIGNIFICANT_FIGURES = 5;
const MIN_SPAN_THRESHOLD = 1e-10;

/**
 * Intelligently rounds a number using span-based or significant figure logic.
 * Designed for UI display where floating-point noise needs to be cleaned up.
 *
 * @param value - The number to be rounded.
 * @param b - Optional bounds. Uses span-based rounding when span is significant,
 * otherwise uses significant figures.
 * @returns The rounded number.
 */
export const smartRound = (value: number, b?: bounds.Bounds<number>): number => {
  if (Number.isNaN(value) || !Number.isFinite(value)) return value;
  const absValue = Math.abs(value);
  if (absValue === 0) return 0;
  let useSpanBased = false;
  let span = 0;
  if (b != null) {
    span = bounds.span(b);
    const spanRatio = span / absValue;
    useSpanBased = span > 0 && spanRatio > MIN_SPAN_THRESHOLD;
  }
  if (useSpanBased) {
    let decimalPlaces: number;
    if (span >= 1000) decimalPlaces = LARGE_SPAN_DECIMAL_PLACES;
    else if (span >= 1) decimalPlaces = MEDIUM_SPAN_DECIMAL_PLACES;
    else {
      const decimalPlacesInSpan = Math.ceil(-Math.log10(span));
      decimalPlaces = decimalPlacesInSpan + EXTRA_DECIMAL_PLACES;
    }
    const multiplier = 10 ** decimalPlaces;
    return Math.round(value * multiplier) / multiplier;
  }
  if (absValue >= 1000) {
    const multiplier = 10 ** LARGE_SPAN_DECIMAL_PLACES;
    return Math.round(value * multiplier) / multiplier;
  }
  if (absValue >= 1) {
    const multiplier = 10 ** MEDIUM_SPAN_DECIMAL_PLACES;
    return Math.round(value * multiplier) / multiplier;
  }
  const magnitude = Math.floor(Math.log10(absValue));
  const decimalPlaces = SIGNIFICANT_FIGURES - magnitude - 1;
  const multiplier = 10 ** decimalPlaces;
  return Math.round(value * multiplier) / multiplier;
};
