// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { narrow } from "@/narrow";
import { type record } from "@/record";

/** Whether the value is a list of plain objects that each have a string key. */
export const isKeyed = (value: unknown): value is record.Keyed<string>[] =>
  Array.isArray(value) &&
  value.every((v) => narrow.isPlainObject(v) && typeof v.key === "string");

export interface Remove {
  <V>(prev: V[], deletes: V | V[]): V[];
  <V>(prev: V[] | undefined, deletes: V | V[]): V[] | undefined;
}

export const remove = (<V>(
  prev: V[] | undefined,
  deletes: V | V[],
): V[] | undefined => {
  if (prev == null) return undefined;
  if (Array.isArray(deletes)) return prev.filter((v) => !deletes.includes(v));
  return prev.filter((v) => v !== deletes);
}) as Remove;

export interface RemoveKeyed {
  <K extends record.Key, E extends record.Keyed<K>>(prev: E[], keys: K | K[]): E[];
  <K extends record.Key, E extends record.Keyed<K>>(
    prev: E[] | undefined,
    keys: K | K[],
  ): E[] | undefined;
}

export const removeKeyed = (<K extends record.Key, E extends record.Keyed<K>>(
  prev: E[] | undefined,
  key: K | K[],
): E[] | undefined => {
  if (prev == null) return undefined;
  if (Array.isArray(key)) return prev.filter((i) => !key.includes(i.key));
  return prev.filter((i) => i.key !== key);
}) as RemoveKeyed;
