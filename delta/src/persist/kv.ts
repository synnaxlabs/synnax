// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type kv } from "@synnaxlabs/x";
import { invoke } from "@tauri-apps/api";
import { type Event } from "@tauri-apps/api/event";
import { appWindow } from "@tauri-apps/api/window";

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

interface KVOpenResult {
  message: string;
}

export const multipleWindowsOpen = new Error("[persist] - windows open");

/**
 * TauriKV an implementation of AsyncKV that communicates with a rust key-value
 * store running on the backend.
 */
export class TauriKV<V> implements kv.Async<string, V> {
  isOpen: boolean;

  constructor() {
    this.isOpen = false;
  }

  async openAck(): Promise<void> {
    if (this.isOpen) return;
    return await new Promise((resolve, reject) => {
      void (async () => {
        await appWindow.listen("kv_open_res", (event: Event<KVOpenResult>) => {
          const { message } = event.payload;
          if (message !== "") reject(multipleWindowsOpen);
          this.isOpen = true;
          resolve();
        });
        await appWindow.emit("kv_open_req");
      })();
    });
  }

  async get(key: string): Promise<V | null> {
    try {
      return await this.exec({ command: KVCommand.Get, key, value: "" });
    } catch (err) {
      if ((err as Error).message === "Key not found") return null;
      throw err;
    }
  }

  async set(key: string, value: V): Promise<void> {
    await this.exec({
      command: KVCommand.Set,
      key,
      value: JSON.stringify(value),
    });
  }

  async delete(key: string): Promise<void> {
    await this.exec({ command: KVCommand.Delete, key, value: "" });
  }

  private async exec<V = null>(request: KVRequest): Promise<V | null> {
    const res: KVResponse = await invoke("kv_exec", { request });
    if (res.error.length > 0) throw new Error(res.error);
    if (res.value.length === 0) return null;
    return JSON.parse(res.value);
  }
}
