// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resolve } from "@/deep/resolve";
import type { record } from "@/record";

export interface Get {
  <V = record.Unknown, T = record.Unknown>(obj: T, path: string): V | null;
}

export interface TypedGet<V = record.Unknown, T = record.Unknown> {
  (obj: T, path: string): V | null;
}

/**
 * Reads the value at a dot-separated path, found as {@link resolve} finds it. The empty
 * path reads the object itself.
 *
 * @returns the value, or null when nothing holds the path or the path holds null.
 * @example deep.get(config, "task.channels.0.name")
 */
export const get: Get = <V, T>(obj: T, path: string): V | null => {
  if (path === "") return obj as unknown as V;
  return (resolve(obj, path)?.value ?? null) as V | null;
};

/** @returns whether the object holds a non-null value at the path. */
export const has = <V = record.Unknown, T = record.Unknown>(
  obj: T,
  path: string,
): boolean => get<V, T>(obj, path) !== null;
