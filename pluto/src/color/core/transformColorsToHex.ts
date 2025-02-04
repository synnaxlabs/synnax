// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Color } from "@/color/core/color";

export type ReplaceColorWithHex<T> = T extends Color
  ? string
  : T extends Array<infer U>
    ? Array<ReplaceColorWithHex<U>>
    : T extends object
      ? { [K in keyof T]: ReplaceColorWithHex<T[K]> }
      : T;

export const transformColorsToHex = <T>(obj: T): ReplaceColorWithHex<T> => {
  if (obj instanceof Color) return obj.hex as ReplaceColorWithHex<T>;
  if (typeof obj === "object" && obj !== null) {
    const newObj: any = Array.isArray(obj) ? [] : {};
    for (const key of Object.keys(obj))
      newObj[key] = transformColorsToHex((obj as any)[key]);
    return newObj as ReplaceColorWithHex<T>;
  }
  return obj as ReplaceColorWithHex<T>;
};
