// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DRIFT_SLICE_NAME,
  MAIN_WINDOW,
  initialState as driftInitialState,
} from "@synnaxlabs/drift";
import { getVersion } from "@tauri-apps/api/app";
import { appWindow } from "@tauri-apps/api/window";

import { VersionState } from "../version";

import { KV } from "./kv";

const PERSISTED_STATE_KEY = "delta-persisted-state";

export interface RequiredState extends VersionState {
  drift: unknown;
}

/**
 * Returns a function that preloads the state from the given key-value store on the main
 * window.
 *
 * @param db - the key-value store to load the state from.
 * @returns a redux middleware.
 */
export const newPreloadState =
  (db: KV) =>
  async <S extends RequiredState>(): Promise<S | undefined> => {
    if (appWindow.label !== MAIN_WINDOW) return undefined;
    const state = await db.get<S>(PERSISTED_STATE_KEY);
    if (state == null) return undefined;
    // TODO: (@emilbon99) drift doesn't inspect initial state and fork windows accordinly
    // so we need to manually set the drift state for now.
    if (DRIFT_SLICE_NAME in state) state.drift = driftInitialState;
    return await reconcileVersions(state);
  };

/**
 * Returns a redux middleware that persists the state to the given key-value store on
 * the main window. NOTE: this key-value store does not encrypt sensitive data! BE CAREFUL!
 *
 * @param db - the key-value store to persist to.
 * @returns a redux middleware.
 */
export const newPersistStateMiddleware =
  (db: KV) => (store: any) => (next: any) => (action: any) => {
    const result = next(action);
    if (appWindow.label !== MAIN_WINDOW) return result;
    void db.set(PERSISTED_STATE_KEY, store.getState());
    return result;
  };

const reconcileVersions = async <S extends RequiredState>(
  state: S
): Promise<S | undefined> => {
  if (state.version !== (await getVersion())) return;
  return state;
};
