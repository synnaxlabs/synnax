// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv } from "@/caseconv";
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

export interface GetOptions<O extends boolean | undefined = boolean | undefined> {
  optional: O;
  getter?: (obj: UnknownRecord, key: string) => unknown;
}

export interface Get {
  <V = unknown, T = UnknownRecord>(
    obj: T,
    path: string,
    options?: GetOptions<false>,
  ): V;
  <V = unknown, T = UnknownRecord>(
    obj: T,
    path: string,
    options?: GetOptions<boolean | undefined>,
  ): V | null;
}

export interface TypedGet<V = unknown, T = UnknownRecord> {
  (obj: T, path: string, options?: GetOptions<false>): V;
  (obj: T, path: string, options?: GetOptions<boolean | undefined>): V | null;
}

export const transformPath = (
  path: string,
  replacer: (
    part: string,
    index: number,
    parts: string[],
  ) => string | string[] | undefined,
): string => {
  const parts = path.split(".");
  const result = parts
    .map((part, index) => {
      const r = replacer(part, index, parts);
      if (r == null) return null;
      if (typeof r === "string") return r;
      return r.join(".");
    })
    .filter((part) => part != null) as string[];
  return result.join(".");
};

export const get = (<V = unknown, T = UnknownRecord>(
  obj: T,
  path: string,
  opts: GetOptions = { optional: false },
): V | null => {
  const { optional, getter = (obj, key) => (obj as UnknownRecord)[key] } = opts;
  const parts = path.split(".");
  if (parts.length === 1 && parts[0] === "") return obj as unknown as V;
  let result: UnknownRecord = obj as UnknownRecord;
  for (const part of parts) {
    const v = getter(result, part);
    if (v == null) {
      if (optional) return null;
      throw new Error(`Path ${path} does not exist. ${part} is null`);
    }
    result = v as UnknownRecord;
  }
  return result as V;
}) as unknown as Get;

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

export const has = <V = unknown, T = UnknownRecord>(obj: T, path: string): boolean => {
  try {
    get<V, T>(obj, path);
    return true;
  } catch {
    return false;
  }
};

export const pathsMatch = (path: string, pattern: string): boolean => {
  if (pattern.length === 0) return true;
  const parts = path.split(".");
  const patterns = pattern.split(".");
  if (patterns.length > parts.length) return false;
  for (let i = 0; i < patterns.length; i++) {
    const part = parts[i];
    const pattern = patterns[i];
    if (pattern === "*") continue;
    if (part !== pattern) return false;
  }
  return true;
};
