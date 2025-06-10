// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Key } from "@/deep/path";

// eslint-disable-next-line @typescript-eslint/no-unnecessary-type-constraint
export const deleteD = <T extends any, D extends number = 5>(
  target: T,
  ...keys: Array<Key<T, D>>
): T => {
  keys.forEach((key) => {
    let curr: any = target;
    const arr = key.split(".");
    arr.forEach((k, i) => {
      if (i === arr.length - 1) delete curr[k];
      else if (k in curr) curr = curr[k];
    });
  });
  return target;
};
