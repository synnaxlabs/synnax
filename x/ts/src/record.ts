// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export type Key = string | number;

export type UnknownRecord = Record<Key, unknown>;

export interface Keyed<K extends Key> {
  key: K;
}

export const unknownRecordZ = z.record(
  z.union([z.number(), z.string(), z.symbol()]),
  z.unknown(),
);

export type Entries<T> = Array<
  {
    [K in keyof T]: [K, T[K]];
  }[keyof T]
>;

export const getEntries = <T extends Record<Key, unknown>>(obj: T): Entries<T> =>
  Object.entries(obj) as Entries<T>;
