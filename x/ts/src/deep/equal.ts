// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnknownRecord } from "@/record";

export const deepEqual = <T extends UnknownRecord<T>>(a: T, b: T): boolean => {
  const aKeys = Object.keys(a);
  const bKeys = Object.keys(b);
  if (aKeys.length !== bKeys.length) return false;
  for (const key of aKeys) {
    // @ts-expect-error
    const aVal = a[key];
    // @ts-expect-error
    const bVal = b[key];
    if (typeof aVal === "object" && typeof bVal === "object") {
      if (!deepEqual(aVal, bVal)) return false;
    } else if (aVal !== bVal) return false;
  }
  return true;
};

export const deepPartialEqual = <T extends UnknownRecord<T>>(
  base: T,
  partial: Partial<T>
): boolean => {
  const baseKeys = Object.keys(base);
  const partialKeys = Object.keys(partial);
  if (partialKeys.length > baseKeys.length) return false;
  for (const key of partialKeys) {
    // @ts-expect-error
    const baseVal = base[key];
    // @ts-expect-error
    const partialVal = partial[key];
    if (typeof baseVal === "object" && typeof partialVal === "object") {
      if (!deepPartialEqual(baseVal, partialVal)) return false;
    } else if (baseVal !== partialVal) return false;
  }
  return true;
};
