// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Join } from "@/deep/join";
import { type record } from "@/record";

export const SEPARATOR = ".";

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
  ...0[],
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

export const transformPath = (
  path: string,
  replacer: (
    part: string,
    index: number,
    parts: string[],
  ) => string | string[] | undefined,
  separator: string = SEPARATOR,
): string => {
  const parts = path.split(separator);
  const result = parts
    .map((part, index) => {
      const r = replacer(part, index, parts);
      if (r == null) return null;
      if (typeof r === "string") return r;
      return r.join(separator);
    })
    .filter((part) => part != null);
  return result.join(separator);
};

export const defaultGetter = (obj: record.Unknown, key: string): unknown => {
  if (!Array.isArray(obj)) return obj[key];
  const res = obj[key];
  if (res != null || obj.length == 0) return res;
  const first = obj[0];
  if (typeof first === "object" && "key" in first)
    return obj.find((o) => o.key === key);
  return undefined;
};

export const resolvePath = <T = record.Unknown>(path: string, obj: T): string => {
  const parts = path.split(SEPARATOR);
  parts.forEach((part, i) => {
    obj = defaultGetter(obj as record.Unknown, part) as T;
    if (obj != null && typeof obj === "object" && "key" in obj)
      parts[i] = obj.key as string;
  });
  return parts.join(SEPARATOR);
};

export const element = (path: string, index: number): string => {
  const parts = path.split(SEPARATOR);
  if (index < 0) return parts[parts.length + index];
  return parts[index];
};

export const pathsMatch = (path: string, pattern: string): boolean => {
  if (pattern.length === 0) return true;
  const parts = path.split(SEPARATOR);
  const patterns = pattern.split(SEPARATOR);
  if (patterns.length > parts.length) return false;
  for (let i = 0; i < patterns.length; i++) {
    const part = parts[i];
    const pattern = patterns[i];
    if (pattern === "*") continue;
    if (part !== pattern) return false;
  }
  return true;
};

export const getIndex = (part: string): number | null => {
  const num = parseInt(part);
  if (isNaN(num) || num < 0 || num.toString() !== part) return null;
  return num;
};

export const findBestKey = (
  obj: record.Unknown,
  remainingParts: string[],
): [string, number] | null => {
  for (let i = 1; i <= remainingParts.length; i++) {
    const candidateKey = remainingParts.slice(0, i).join(SEPARATOR);
    const v = defaultGetter(obj, candidateKey);
    if (v != null) return [candidateKey, i];
  }
  return null;
};
