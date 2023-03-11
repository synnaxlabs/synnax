// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DeepKey } from "@/deep/key";
import { UnknownRecord } from "@/record";

export const deepDelete = <T extends UnknownRecord<T>, D extends number = 5>(
  target: T,
  ...keys: Array<DeepKey<T, D>>
): T => {
  // sometimes we get 'unnecessarily deep' errors here with certain versions of typescript
  // so don't remove this directive even if ts says it's unnecessary
  // @ts-expect-error
  keys.forEach((key) => {
    let curr: any = target;
    const arr = key.split(".");
    // @ts-expect-error
    arr.forEach((k, i) => {
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      if (i === arr.length - 1) delete curr[k];
      else curr = curr[k];
    });
  });
  return target;
};
