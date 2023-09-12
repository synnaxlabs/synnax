// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Partial } from "@/deep/partial";
import { isObject } from "@/identity";
import { type UnknownRecord } from "@/record";

export const merge = <T extends UnknownRecord<T>>(
  base: T,
  ...objects: Array<Partial<T>>
): T => {
  if (objects.length === 0) return base;
  const source = objects.shift();

  if (isObject(base) && isObject(source)) {
    for (const key in source) {
      try {
        if (isObject(source[key])) {
          if (!(key in base)) Object.assign(base, { [key]: {} });
          // @ts-expect-error
          merge(base[key], source[key]);
        } else {
          Object.assign(base, { [key]: source[key] });
        }
      } catch (e) {
        if (e instanceof TypeError) {
          throw new TypeError(`.${key}: ${e.message}`);
        }
        throw e;
      }
    }
  }

  return merge(base, ...objects);
};
