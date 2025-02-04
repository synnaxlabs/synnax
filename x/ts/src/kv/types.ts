// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export interface Reader<V = unknown> {
  /** @returns the value for a given key, or null if the key is not present. */
  get: (key: string) => V | null;
}

export interface Writer<V = unknown> {
  /** Sets a key-value pair in the store. */
  set: (key: string, value: V) => void;
}

export interface Deleter {
  /** Deletes a key-value pair from the store. */
  delete: (key: string) => void;
}

export interface KV<R = unknown, W = R> extends Reader<R>, Writer<W>, Deleter {}

/** A readable key-value store. */
export interface AsyncReader<V = unknown> {
  /** Get the value for a given key. */
  get: (key: string) => Promise<V | null>;
}

/** A writable key-value store. */
export interface AsyncWriter<V = unknown> {
  /** Sets a key-value pair in the store. The value must be serializable. */
  set: (key: string, value: V) => Promise<void>;
}

/** A key-value store that can delete key-value pairs. */
export interface AsyncDeleter {
  /** Deletes a key-value pair from the store. */
  delete: (key: string) => Promise<void>;
}

/** A read-writable key-value store. */
export interface Async<R = unknown, W = R>
  extends AsyncReader<R>,
    AsyncWriter<W>,
    AsyncDeleter {}

export const stringPairZ = z.object({ key: z.string(), value: z.string() });

/** A general purpose key-value pair. */
export interface Pair<V = unknown> {
  key: string;
  value: V;
}
