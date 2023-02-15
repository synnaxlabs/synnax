// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export interface KV<K = string, V = string, WK = K, WV = V, D = K>
  extends KVReader<K, V>,
    KVWriter<WK, WV>,
    KVDeleter<D> {}

export interface KVReader<K = string, V = string> {
  /** @returns the value for a given key, or null if the key is not present. */
  get: (key: K) => V | null;
}

export interface KVWriter<K = string, V = string> {
  /** Sets a key-value pair in the store. */
  set: (key: K, value: V) => void;
}

export interface KVDeleter<K = string> {
  /** Deletes a key-value pair from the store. */
  delete: (key: K) => void;
}

/** A read-writable key-value store. */
export interface AsyncKV<K = string, V = string, WK = K, WV = V, D = K>
  extends AsyncKVReader<K, V>,
    AsyncKVWriter<WK, WV>,
    AsyncKVDeleter<D> {}

/** A readable key-value store. */
export interface AsyncKVReader<K = string, V = string> {
  /** Get the value for a given key. */
  get: (key: K) => Promise<V | null>;
}

/** A writable key-value store. */
export interface AsyncKVWriter<K = string, V = string> {
  /** Sets a key-value pair in the store. The value must be serializable. */
  set: (key: K, value: V) => Promise<void>;
}

/** A key-value store that can delete key-value pairs. */
export interface AsyncKVDeleter<K = string> {
  /** Deletes a key-value pair from the store. */
  delete: (key: K) => Promise<void>;
}
