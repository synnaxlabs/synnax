// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Notation } from "@/notation/types.gen";

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
 * - Scientific and engineering output is renormalized after rounding so the mantissa
 *   stays in its canonical range (|m| < 10 for scientific, |m| < 1000 for engineering).
 *   For example, 9.999 at precision 1 in scientific renders as "1.0ᴇ1", not "10.0ᴇ0".
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
  value: number | bigint,
  precision: number,
  notation: Notation,
): string => {
  // Standard notation on bigint must preserve full integer precision; coercing through
  // Number() would quantize values above 2^53. All other branches are float-mantissa
  // truncated by definition, so coercing bigint to number is safe.
  if (typeof value === "bigint" && notation === "standard")
    return precision === 0 ? value.toString() : `${value}.${"0".repeat(precision)}`;
  if (typeof value === "bigint") value = Number(value);
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
  let mantissa = value / 10 ** exp;
  // After rounding via toFixed, the mantissa may have crossed the canonical upper bound
  // (>= 10 for scientific, >= 1000 for engineering). Predict the rounded magnitude with
  // Math.round — for non-negative values it matches toFixed's half-away-from-zero
  // semantics, and we're already taking Math.abs — so we avoid having to parseFloat the
  // formatted string back into a number. Bump the exponent if needed so e.g. 9.999 at
  // precision 1 in scientific becomes "1.0ᴇ1" rather than "10.0ᴇ0".
  const upperBound = notation === "scientific" ? 10 : 1000;
  const factor = 10 ** precision;
  if (Math.round(Math.abs(mantissa) * factor) / factor >= upperBound) {
    exp += notation === "scientific" ? 1 : 3;
    mantissa = value / 10 ** exp;
  }
  return `${mantissa.toFixed(precision)}ᴇ${exp}`;
};
