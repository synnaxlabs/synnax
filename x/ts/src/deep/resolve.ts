// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SEPARATOR } from "@/deep/path";
import type { record } from "@/record";

/** @returns the array index a path part names, or null when it is not one. */
export const getIndex = (part: string): number | null => {
  const num = parseInt(part, 10);
  if (isNaN(num) || num < 0 || num.toString() !== part) return null;
  return num;
};

const isContainer = (value: unknown): value is record.Unknown =>
  (typeof value === "object" && value !== null) || typeof value === "function";

/**
 * Finds the child of a value that one name reaches. On an array, the name is an index
 * or else the key of an entry.
 *
 * @returns the property that holds the child and the child, or null.
 */
const child = (
  value: unknown,
  name: string,
): [key: string | number, child: unknown] | null => {
  if (!isContainer(value)) return null;
  if (!Array.isArray(value))
    return Object.hasOwn(value, name) ? [name, value[name]] : null;
  const index = getIndex(name);
  if (index != null && index in value) return [index, value[index]];
  const i = value.findIndex((entry) => isContainer(entry) && entry.key === name);
  return i === -1 ? null : [i, value[i]];
};

/** A child that a run of path parts names. */
export interface Step {
  /** The property that holds the child, an index on an array. */
  key: string | number;
  value: unknown;
  /** The index of the first part after the run. */
  end: number;
}

/**
 * Finds the child of a value that the longest run of parts from start names, so a key
 * can hold the separator.
 */
export const step = (value: unknown, parts: string[], start: number): Step | null => {
  if (!isContainer(value)) return null;
  let name = parts.slice(start).join(SEPARATOR);
  for (let end = parts.length; end > start; end--) {
    const found = child(value, name);
    if (found != null) return { key: found[0], value: found[1], end };
    // end > start >= 0, so parts[end - 1] exists.
    name = name.slice(0, name.length - parts[end - 1]!.length - SEPARATOR.length);
  }
  return null;
};

/** Where a path lands. */
export interface Resolved {
  /** The object or array that holds the value. */
  parent: record.Unknown;
  /** The property of the parent that holds the value, an index on an array. */
  key: string | number;
  value: unknown;
}

/**
 * Finds the value at a dot-separated path. Each step takes the longest run of parts
 * that names a child, so a key can hold the separator. On an array, a part is an index
 * or else the key of an entry.
 *
 * @returns the value, the parent that holds it, and the key it sits at, or null when
 * nothing holds the path.
 * @example deep.resolve(task, "config.channels.ai0.port")
 */
export const resolve = (obj: unknown, path: string): Resolved | null => {
  const parts = path.split(SEPARATOR);
  let parent = obj;
  let found = step(parent, parts, 0);
  while (found != null && found.end < parts.length) {
    parent = found.value;
    found = step(parent, parts, found.end);
  }
  if (found == null) return null;
  return { parent: parent as record.Unknown, key: found.key, value: found.value };
};

/**
 * Rewrites each array index in a path into the key of the entry it reaches. The parts
 * past where the path leaves the object stay as they are.
 */
export const resolvePath = <T = record.Unknown>(path: string, obj: T): string => {
  const parts = path.split(SEPARATOR);
  const resolved: string[] = [];
  let value: unknown = obj;
  let start = 0;
  while (start < parts.length) {
    const found = step(value, parts, start);
    if (found == null) break;
    const entry =
      Array.isArray(value) && isContainer(found.value) && "key" in found.value;
    resolved.push(
      entry
        ? String((found.value as record.Unknown).key)
        : parts.slice(start, found.end).join(SEPARATOR),
    );
    value = found.value;
    start = found.end;
  }
  return [...resolved, ...parts.slice(start)].join(SEPARATOR);
};
