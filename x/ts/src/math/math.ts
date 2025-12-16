// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** A numeric value is either a a number, or a bigint. */
export type Numeric = number | bigint;

const multiCoercedOp =
  (func: (a: number, b: number) => Numeric) =>
  <V extends Numeric>(a: V, b: Numeric): V => {
    if (typeof a === "bigint") {
      if (isInteger(b))
        return func(a as unknown as number, BigInt(b) as unknown as number) as V;
      const res = func(Number(a), Number(b)) as number;
      if (typeof res === "number") return BigInt(Math.round(res)) as V;
      return res;
    }
    return func(Number(a), Number(b)) as V;
  };

/**
 * @returns the product of a and b, coercing b to the type of a if necessary. */
export const sub = multiCoercedOp((a, b) => a - b);

/** @returns the sum of a and b, coercing b to the type of a if necessary. */
export const add = multiCoercedOp((a, b) => a + b);

/** @returns true if a is close to b within epsilon. */
export const closeTo = (a: number, b: number, epsilon = 0.0001): boolean =>
  Math.abs(a - b) < epsilon;

/** @returns true if a is equal to b, coercing b to the type of a if necessary. */
export const equal = (a: Numeric, b: Numeric): boolean => {
  const aIsBigInt = typeof a === "bigint";
  const bIsBigInt = typeof b === "bigint";
  if (aIsBigInt && bIsBigInt) return a === b;
  if (aIsBigInt && isInteger(b)) return a === BigInt(b);
  if (bIsBigInt && isInteger(a)) return b === BigInt(a);
  return a === b;
};

/**
 * @returns the number rounded to the nearest magnitude of 10.
 * @example roundToNearestMagnitude(1234) => 1000
 * @example roundToNearestMagnitude(12345) => 10000
 * @example roundToNearestMagnitude(123456) => 100000
 */
export const roundToNearestMagnitude = (num: number): number => {
  const magnitude = 10 ** Math.floor(Math.log10(num));
  return Math.round(num / magnitude) * magnitude;
};

/** @returns the minimum of a and b, coercing b to the type of a if necessary. */
export const min = multiCoercedOp((a, b) => (a <= b ? a : b));

export const isInteger = (a: Numeric): boolean => {
  if (typeof a === "bigint") return true;
  return Number.isInteger(a);
};

/** @returns the maximum of a and b, coercing b to the type of a if necessary. */
export const max = multiCoercedOp((a, b) => (a >= b ? a : b));

/** @returns the absolute value of a. */
export const abs = <V extends Numeric>(a: V): V => {
  if (typeof a === "bigint") return (a < 0n ? -a : a) as V;
  return (a < 0 ? -a : a) as V;
};

/** @returns the multiplication of a and b, coercing b to the type of a if necessary. */
export const mult = multiCoercedOp((a, b) => a * b);

/** @returns the division of a and b, coercing b to the type of a if necessary. */
export const div = multiCoercedOp((a, b) => a / b);

/** @returns the average of an array of numbers. Returns 0 for empty arrays. */
export const average = (values: number[]): number => {
  if (values.length === 0) return 0;
  return values.reduce((a, b) => a + b, 0) / values.length;
};

/**
 * Rounds a number to a specified number of decimal places.
 * @param value The number to round.
 * @param decimals The number of decimal places (default: 1).
 * @returns The rounded number.
 * @example roundTo(1.234, 1) => 1.2
 * @example roundTo(1.234, 2) => 1.23
 */
export const roundTo = (value: number, decimals = 1): number => {
  const multiplier = 10 ** decimals;
  return Math.round(value * multiplier) / multiplier;
};

/**
 * Compares the average values of the first and last quarters of an array.
 * Useful for detecting trends like degradation or growth over time.
 *
 * @param items Array of items to analyze.
 * @param getValue Function to extract the numeric value from each item.
 * @returns Object with first quarter average, last quarter average, and quarter size.
 * @example
 * const samples = [{ fps: 60 }, { fps: 58 }, { fps: 55 }, { fps: 50 }];
 * const result = compareQuarters(samples, s => s.fps);
 * // result: { first: 60, last: 50, quarterSize: 1 }
 */
export const compareQuarters = <T>(
  items: T[],
  getValue: (item: T) => number,
): { first: number; last: number; quarterSize: number } => {
  if (items.length < 2) return { first: 0, last: 0, quarterSize: 0 };
  const quarterSize = Math.max(1, Math.floor(items.length / 4));
  const firstQuarter = items.slice(0, quarterSize);
  const lastQuarter = items.slice(-quarterSize);
  return {
    first: average(firstQuarter.map(getValue)),
    last: average(lastQuarter.map(getValue)),
    quarterSize,
  };
};
