// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, type record } from "@synnaxlabs/x";

import { type ContextValue } from "@/form/Context";
import { type Entry } from "@/form/store";

const isPlainObject = (value: unknown): value is record.Unknown =>
  value != null &&
  typeof value === "object" &&
  Object.getPrototypeOf(value) === Object.prototype;

const isEntryList = (value: unknown): value is Entry[] =>
  Array.isArray(value) &&
  value.every((v) => isPlainObject(v) && typeof v.key === "string");

const sameKeys = (a: string[], b: string[]): boolean =>
  a.length === b.length && a.every((key, i) => key === b[i]);

const child = (path: string, key: string): string =>
  path === "" ? key : `${path}.${key}`;

type Target = Pick<ContextValue, "batch" | "get" | "set" | "value">;

const write = (
  ctx: Target,
  path: string,
  prev: unknown,
  next: unknown,
): void => {
  if (prev === next) return;
  if (isEntryList(prev) && isEntryList(next)) {
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
  if (isPlainObject(prev) && isPlainObject(next)) {
    const keys = Object.keys(next);
    if (sameKeys(Object.keys(prev).sort(), [...keys].sort())) {
      keys.forEach((key) => write(ctx, child(path, key), prev[key], next[key]));
      return;
    }
  }
  if (!deep.equal(prev, next)) ctx.set(path, next);
};

/**
 * Sets the value at the path as one batch, writing only the values that differ from
 * the form's. Lists of keyed entries with the same keys in the same order compare entry
 * by entry, and objects with the same fields compare field by field. Other values that
 * differ are set whole.
 */
export const setChanged = (
  ctx: Target,
  path: string,
  value: unknown,
): void => {
  const prev = path === "" ? ctx.value() : ctx.get(path, { optional: true })?.value;
  ctx.batch(() => write(ctx, path, prev, value));
};
