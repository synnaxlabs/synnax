import {
  DRIFT_SLICE_NAME,
  MAIN_WINDOW,
  initialState as driftInitialState,
} from "@synnaxlabs/drift";
import { appWindow } from "@tauri-apps/api/window";

import { KV } from "./kv";

const PERSISTED_STATE_KEY = "delta-persisted-state";

/**
 * Returns a function that preloads the state from the given key-value store on the main
 * window.
 *
 * @param db - the key-value store to load the state from.
 * @returns a redux middleware.
 */
export const newPreloadState =
  (db: KV) =>
  async <S extends Record<string, unknown> & { drift: unknown }>(): Promise<
    S | undefined
  > => {
    if (appWindow.label !== MAIN_WINDOW) return undefined;
    const state = await db.get<S>(PERSISTED_STATE_KEY);
    if (state == null) return undefined;
    // TODO: (@emilbon99) drift doesn't inspect initial state and fork windows accordinly
    // so we need to manually set the drift state for now.
    if (DRIFT_SLICE_NAME in state) state.drift = driftInitialState;
    return state;
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
