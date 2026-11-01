// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, deep, narrow } from "@synnaxlabs/x";

import { type ContextValue } from "@/form/Context";

const sameKeys = (a: string[], b: string[]): boolean =>
  a.length === b.length && a.every((key, i) => key === b[i]);

const child = (path: string, key: string): string =>
  path === "" ? key : `${path}.${key}`;

type Target = Pick<ContextValue, "batch" | "set">;

const write = (ctx: Target, path: string, prev: unknown, next: unknown): void => {
  if (prev === next) return;
  if (array.isKeyed(prev) && array.isKeyed(next)) {
    const keys = next.map(({ key }) => key);
    if (
      sameKeys(
        prev.map(({ key }) => key),
        keys,
      )
    ) {
      next.forEach((entry, i) => write(ctx, child(path, keys[i]), prev[i], entry));
      return;
    }
  }
  if (narrow.isPlainObject(prev) && narrow.isPlainObject(next)) {
    const keys = Object.keys(next);
    if (sameKeys(Object.keys(prev).sort(), [...keys].sort())) {
      keys.forEach((key) => write(ctx, child(path, key), prev[key], next[key]));
      return;
    }
  }
  if (!deep.equal(prev, next)) ctx.set(path, next);
};

/**
 * Sets at the path, as one batch, only the values in next that differ from base. A form
 * value that changed after base was read keeps its change unless next changes it too.
 * Lists of keyed entries with the same keys in the same order compare entry by entry,
 * and objects with the same fields compare field by field. Other values that differ are
 * set whole.
 */
export const setChanged = (
  ctx: Target,
  path: string,
  base: unknown,
  next: unknown,
): void => ctx.batch(() => write(ctx, path, base, next));
