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

/**
 * @returns the product of a and b, coercing b to the type of a if necessary. */
export const sub = <V extends Numeric>(a: V, b: Numeric): V => {
  if (typeof a === "bigint") return (a - BigInt(b)) as V;
  return (a - Number(b)) as V;
};

/** @returns the sum of a and b, coercing b to the type of a if necessary. */
export const add = <V extends Numeric>(a: V, b: Numeric): V => {
  if (typeof a === "bigint") return (a + BigInt(b)) as V;
  return ((a as unknown as number) + Number(b)) as V;
};

/** @returns true if a is close to b within epsilon. */
export const closeTo = (a: number, b: number, epsilon = 0.0001): boolean =>
  Math.abs(a - b) < epsilon;

/** @returns true if a is equal to b, coercing b to the type of a if necessary. */
export const equal = <V extends Numeric>(a: V, b: Numeric): boolean => {
  if (typeof a === "bigint") return a === BigInt(b);
  return a === Number(b);
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
export const min = <V extends Numeric>(a: V, b: Numeric): V => {
  if (a <= b) return a;
  if (typeof a === "bigint") return (a <= b ? a : BigInt(b)) as V;
  return (a <= b ? a : Number(b)) as V;
};

/** @returns the maximum of a and b, coercing b to the type of a if necessary. */
export const max = <V extends Numeric>(a: V, b: Numeric): V => {
  if (typeof a === "bigint") return (a >= b ? a : BigInt(b)) as V;
  return (a >= b ? a : Number(b)) as V;
};

/** @returns the absolute value of a. */
export const abs = <V extends Numeric>(a: V): V => {
  if (a < 0) return -a as V;
  return a;
};

/** @returns the multiplication of a and b, coercing b to the type of a if necessary. */
export const mult = <V extends Numeric>(a: V, b: Numeric): V => {
  if (typeof a === "bigint") return (a * BigInt(b)) as V;
  return (a * Number(b)) as V;
};

/** @returns the division of a and b, coercing b to the type of a if necessary. */
export const div = <V extends Numeric>(a: V, b: Numeric): V => {
  if (typeof a === "bigint") return (a / BigInt(b)) as V;
  return (a / Number(b)) as V;
};
