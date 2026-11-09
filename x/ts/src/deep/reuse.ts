// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array } from "@/array";
import { equal } from "@/deep/equal";
import { narrow } from "@/narrow";
import { type record } from "@/record";

const reuseList = (prev: unknown[], next: unknown[]): unknown[] => {
  let matched: (i: number) => unknown;
  if (array.isKeyed(prev) && array.isKeyed(next)) {
    const byKey = new Map(prev.map((entry) => [entry.key, entry]));
    matched = (i) => byKey.get(next[i].key);
  } else matched = (i) => prev[i];
  const out = next.map((v, i) => reuse(matched(i), v));
  return out.length === prev.length && out.every((v, i) => v === prev[i]) ? prev : out;
};

const reuseObject = (prev: record.Unknown, next: record.Unknown): record.Unknown => {
  const keys = Object.keys(next);
  const out: record.Unknown = {};
  keys.forEach((key) => (out[key] = reuse(prev[key], next[key])));
  const same =
    keys.length === Object.keys(prev).length &&
    keys.every((key) => key in prev && out[key] === prev[key]);
  return same ? prev : out;
};

/**
 * @returns next, with each part that equals the matching part of prev replaced by that
 * part of prev, so an unchanged part keeps its identity. Plain objects compare field by
 * field. Lists compare entry by entry, matching entries by key when every entry of both
 * lists has a string key, and by position otherwise. Other values compare with
 * {@link equal}.
 */
export const reuse = <T>(prev: unknown, next: T): T => {
  if (prev === next) return next;
  if (Array.isArray(prev) && Array.isArray(next)) return reuseList(prev, next) as T;
  if (narrow.isPlainObject(prev) && narrow.isPlainObject(next))
    return reuseObject(prev, next) as T;
  return (equal(prev, next) ? prev : next) as T;
};
