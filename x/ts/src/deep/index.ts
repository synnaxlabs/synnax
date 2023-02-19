// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnknownRecord } from "@/types/record";

export const isObject = (item?: unknown): boolean =>
  item != null && typeof item === "object" && !Array.isArray(item);

const deepMerge = <T extends UnknownRecord<T>>(
  base: T,
  ...objects: Array<Partial<T>>
): T => {
  if (objects.length === 0) return base;
  const source = objects.shift();

  if (isObject(base) && isObject(source)) {
    for (const key in source) {
      if (isObject(source[key])) {
        // @ts-expect-error
        if (key in base) deepMerge(base[key], source[key]);
        else Object.assign(base, { [key]: {} });
      } else {
        Object.assign(base, { [key]: source[key] });
      }
    }
  }

  return deepMerge(base, ...objects);
};

export const Deep = {
  merge: deepMerge,
};
