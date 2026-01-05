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
  if ("equals" in a) return (a.equals as (other: unknown) => boolean)(b);
  const aKeys = Object.keys(a);
  const bKeys = Object.keys(b);
  if (aKeys.length !== bKeys.length) return false;
  for (const key of aKeys) {
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

export const partialEqual = <T extends unknown | DeepEqualBaseRecord | primitive.Value>(
  base: T,
  partial: Partial<T>,
): boolean => {
  if (typeof base !== "object" || base == null) return base === partial;
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
