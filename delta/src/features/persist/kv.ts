// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { EncoderDecoder } from "@synnaxlabs/freighter";
import { invoke } from "@tauri-apps/api";

/** A read-writable key-value store. */
export interface KV extends KVReader, KVWriter {}

/** A readable key-value store. */
export interface KVReader {
  /** Get returns the value for a given key. */
  get: <V>(key: string) => Promise<V | undefined>;
}

/** A writable key-value store. */
export interface KVWriter {
  /** Sets a key-value pair in the store. The value must be serializable. */
  set: <V>(key: string, value: V) => Promise<void>;
  /** Deletes a key-value pair from the store. */
  delete: (key: string) => Promise<void>;
}

enum KVCommand {
  Get = "get",
  Set = "set",
  Delete = "delete",
}

interface KVResponse {
  value: string;
  error: string;
}

interface KVRequest {
  command: KVCommand;
  key: string;
  value: string;
}

/** TauriKV communicates with a rust key-value store running on the backend. */
export class TauriKV implements KV {
  private readonly ecd: EncoderDecoder;

  constructor(ecd: EncoderDecoder) {
    this.ecd = ecd;
  }

  async get<V>(key: string): Promise<V | undefined> {
    try {
      return await this.exec({ command: KVCommand.Get, key, value: "" });
    } catch (err) {
      if ((err as Error).message === "Key not found") return undefined;
      throw err;
    }
  }

  async set<V>(key: string, value: V): Promise<void> {
    return await this.exec({
      command: KVCommand.Set,
      key,
      value: new TextDecoder().decode(this.ecd.encode(value)),
    });
  }

  async delete(key: string): Promise<void> {
    return await this.exec({ command: KVCommand.Delete, key, value: "" });
  }

  private async exec<V = null>(request: KVRequest): Promise<V | undefined> {
    const res: KVResponse = await invoke("kv_exec", { request });
    if (res.error.length > 0) throw new Error(res.error);
    if (res.value.length === 0) return undefined;
    const buf = new TextEncoder().encode(res.value);
    return await this.ecd.decode(buf);
  }
}
