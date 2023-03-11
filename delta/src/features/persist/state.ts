// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Middleware } from "@reduxjs/toolkit";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { Deep, DeepKey } from "@synnaxlabs/x";
import { getVersion } from "@tauri-apps/api/app";
import { appWindow } from "@tauri-apps/api/window";

import { VersionStoreState } from "../version";

import { TauriKV } from "./kv";

const PERSISTED_STATE_KEY = "delta-persisted-state";

export interface RequiredState extends VersionStoreState {}

export interface PersistConfig<S extends RequiredState> {
  exclude: Array<DeepKey<S>>;
}

export const openPersist = async <S extends RequiredState>({
  exclude = [],
}: PersistConfig<S>): Promise<[S | undefined, Middleware<{}, S>]> => {
  if (appWindow.label !== MAIN_WINDOW) return [undefined, noOpMiddleware];
  const db = new TauriKV<S>();
  await db.openAck();
  let state = (await db.get(PERSISTED_STATE_KEY)) ?? undefined;
  if (state != null) state = await reconcileVersions(state);

  return [
    state,
    (store) => (next) => (action) => {
      const result = next(action);
      if (appWindow.label !== MAIN_WINDOW) return result;
      // We need to make a deep copy here to make immer happy
      // when we do exclusions.
      const deepCopy = Deep.copy(store.getState());
      const filtered = Deep.delete<S>(deepCopy, ...exclude);
      void db.set(PERSISTED_STATE_KEY, filtered);
      return result;
    },
  ];
};

const noOpMiddleware: Middleware<{}, any> = () => (next) => (action) => next(action);

const reconcileVersions = async <S extends RequiredState>(
  state: S
): Promise<S | undefined> => {
  const storedVersion = state.version.version;
  const tauriVersion = await getVersion();
  return storedVersion === tauriVersion ? state : undefined;
};
