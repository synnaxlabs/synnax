// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LazyStore } from "@tauri-apps/plugin-store";
import { z } from "zod";

import { Runtime } from "@/session/runtime";

/**
 * The store the 0.56 and earlier Console wrote. Relative, so it resolves against the
 * roaming app data directory where 0.56 left it. Opened read-only, never written.
 */
export const STORE_PATH = "persisted-state.json";

const SLOT_KEY = "console-version";
const STATE_KEY = "console-persisted-state";

const slotZ = z.object({ version: z.number().int() });

export interface Reader {
  (key: string): Promise<unknown>;
}

const tauriReader = (): Reader => {
  const store = new LazyStore(STORE_PATH, { autoSave: false });
  return async (key) => (await store.get(key)) ?? null;
};

// The pre-IndexedDB browser build namespaced localStorage by the store's file name.
const localStorageReader = (): Reader => {
  const read = (key: string): unknown => {
    const item = localStorage.getItem(`${STORE_PATH}:${key}`);
    return item == null ? null : JSON.parse(item);
  };
  return async (key) => read(key);
};

export const openReader = (): Reader =>
  Runtime.ENGINE === "tauri" ? tauriReader() : localStorageReader();

/**
 * Reads the whole state blob the previous release last committed, or null when it left
 * none behind.
 * @param read - Reader over the legacy store.
 */
export const readState = async (read: Reader): Promise<unknown> => {
  const slot = slotZ.safeParse(await read(SLOT_KEY));
  if (!slot.success) return null;
  return await read(`${STATE_KEY}.${slot.data.version}`);
};
