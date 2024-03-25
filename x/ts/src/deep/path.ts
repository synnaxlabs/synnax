// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Join } from "@/join";
import { type UnknownRecord } from "@/record";

type Prev = [
  never,
  0,
  1,
  2,
  3,
  4,
  5,
  6,
  7,
  8,
  9,
  10,
  11,
  12,
  13,
  14,
  15,
  16,
  17,
  18,
  19,
  20,
  ...Array<0>,
];

export type Key<T, D extends number = 5> = [D] extends [never]
  ? never
  : T extends object
    ? {
        [K in keyof T]-?: K extends string | number
          ? `${K}` | Join<K, Key<T[K], Prev[D]>>
          : never;
      }[keyof T]
    : "";

export type Get = (<T>(obj: T, path: string, allowNull: true) => unknown | null) &
  (<T>(obj: T, path: string, allowNull?: boolean) => unknown);

export const get: Get = <V>(
  obj: V,
  path: string,
  allowNull: boolean = false,
): unknown | null => {
  const parts = path.split(".");
  if (parts.length === 1 && parts[0] === "") return obj;
  let result: UnknownRecord = obj as UnknownRecord;
  for (const part of parts) {
    const v = result[part];
    if (v == null) {
      if (allowNull) return null;
      throw new Error(`Path ${path} does not exist. ${part} is null`);
    }
    result = v as UnknownRecord;
  }
  return result;
};

export const set = <V>(obj: V, path: string, value: unknown): void => {
  const parts = path.split(".");
  let result: UnknownRecord = obj as UnknownRecord;
  for (let i = 0; i < parts.length - 1; i++) {
    const part = parts[i];
    if (result[part] == null) {
      throw new Error(`Path ${path} does not exist`);
    }
    result = result[part] as UnknownRecord;
  }
  result[parts[parts.length - 1]] = value;
};

export const element = (path: string, index: number): string => {
  const parts = path.split(".");
  if (index < 0) return parts[parts.length + index];
  return parts[index];
};

export const join = (path: string[]): string => path.join(".");

export const has = <V>(obj: V, path: string): boolean => {
  try {
    get<V>(obj, path);
    return true;
  } catch {
    return false;
  }
};
