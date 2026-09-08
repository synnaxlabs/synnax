// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Join } from "@/deep/join";
import { type record } from "@/record";

/** The character joining the parts of a deep path. */
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

/** Every deep path into T, as a dot-separated string literal union. */
export type Key<T, D extends number = 5> = [D] extends [never]
  ? never
  : T extends object
    ? {
        [K in keyof T]-?: K extends string | number
          ? `${K}` | Join<K, Key<T[K], Prev[D]>>
          : never;
      }[keyof T]
    : "";

/**
 * Rewrites each part of a path. A replacer may return one part, several, or undefined
 * to drop the part.
 */
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

/**
 * Reads one path part off an object. On an array of keyed entries, the part matches the
 * `key` field rather than the index.
 */
export const defaultGetter = (obj: record.Unknown, key: string): unknown => {
  if (!Array.isArray(obj)) return obj[key];
  const res = obj[key];
  if (res != null || obj.length == 0) return res;
  const first = obj[0];
  if (typeof first === "object" && "key" in first)
    return obj.find((o) => o.key === key);
  return undefined;
};

/** Rewrites the array indexes in a path into the `key` of the entry each one reaches. */
export const resolvePath = <T = record.Unknown>(path: string, obj: T): string => {
  const parts = path.split(SEPARATOR);
  parts.forEach((part, i) => {
    obj = defaultGetter(obj as record.Unknown, part) as T;
    if (obj != null && typeof obj === "object" && "key" in obj)
      parts[i] = obj.key as string;
  });
  return parts.join(SEPARATOR);
};

/** @returns one part of a path. A negative index counts back from the end. */
export const element = (path: string, index: number): string => {
  const parts = path.split(SEPARATOR);
  if (index < 0) return parts[parts.length + index];
  return parts[index];
};

/**
 * @returns whether the path starts with the pattern, where `*` matches any one part.
 * An empty pattern matches everything.
 */
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

/** @returns the array index a path part names, or null when it is not one. */
export const getIndex = (part: string): number | null => {
  const num = parseInt(part, 10);
  if (isNaN(num) || num < 0 || num.toString() !== part) return null;
  return num;
};

/**
 * Finds the longest run of leading path parts that names something on the object, for
 * keys that carry the separator inside them.
 *
 * @returns the key and how many parts it consumed, or null when nothing matches.
 */
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
