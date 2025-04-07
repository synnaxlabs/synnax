// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { bounds } from "@/spatial/bounds";

export const NOTATIONS = ["standard", "scientific", "engineering"] as const;
export const notationZ = z.enum(NOTATIONS);
export type Notation = z.infer<typeof notationZ>;

/**
 * Converts a number to a string representation with a specified precision and notation.
 *
 * @param value - The number to be converted.
 * @param precision - The number of decimal places to include in the output. Must be between 0 and 20.
 * @param notation - The notation to use for the conversion. Can be "standard", "scientific", or "engineering".
 * @returns The string representation of the number.
 *
 * Edge cases:
 * - If the value is `NaN`, returns "NaN".
 * - If the value is `Infinity`, returns "∞".
 * - If the value is `-Infinity`, returns "-∞".
 *
 * Examples:
 *
 * ```typescript
 * stringifyNumber(1234.5678, 2, "standard"); // "1234.57"
 * stringifyNumber(1234.5678, 2, "scientific"); // "1.23ᴇ3"
 * stringifyNumber(1234.5678, 2, "engineering"); // "1.23ᴇ3"
 * stringifyNumber(0.0001234, 4, "standard"); // "0.0001"
 * stringifyNumber(0.0001234, 4, "scientific"); // "1.2340ᴇ-4"
 * stringifyNumber(0.0001234, 4, "engineering"); // "123.4000ᴇ-6"
 * stringifyNumber(NaN, 2, "standard"); // "NaN"
 * stringifyNumber(Infinity, 2, "standard"); // "∞"
 * stringifyNumber(-Infinity, 2, "standard"); // "-∞"
 * ```
 */
export const stringifyNumber = (
  value: number,
  precision: number,
  notation: Notation,
): string => {
  if (Number.isNaN(value)) return "NaN";
  if (value === Infinity) return "∞";
  if (value === -Infinity) return "-∞";
  if (notation === "standard") return value.toFixed(precision);
  if (value === 0) {
    if (precision === 0) return "0ᴇ0";
    return `0.${"0".repeat(precision)}ᴇ0`;
  }
  let exp: number;
  if (notation === "scientific") exp = Math.floor(Math.log10(Math.abs(value)));
  else exp = Math.floor(Math.log10(Math.abs(value)) / 3) * 3;
  const mantissa = value / 10 ** exp;
  return `${mantissa.toFixed(precision)}ᴇ${exp}`;
};

/**
 * Rounds a number intelligently based on the span of the provided bounds. The function
 * adjusts the number of significant digits based on the magnitude of the bounds span.
 *
 * @param value - The number to be rounded.
 * @param bounds - The bounds object containing the min and max values that provide context for rounding.
 * @returns The rounded number.
 *
 * Rules for significant digits:
 * - For spans >= 1000: 2 significant digits
 * - For spans >= 1: 3 significant digits
 * - For spans < 1: max(2, |floor(log10(span))| + 2) significant digits
 *
 * Edge cases:
 * - If the value is `NaN`, returns `NaN`
 * - If the value is `Infinity` or `-Infinity`, returns the original value
 * - If the bounds span is 0, returns the original value
 *
 * Examples:
 * ```typescript
 * // Large spans (>= 1000) use 2 significant digits
 * roundSmart(1234.5678, { start: 0, end: 2000 }); // 1200
 *
 * // Medium spans (>= 1) use 3 significant digits
 * roundSmart(1.23456, { start: 0, end: 2 }); // 1.23
 *
 * // Small spans (< 1) adapt based on the span
 * roundSmart(0.123456, { start: 0, end: 0.2 }); // 0.123
 * roundSmart(0.0001234, { start: 0, end: 0.001 }); // 0.00012
 *
 * // Edge cases
 * roundSmart(NaN, { start: 0, end: 1 }); // NaN
 * roundSmart(Infinity, { start: 0, end: 1 }); // Infinity
 * roundSmart(123, { start: 1, end: 1 }); // 123 (span is 0)
 * ```
 */
export const roundSmart = (value: number, b: bounds.Bounds<number>): number => {
  if (Number.isNaN(value) || !Number.isFinite(value)) return value;
  const span = bounds.span(b);
  if (span == 0) return value;
  let significantDigits: number;
  if (span >= 1000) significantDigits = 2;
  else if (span >= 1) significantDigits = 3;
  else significantDigits = Math.max(2, Math.abs(Math.floor(Math.log10(span))) + 2);
  const multiplier = 10 ** significantDigits;
  return Math.round(value * multiplier) / multiplier;
};
