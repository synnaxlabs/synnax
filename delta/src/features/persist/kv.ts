// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { invoke } from "@tauri-apps/api";

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
      value: JSON.stringify(value),
    });
  }

  async delete(key: string): Promise<void> {
    return await this.exec({ command: KVCommand.Delete, key, value: "" });
  }

  private async exec<V = null>(request: KVRequest): Promise<V | undefined> {
    const res: KVResponse = await invoke("kv_exec", { request });
    if (res.error.length > 0) throw new Error(res.error);
    if (res.value.length === 0) return undefined;
    return JSON.parse(res.value);
  }
}
