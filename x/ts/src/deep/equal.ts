// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type primitive } from "@/primitive";

interface DeepEqualBaseRecord {
  equals?: (other: unknown) => boolean;
}

interface Equatable {
  equals: (other: unknown) => boolean;
}

const isEquatable = (value: object): value is Equatable =>
  typeof (value as DeepEqualBaseRecord).equals === "function";

const mapsEqual = (a: Map<unknown, unknown>, b: Map<unknown, unknown>): boolean => {
  if (a.size !== b.size) return false;
  for (const [key, value] of a)
    if (!b.has(key) || !equal(value, b.get(key))) return false;
  return true;
};

const setsEqual = (a: Set<unknown>, b: Set<unknown>): boolean => {
  if (a.size !== b.size) return false;
  for (const value of a) if (!b.has(value)) return false;
  return true;
};

// The verdict for values that do not compare field by field, or undefined for values
// that do.
const compareWhole = (a: object, b: object): boolean | undefined => {
  if (isEquatable(a)) return a.equals(b);
  if (isEquatable(b)) return b.equals(a);
  if (a instanceof Date || b instanceof Date)
    return a instanceof Date && b instanceof Date && a.getTime() === b.getTime();
  if (a instanceof Map || b instanceof Map)
    return a instanceof Map && b instanceof Map && mapsEqual(a, b);
  if (a instanceof Set || b instanceof Set)
    return a instanceof Set && b instanceof Set && setsEqual(a, b);
  return undefined;
};

/**
 * @returns whether the two values are equal all the way down. A value with an `equals`
 * method decides, on either side, so a `TimeStamp` compares by value. A Date compares
 * by time, a Map by its entries, and a Set by its members. Other objects compare by
 * their own enumerable fields.
 */
export const equal = <
  T extends unknown | DeepEqualBaseRecord | DeepEqualBaseRecord[] | primitive.Value[],
>(
  a: T,
  b: T,
): boolean => {
  const aIsArray = Array.isArray(a);
  const bIsArray = Array.isArray(b);
  if (aIsArray !== bIsArray) return false;
  if (aIsArray && bIsArray) {
    const aArr = a as DeepEqualBaseRecord[];
    const bArr = b as DeepEqualBaseRecord[];
    if (aArr.length !== bArr.length) return false;
    for (let i = 0; i < aArr.length; i++) if (!equal(aArr[i], bArr[i])) return false;
    return true;
  }
  if (a == null || b == null || typeof a !== "object" || typeof b !== "object")
    return a === b;
  const whole = compareWhole(a, b);
  if (whole !== undefined) return whole;
  const aKeys = Object.keys(a);
  const bKeys = Object.keys(b);
  if (aKeys.length !== bKeys.length) return false;
  for (const key of aKeys) {
    if (!Object.hasOwn(b, key)) return false;
    // @ts-expect-error - indexing nested value.
    const aVal = a[key];
    // @ts-expect-error - indexing nested value.
    const bVal = b[key];
    if (typeof aVal === "object" && typeof bVal === "object") {
      if (!equal(aVal, bVal)) return false;
    } else if (aVal !== bVal) return false;
  }
  return true;
};

/**
 * @returns whether the base matches the partial on every field the partial names.
 * Fields absent from the partial are ignored. Values that {@link equal} does not
 * compare field by field must equal in whole.
 */
export const partialEqual = <T extends unknown | DeepEqualBaseRecord | primitive.Value>(
  base: T,
  partial: Partial<T>,
): boolean => {
  if (typeof base !== "object" || base == null) return base === partial;
  if (typeof partial === "object" && partial != null) {
    const whole = compareWhole(base, partial);
    if (whole !== undefined) return whole;
  }
  const baseKeys = Object.keys(base);
  const partialKeys = Object.keys(partial);
  if (partialKeys.length > baseKeys.length) return false;
  for (const key of partialKeys) {
    // @ts-expect-error - indexing nested value.
    const baseVal = base[key];
    // @ts-expect-error - indexing nested value.
    const partialVal = partial[key];
    if (typeof baseVal === "object" && typeof partialVal === "object") {
      if (!partialEqual(baseVal, partialVal)) return false;
    } else if (baseVal !== partialVal) return false;
  }
  return true;
};
