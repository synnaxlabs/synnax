// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { get } from "@/deep/get";
import { SEPARATOR } from "@/deep/path";
import { getIndex, resolve } from "@/deep/resolve";
import { type record } from "@/record";

// A part that names nothing becomes an index on an array and a key elsewhere.
const add = (parent: unknown, part: string, value: unknown, path: string): void => {
  if (!Array.isArray(parent)) {
    (parent as record.Unknown)[part] = value;
    return;
  }
  const index = getIndex(part);
  if (index == null)
    throw new Error(`Cannot set ${path}: the list holds no entry with the key ${part}`);
  parent[index] = value;
};

// The value at the path, built when it is missing: an array when the next part is an
// index, an object otherwise.
const container = (obj: unknown, path: string, next: string, full: string): unknown => {
  const existing = get(obj, path);
  if (existing != null) return existing;
  const created = getIndex(next) == null ? {} : [];
  write(obj, path, created, full);
  return created;
};

const write = (obj: unknown, path: string, value: unknown, full: string): void => {
  const found = resolve(obj, path);
  if (found != null) {
    found.parent[found.key] = value;
    return;
  }
  const cut = path.lastIndexOf(SEPARATOR);
  const last = path.slice(cut + 1);
  const parent = cut === -1 ? obj : container(obj, path.slice(0, cut), last, full);
  add(parent, last, value, full);
};

/**
 * Writes the value at a dot-separated path, found as {@link resolve} finds it. A
 * missing last part becomes one key, and missing parents are built on the way: an array
 * when the next part is an index, an object otherwise.
 *
 * @throws {Error} if the path goes into an array at a part that is neither an index nor
 * the key of an entry, as when the entry was removed.
 * @example deep.set(config, "task.channels.ai0.port", 2)
 */
export const set = <V>(obj: V, path: string, value: unknown): void =>
  write(obj, path, value, path);
