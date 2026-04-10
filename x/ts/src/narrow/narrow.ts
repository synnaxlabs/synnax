// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@/record";

export type IsUndefined<T> = [T] extends [undefined] // T can be assigned to undefined
  ? [undefined] extends [T] // undefined can be assigned to T
    ? true // both directions → exactly undefined
    : false
  : false;

export const isObject = <T extends record.Unknown = record.Unknown>(
  item?: unknown,
): item is T => item != null && typeof item === "object" && !Array.isArray(item);

/**
 * A stricter version of {@link isObject} that additionally rejects class instances,
 * arrays, and any non-`Object.prototype` objects. Returns true only for plain objects
 * created via `{}`, `Object.create(null)`, or an object literal.
 *
 * Useful for walkers that need to distinguish "plain data bag" from "wrapped instance"
 * (e.g. `Date`, `Map`, `Error`), which `isObject` treats the same.
 */
export const isPlainObject = (item?: unknown): item is Record<string, unknown> => {
  if (item == null || typeof item !== "object") return false;
  if (Array.isArray(item)) return false;
  const proto = Object.getPrototypeOf(item);
  return proto === Object.prototype || proto === null;
};
