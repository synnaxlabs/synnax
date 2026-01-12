// Copyright 2026 Synnax Labs, Inc.
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
