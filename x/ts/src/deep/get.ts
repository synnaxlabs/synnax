// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { defaultGetter, SEPARATOR } from "@/deep/path";
import { type record } from "@/record";

export interface GetOptions<O extends boolean | undefined = boolean | undefined> {
  optional: O;
  getter?: (obj: record.Unknown, key: string) => unknown;
}

export interface Get {
  <V = record.Unknown, T = record.Unknown>(
    obj: T,
    path: string,
    options?: GetOptions<false>,
  ): V;
  <V = record.Unknown, T = record.Unknown>(
    obj: T,
    path: string,
    options?: GetOptions<boolean | undefined>,
  ): V | null;
}

export interface TypedGet<V = record.Unknown, T = record.Unknown> {
  (obj: T, path: string, options?: GetOptions<false>): V;
  (obj: T, path: string, options?: GetOptions<boolean | undefined>): V | null;
}

export const get = (<V = record.Unknown, T = record.Unknown>(
  obj: T,
  path: string,
  opts: GetOptions = { optional: false },
): V | null => {
  const { optional, getter = defaultGetter } = opts;
  if (path === "") return obj as record.Unknown as V;

  const parts = path.split(SEPARATOR);
  if (parts.length === 1) {
    const v = getter(obj as record.Unknown, parts[0]);
    if (v === undefined) {
      if (optional) return null;
      throw new Error(`Path ${path} does not exist. ${parts[0]} is undefined`);
    }
    return v as V;
  }

  const tryGet = (currentObj: record.Unknown, partIndex: number): V | null => {
    if (partIndex >= parts.length) return currentObj as V;

    for (let i = parts.length - partIndex; i >= 1; i--) {
      const combinedKey = parts.slice(partIndex, partIndex + i).join(SEPARATOR);
      const v = getter(currentObj, combinedKey);
      if (v !== undefined) {
        if (partIndex + i === parts.length) return v as V;
        if (v === null) {
          if (optional) return null;
          throw new Error(`Path ${path} does not exist. ${combinedKey} is null`);
        }
        return tryGet(v as record.Unknown, partIndex + i);
      }
    }
    if (optional) return null;
    throw new Error(`Path ${path} does not exist. ${parts[partIndex]} is undefined`);
  };

  return tryGet(obj as record.Unknown, 0);
}) as Get;

export const has = <V = record.Unknown, T = record.Unknown>(
  obj: T,
  path: string,
): boolean => get<V, T>(obj, path, { optional: true }) !== null;
