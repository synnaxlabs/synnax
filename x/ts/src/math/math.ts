// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

type Value = number | bigint;

/** @returns the product of a and b, coercing b to the type of a if necessary. */
export const sub = <V extends Value>(a: V, b: Value): V => {
  if (typeof a === "bigint") return (a - BigInt(b)) as V;
  return (a - Number(b)) as V;
};

/** @returns the sum of a and b, coercing b to the type of a if necessary. */
export const add = <V extends Value>(a: V, b: Value): V => {
  if (typeof a === "bigint") return (a + BigInt(b)) as V;
  // @ts-expect-error - a is a number but typescript doesn't recognize that.
  return (a + Number(b)) as V;
};

/** @returns true if a is close to b within epsilon. */
export const closeTo = (a: number, b: number, epsilon = 0.0001): boolean =>
  Math.abs(a - b) < epsilon;

/** @returns the nearest number to num with 1 significant digit. */
export const roundToNearestMagnitude = (num: number): number => {
  const magnitude = Math.pow(10, Math.floor(Math.log10(num)));
  return Math.round(num / magnitude) * magnitude;
};
