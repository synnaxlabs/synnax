// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export interface KV<V = string> extends Reader<V>, Writer<V>, Deleter {}

export interface Reader<V = string> {
  /** @returns the value for a given key, or null if the key is not present. */
  get: <IV = V>(key: string) => IV | null;
}

export interface Writer<V = string> {
  /** Sets a key-value pair in the store. */
  set: <IV = V>(key: string, value: IV) => void;
}

export interface Deleter {
  /** Deletes a key-value pair from the store. */
  delete: (key: string) => void;
}

/** A read-writable key-value store. */
export interface Async<V = string>
  extends AsyncReader<V>,
    AsyncWriter<V>,
    AsyncDeleter {}

/** A readable key-value store. */
export interface AsyncReader<V = string> {
  /** Get the value for a given key. */
  get: <IV = V>(key: string) => Promise<IV | null>;
}

/** A writable key-value store. */
export interface AsyncWriter<V = string> {
  /** Sets a key-value pair in the store. The value must be serializable. */
  set: <IV = V>(key: string, value: IV) => Promise<void>;
}

/** A key-value store that can delete key-value pairs. */
export interface AsyncDeleter {
  /** Deletes a key-value pair from the store. */
  delete: (key: string) => Promise<void>;
}

export const stringPairZ = z.object({
  key: z.string(),
  value: z.string(),
});

/** A general purpose key-value pair. */
export interface Pair<V = string> {
  key: string;
  value: V;
}
