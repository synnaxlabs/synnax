// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type UnknownAction,
  type Dispatch,
  type Middleware,
  type MiddlewareAPI,
} from "@reduxjs/toolkit";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { debounce, deep, type UnknownRecord } from "@synnaxlabs/x";
import { getVersion } from "@tauri-apps/api/app";
import { appWindow } from "@tauri-apps/api/window";

import { TauriKV } from "@/persist/kv";
import { type Version } from "@/version";

const PERSISTED_STATE_KEY = "console-persisted-state";

export interface RequiredState extends Version.StoreState {}

export interface Config<S extends RequiredState> {
  exclude: Array<deep.Key<S>>;
}

export const open = async <S extends RequiredState>({
  exclude = [],
}: Config<S>): Promise<[S | undefined, Middleware<UnknownRecord, S>]> => {
  if (appWindow.label !== MAIN_WINDOW) return [undefined, noOpMiddleware];
  const db = new TauriKV<S>();
  await db.openAck();
  let state = (await db.get(PERSISTED_STATE_KEY)) ?? undefined;
  if (state != null) state = await reconcileVersions(state);

  const persist = debounce((store: MiddlewareAPI<Dispatch<UnknownAction>, S>) => {
    if (appWindow.label !== MAIN_WINDOW) return;
    // We need to make a deep copy here to make immer happy
    // when we do exclusions.
    const deepCopy = deep.copy(store.getState());
    const filtered = deep.deleteD<S>(deepCopy, ...exclude);
    db.set(PERSISTED_STATE_KEY, filtered).catch(console.error);
  }, 500);

  return [
    state,
    (store) => (next) => (action) => {
      const result = next(action);
      persist(store);
      return result;
    },
  ];
};

const noOpMiddleware: Middleware<UnknownRecord, any> = () => (next) => (action) =>
  next(action);

const reconcileVersions = async <S extends RequiredState>(
  state: S,
): Promise<S | undefined> => {
  const storedVersion = state.version.version;
  const tauriVersion = await getVersion();
  if (storedVersion !== tauriVersion) return undefined;
  return state;
};
