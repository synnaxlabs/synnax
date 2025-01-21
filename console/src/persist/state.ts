// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action, type Middleware } from "@reduxjs/toolkit";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { debounce, deep, TimeSpan, type UnknownRecord } from "@synnaxlabs/x";
import { getCurrentWindow } from "@tauri-apps/api/window";

import { openTauriKV, type SugaredKV } from "@/persist/kv";
import { type Version } from "@/version";

export const PERSISTED_STATE_KEY = "console-persisted-state";
export const DB_VERSION_KEY = "console-version";

export const V1_STORE_PATH = "~/.synnax/console/persisted-state.dat";
export const V2_STORE_PATH = "persisted-state.json";

interface StateVersionValue {
  version: number;
}

export interface RequiredState extends Version.StoreState {}

export type KVOpener = (base: string) => Promise<SugaredKV>;

const openAndMigrateKV = async (openKV: KVOpener = openTauriKV): Promise<SugaredKV> => {
  // Open V2 store and check its length. If it's greater than 0, return it.
  // Otherwise, open V1 store and return it.
  const v2Store = await openKV(V2_STORE_PATH);
  if ((await v2Store.length()) > 0) return v2Store;
  const v1Store = await openKV(V1_STORE_PATH);
  // If it's empty, we can just return the V2 store.
  if ((await v1Store.length()) === 0) return v2Store;
  // Otherwise, we need to migrate the V1 store to V2. Get the DB version
  // key and use it to get the state.
  const v1Version = await v1Store.get<StateVersionValue>(DB_VERSION_KEY);
  if (v1Version == null) return v2Store;
  const v1State = await v1Store.get<RequiredState>(
    persistedStateKey(v1Version.version),
  );
  // We no longer need the V1 store, so we can clear it out.
  await v1Store.clear();
  if (v1State == null) return v2Store;
  // Make sure we normalize the version number in case it is something massive.
  const version = nextVersion(v1Version.version);
  await v2Store.set(persistedStateKey(version), v1State);
  await v2Store.set(DB_VERSION_KEY, { version });
  return v2Store;
};

export interface Config<S extends RequiredState> {
  migrator?: (state: S) => S;
  initial: S;
  exclude?: Array<deep.Key<S> | ((func: S) => S)>;
  openKV?: KVOpener;
  historyLength?: number;
}

export const REVERT_STATE: Action = { type: "persist.revert-state" };
export const CLEAR_STATE: Action = { type: "persist.clear-state" };

export const persistedStateKey = (version: number): string =>
  `${PERSISTED_STATE_KEY}.${version}`;

const KEEP_HISTORY = 4;

const nextVersion = (currentVersion: number): number =>
  (currentVersion + 1) % KEEP_HISTORY;

/**
 * Clear the entire store and reload the page.
 */
export const hardClearAndReload = () => {
  const appWindow = getCurrentWindow();
  if (appWindow == null || appWindow.label !== MAIN_WINDOW) return;
  openAndMigrateKV()
    .then(async (db) => await db.clear())
    .finally(window.location.reload);
};

interface Engine<S extends RequiredState> {
  /** Revert reverts to the previous state. */
  revert(): Promise<void>;
  /** Clear clears the entire store. */
  clear(): Promise<void>;
  /** Persist the provided state to disk. */
  persist(state: S): Promise<void>;
  /** The initial state that is persisted to disk. Loaded from disk on engine creation. */
  initialState?: S;
}

/**
 * Open a new persistence engine instance with the provided configuration. This is used
 * to persist redux state to disk. It is kept independently of the middleware implementation
 * for easy testing.
 * @param config - The configuration for the engine.
 * @returns A new engine instance.
 */
export const open = async <S extends RequiredState>({
  exclude = [],
  initial,
  migrator,
  openKV,
}: Config<S>): Promise<Engine<S>> => {
  const db = await openAndMigrateKV(openKV);
  const kvVersion = await db.get<StateVersionValue>(DB_VERSION_KEY);
  let version: number = kvVersion?.version ?? 0;
  if (kvVersion == null) await db.set(DB_VERSION_KEY, { version });

  const revert = async (): Promise<void> => {
    version = (version - 1 + KEEP_HISTORY) % KEEP_HISTORY;
    await db.set(DB_VERSION_KEY, { version });
  };

  const clear = async (): Promise<void> => {
    await db.clear();
    version = 0;
    await db.set(DB_VERSION_KEY, { version });
  };

  const persist = async (state: S) => {
    version = nextVersion(version);
    let deepCopy = deep.copy(state);
    exclude.forEach((key) => {
      if (typeof key === "function") deepCopy = key(deepCopy);
      // @ts-expect-error - we know this is a key
      else deepCopy = deep.deleteD(deepCopy, key);
    });
    await db.set(persistedStateKey(version), deepCopy).catch(console.error);
    await db.set(DB_VERSION_KEY, { version }).catch(console.error);
  };

  let state = await db.get<S>(persistedStateKey(version));
  // If we have migrations, apply them.
  if (state != null && migrator != null)
    try {
      state = migrator(state);
      // Immediately persist the migrated state.
      await persist(state);
    } catch (e) {
      console.error("unable to apply migrations. continuing with initial state.");
      console.error(e);
      state = null;
    }
  state ??= initial;

  // Override defaults for key-value pairs that should be excluded from state.
  if (state != null)
    exclude.forEach((key) => {
      if (typeof key === "function") return;
      const v = deep.get(initial, key, { optional: true });
      if (v == null) return;
      deep.set(state, key, v);
    });

  return { revert, clear, persist, initialState: state };
};

const passThroughMiddleware: Middleware<UnknownRecord, any> =
  () => (next) => (action) =>
    next(action);

const PERSIST_DEBOUNCE = TimeSpan.milliseconds(250);

/**
 * Creates a middleware that persists the redux store state to the provided persistence
 * engine after an action is dispatched.
 *
 * @param engine - The persistence engine to store data in.
 * @param debounceInterval - The interval to debounce persistence operations by. Defaults
 * to 250ms.
 */
export const middleware = <S extends RequiredState>(
  engine: Engine<S>,
  debounceInterval: TimeSpan = PERSIST_DEBOUNCE,
): Middleware<UnknownRecord> => {
  const appWindow = getCurrentWindow();
  if (appWindow.label !== MAIN_WINDOW) return passThroughMiddleware;
  const debouncedPersist = debounce(engine.persist, debounceInterval.milliseconds);
  return (store) => (next) => (action) => {
    const result = next(action);
    const type = (action as Action | undefined)?.type;
    if (type === REVERT_STATE.type)
      engine
        .revert()
        .then(() => window.location.reload())
        .catch(console.error);
    else if (type === CLEAR_STATE.type) engine.clear().catch(console.error);
    else debouncedPersist(store.getState());
    return result;
  };
};
