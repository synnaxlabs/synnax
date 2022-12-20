import {
  DRIFT_SLICE_NAME,
  MAIN_WINDOW,
  initialState as driftInitialState,
} from "@synnaxlabs/drift";
import { appWindow } from "@tauri-apps/api/window";

import { KV } from "./kv";

const PERSISTED_STATE_KEY = "persisted_state";

export const createPreloadedState =
  (db: KV) =>
  async <S extends Record<string, unknown> & { drift: unknown }>(): Promise<
    S | undefined
  > => {
    if (appWindow.label !== MAIN_WINDOW) return undefined;
    const state = await db.get<S>(PERSISTED_STATE_KEY);
    if (state == null) return undefined;
    if (DRIFT_SLICE_NAME in state) state.drift = driftInitialState;
    return state;
  };

export const createPersistStateMiddleware =
  (db: KV) => (store: any) => (next: any) => (action: any) => {
    const result = next(action);
    if (appWindow.label !== MAIN_WINDOW) return result;
    void db.set(PERSISTED_STATE_KEY, store.getState());
    return result;
  };
