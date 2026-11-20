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

type Writer = Pick<ContextValue, "batch" | "set" | "remove">;

const write = (
  ctx: Writer,
  path: deep.Path<unknown>,
  prev: unknown,
  next: unknown,
): void => {
  if (prev === next) return;
  if (array.isKeyed(prev) && array.isKeyed(next)) {
    const keys = next.map(({ key }) => key);
    if (
      array.equal(
        prev.map(({ key }) => key),
        keys,
      )
    ) {
      next.forEach((entry, i) => write(ctx, path[keys[i]], prev[i], entry));
      return;
    }
  }
  if (narrow.isPlainObject(prev) && narrow.isPlainObject(next)) {
    Object.keys(next).forEach((key) => write(ctx, path[key], prev[key], next[key]));
    Object.keys(prev).forEach((key) => {
      if (!(key in next)) ctx.remove(path[key]);
    });
    return;
  }
  if (!deep.equal(prev, next)) ctx.set(path, next);
};

/**
 * Sets at the path, as one batch, only the values in next that differ from base, and
 * removes the fields of base that next lacks. A form value that changed after base was
 * read keeps its change unless next changes it too. Lists of keyed entries with the
 * same keys in the same order compare entry by entry, and objects compare field by
 * field. Other values that differ are set whole.
 */
export const setChanged = (
  ctx: Writer,
  path: deep.Target<unknown>,
  base: unknown,
  next: unknown,
): void => ctx.batch(() => write(ctx, deep.widen<unknown>(path), base, next));
