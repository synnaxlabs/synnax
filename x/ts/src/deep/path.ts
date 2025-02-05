// Copyright 2025 Synnax Labs, Inc.
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
  ...0[],
];

/**
 * A type that represents a deep key in an object.
 * @example type Key<T, 3> = "a" | "a.b" | "a.b.c"
 */
export type Key<T, D extends number = 5> = [D] extends [never]
  ? never
  : T extends object
    ? {
        [K in keyof T]-?: K extends string | number
          ? `${K}` | Join<K, Key<T[K], Prev[D]>>
          : never;
      }[keyof T]
    : "";

/** Options for the get function. */
export interface GetOptions<O extends boolean | undefined = boolean | undefined> {
  optional: O;
  getter?: (obj: UnknownRecord, key: string) => unknown;
  separator?: string;
}

/**
 * A function that gets the value at the given path on the object. If the path does not exist
 * and the optional flag is set to true, null will be returned. If the path does not exist and
 * the optional flag is set to false, an error will be thrown.
 * @param obj the object to get the value from.
 * @param path the path to get the value at.
 * @param options the options for getting the value.
 * @param options.optional whether the path is optional.
 * @param options.getter a custom getter function to use on each part of the path.
 * @returns the value at the given path on the object.
 */
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

/** A strongly typed version of the @link Get function. */
export interface TypedGet<V = unknown, T = UnknownRecord> {
  (obj: T, path: string, options?: GetOptions<false>): V;
  (obj: T, path: string, options?: GetOptions<boolean | undefined>): V | null;
}

/**
 * Executes the given replacer function on each part of the path.
 * @param path the path to transform
 * @param replacer the function to execute on each part of the path. If multiple
 * parts are returned, they will be joined with a period. If null/undefined is returned,
 * the part will be removed from the path.
 * @returns the transformed path.
 */
export const transformPath = (
  path: string,
  replacer: (
    part: string,
    index: number,
    parts: string[],
  ) => string | string[] | undefined,
  separator = ".",
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
 * Gets the value at the given path on the object. If the path does not exist
 * and the optional flag is set to true, null will be returned. If the path does
 * not exist and the optional flag is set to false, an error will be thrown.
 * @param obj the object to get the value from.
 * @param path the path to get the value at.
 * @param opts the options for getting the value.
 * @param opts.optional whether the path is optional.
 * @param opts.getter a custom getter function to use on each part of the path.
 * @returns the value at the given path on the object.
 */
export const get = (<V = unknown, T = UnknownRecord>(
  obj: T,
  path: string,
  opts: GetOptions = { optional: false, separator: "." },
): V | null => {
  opts.separator ??= ".";
  const { optional, getter = (obj, key) => (obj)[key] } = opts;
  const parts = path.split(opts.separator);
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
}) as Get;

/**
 * Sets the value at the given path on the object. If the parents of the deep path
 * do not exist, new objects will be created.
 * @param obj the object to set the value on.
 * @param path the path to set the value at.
 * @param value the value to set.
 */
export const set = <V>(obj: V, path: string, value: unknown): void => {
  const parts = path.split(".");
  let result: UnknownRecord = obj as UnknownRecord;
  for (let i = 0; i < parts.length - 1; i++) {
    const part = parts[i];
    result[part] ??= {};
    result = result[part] as UnknownRecord;
  }
  try {
    result[parts[parts.length - 1]] = value;
  } catch (e) {
    console.error("failed to set value", value, "at path", path, "on object", obj);
    throw e;
  }
};

/**
 * Removes the value at the given path, modifying the object in place.
 * @param obj the object to remove the value from.
 * @param path the path to remove the value from.
 * @returns the object with the value removed.
 */
export const remove = <V>(obj: V, path: string): void => {
  const parts = path.split(".");
  let result: UnknownRecord = obj as UnknownRecord;
  for (let i = 0; i < parts.length - 1; i++) {
    const part = parts[i];
    if (result[part] == null) return;
    result = result[part] as UnknownRecord;
  }
  // if its an array, we need to splice it
  if (Array.isArray(result)) {
    const index = parseInt(parts[parts.length - 1]);
    if (isNaN(index)) return;
    result.splice(index, 1);
    return;
  }
  delete result[parts[parts.length - 1]];
};

/**
 * Returns the element at the given index in the path.
 * @param path the path to get the element from
 * @param index the index of the element to get
 * @returns the element at the given index in the path
 */
export const element = (path: string, index: number): string => {
  const parts = path.split(".");
  if (index < 0) return parts[parts.length + index];
  return parts[index];
};

/**
 * Checks if the path exists in the object.
 * @param obj the object to check
 * @param path the path to check
 * @returns whether the path exists in the object
 */
export const has = <V = unknown, T = UnknownRecord>(obj: T, path: string): boolean => {
  try {
    get<V, T>(obj, path);
    return true;
  } catch {
    return false;
  }
};

/**
 * Checks if the path matches the given pattern.
 *
 * @param path The path to check.
 * @param pattern The pattern to match against. Only "*" is supported as a wildcard.
 * @returns Whether the path matches the pattern.
 *
 *  * @example
 * pathsMatch("a.b.c", "a.b.c") // true
 * pathsMatch("a.b.c", "a.b") // true
 * pathsMatch("a.b", "a.b.c") // false
 * pathsMatch("a.b.c", "a.*") // true
 * pathsMatch("a.b.c", "a.*.c") // true
 */
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
