// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createAction, type Middleware, type MiddlewareAPI } from "@reduxjs/toolkit";
import {
  type CrudeTimeSpan,
  debounce,
  deep,
  type record,
  TimeSpan,
} from "@synnaxlabs/x";

import { openSugaredKV, type SugaredKV } from "@/session/persist/kv";
import { Runtime } from "@/session/runtime";

// Note that this is a relative path within the tauri standard app data directory.
// On macOS, this is ~/Library/Application Support/com.synnaxlabs.dev.
// On Windows, this is %APPDATA%/com.synnaxlabs.dev.
export const STORE_PATH = "persisted-state.json";

/**
 * The scope state is persisted under. State partitions into three levels: L0
 * is global, L1 is per-cluster, and L2 is per-cluster-per-project.
 */
export interface Context {
  cluster?: string;
  project?: string;
}

export const contextsEqual = (a: Context, b: Context): boolean =>
  a.cluster === b.cluster && a.project === b.project;

interface StateVersionValue {
  version: number;
}

export interface KVOpener {
  (base: string): SugaredKV;
}

type ExcludeFn<S extends object> = (state: S) => S;

const isExcludeFn = <S extends object>(
  exclude: deep.Key<S> | ExcludeFn<S>,
): exclude is ExcludeFn<S> => typeof exclude === "function";

type SliceKey<S extends object> = keyof S & string;

export type SliceMigrators<S extends object> = {
  [K in keyof S]?: (raw: unknown) => S[K];
};

/** Slice names belonging to each partition level. Every persisted slice
 * appears in exactly one level. */
export interface Levels<S extends object> {
  l0: Array<SliceKey<S>>;
  l1: Array<SliceKey<S>>;
  l2: Array<SliceKey<S>>;
}

export interface Config<S extends object> {
  initial: S;
  levels: Levels<S>;
  /** getContext reads the partition scope from state. */
  getContext: (state: S) => Context;
  /** Per-slice migrators applied to each slice as its partition loads. */
  migrators?: SliceMigrators<S>;
  exclude?: Array<deep.Key<S> | ExcludeFn<S>>;
  openKV?: KVOpener;
  historyLength?: number;
}

export const revertState = createAction("persist/revertState");
export const clearState = createAction("persist/clearState");
/**
 * Replaces the swapped slices wholesale when the session context changes.
 * Dispatched by the persistence middleware, applied by a root reducer wrapper.
 */
export const hydrate = createAction<record.Unknown>("persist/hydrate");
export type Action = ReturnType<
  typeof revertState | typeof clearState | typeof hydrate
>;

export const l0PartitionBase = (): string => "l0";
export const l1PartitionBase = (cluster: string): string => `l1.${cluster}`;
export const l2PartitionBase = (cluster: string, project: string): string =>
  `l2.${cluster}.${project}`;

export const partitionStateKey = (base: string, version: number): string =>
  `${base}.${version}`;
export const partitionVersionKey = (base: string): string => `${base}.version`;

const DEFAULT_HISTORY_LENGTH = 4;

/**
 * Clear the entire store and reload the page.
 */
export const hardClearAndReload = () => {
  if (!Runtime.isMainWindow()) return;
  openSugaredKV(STORE_PATH)
    .clear()
    .finally(() => window.location.reload())
    .catch((err: unknown) => {
      console.error("failed to clear store during hard reload", err);
    });
};

export interface Engine<S extends object> {
  /** The context the initial state was composed for. */
  context: Context;
  /** The composed L0 + L1 + L2 state loaded on engine creation. */
  initialState?: S;
  /** Persist the state's partitions under the given context's keys. */
  persist: (state: S, context: Context) => Promise<void>;
  /**
   * Load the L1/L2 (or L2-only) slices for the target context, defaulting
   * unvisited partitions to their initial slices. When includeL1 is true the
   * target project is re-derived from the loaded L1 partition.
   */
  loadSwap: (state: S, context: Context, includeL1: boolean) => Promise<Partial<S>>;
  /** Step every active partition back one version. */
  revert: (context: Context) => Promise<void>;
  /** Clear the entire store. */
  clear: () => Promise<void>;
}

/**
 * Open a new persistence engine instance with the provided configuration. This is used
 * to persist the Redux store state to disk. It's kept independently of the middleware
 * implementation for easy testing.
 * @param config - The configuration for the engine.
 * @returns A new engine instance.
 */
export const open = async <S extends object>(config: Config<S>): Promise<Engine<S>> => {
  const {
    initial,
    levels,
    getContext,
    migrators,
    exclude = [],
    openKV = openSugaredKV,
    historyLength = DEFAULT_HISTORY_LENGTH,
  } = config;
  const db = openKV(STORE_PATH);
  const copiedInitial = deep.copy(initial);
  const versions = new Map<string, number>();

  const nextVersion = (v: number): number => (v + 1) % historyLength;
  const prevVersion = (v: number): number => (v - 1 + historyLength) % historyLength;

  const readVersion = async (base: string): Promise<number> => {
    const cached = versions.get(base);
    if (cached != null) return cached;
    const stored = (await db.get(
      partitionVersionKey(base),
    )) as StateVersionValue | null;
    const version = stored?.version ?? 0;
    versions.set(base, version);
    return version;
  };

  const setVersion = async (base: string, version: number): Promise<void> => {
    versions.set(base, version);
    await db.set(partitionVersionKey(base), { version }).catch((err: unknown) => {
      console.error(`failed to bump version of partition ${base}`, err);
    });
  };

  const migrateSlice = (key: SliceKey<S>, raw: unknown): unknown => {
    const migrator = migrators?.[key];
    if (migrator == null || raw == null) return raw;
    try {
      return migrator(raw);
    } catch (err) {
      console.error(`failed to migrate slice ${key}. using its initial state.`, err);
      return null;
    }
  };

  const readPartition = async (
    slices: Array<SliceKey<S>>,
    base: string,
  ): Promise<Partial<S>> => {
    const version = await readVersion(base);
    const data = (await db.get(partitionStateKey(base, version))) as Partial<S> | null;
    const out: Partial<S> = {};
    if (data == null) return out;
    slices.forEach((key) => {
      const migrated = migrateSlice(key, data[key]);
      if (migrated != null) out[key] = migrated as S[typeof key];
    });
    return out;
  };

  const writePartition = async (
    state: S,
    slices: Array<SliceKey<S>>,
    base: string,
  ): Promise<void> => {
    const version = nextVersion(await readVersion(base));
    const data: Partial<S> = {};
    slices.forEach((key) => {
      data[key] = state[key];
    });
    await db.set(partitionStateKey(base, version), data).catch((err: unknown) => {
      console.error(`failed to write partition ${base} at version ${version}`, err);
    });
    await setVersion(base, version);
  };

  // Excluded deep keys are removed at write time, so restore their defaults on
  // every load.
  const resetExcluded = (state: S): S => {
    exclude.forEach((key) => {
      if (isExcludeFn(key)) return;
      const v = deep.get(copiedInitial, key, { optional: true });
      if (v == null) return;
      deep.set(state, key, v);
    });
    return state;
  };

  const persist = async (rawState: S, context: Context): Promise<void> => {
    let state = deep.copy(rawState);
    exclude.forEach((key) => {
      if (isExcludeFn(key)) state = key(state);
      else deep.remove(state, key);
    });
    await writePartition(state, levels.l0, l0PartitionBase());
    if (context.cluster == null) return;
    await writePartition(state, levels.l1, l1PartitionBase(context.cluster));
    if (context.project == null) return;
    await writePartition(
      state,
      levels.l2,
      l2PartitionBase(context.cluster, context.project),
    );
  };

  const fill = (
    out: Partial<S>,
    slices: Array<SliceKey<S>>,
    data: Partial<S>,
  ): void => {
    slices.forEach((key) => {
      out[key] = data[key] ?? deep.copy(copiedInitial[key]);
    });
  };

  const loadSwap = async (
    state: S,
    context: Context,
    includeL1: boolean,
  ): Promise<Partial<S>> => {
    const out: Partial<S> = {};
    let project = context.project;
    if (includeL1) {
      const l1 =
        context.cluster == null
          ? {}
          : await readPartition(levels.l1, l1PartitionBase(context.cluster));
      fill(out, levels.l1, l1);
      // The target cluster's L1 records which of its projects was last active.
      project = getContext({ ...state, ...out }).project;
    }
    const l2 =
      context.cluster == null || project == null
        ? {}
        : await readPartition(levels.l2, l2PartitionBase(context.cluster, project));
    fill(out, levels.l2, l2);
    const composed = resetExcluded({ ...state, ...out });
    [...(includeL1 ? levels.l1 : []), ...levels.l2].forEach((key) => {
      out[key] = composed[key];
    });
    return out;
  };

  const revert = async (context: Context): Promise<void> => {
    const bases = [l0PartitionBase()];
    if (context.cluster != null) {
      bases.push(l1PartitionBase(context.cluster));
      if (context.project != null)
        bases.push(l2PartitionBase(context.cluster, context.project));
    }
    for (const base of bases)
      await setVersion(base, prevVersion(await readVersion(base)));
  };

  const clear = async (): Promise<void> => {
    await db.clear();
    versions.clear();
  };

  // Startup composition: L0 names the selected cluster, whose L1 names the
  // active project, whose L2 holds the workspace.
  const state = deep.copy(initial);
  Object.assign(state, await readPartition(levels.l0, l0PartitionBase()));
  const { cluster } = getContext(state);
  if (cluster != null)
    Object.assign(state, await readPartition(levels.l1, l1PartitionBase(cluster)));
  const context = getContext(state);
  if (context.cluster != null && context.project != null)
    Object.assign(
      state,
      await readPartition(levels.l2, l2PartitionBase(context.cluster, context.project)),
    );
  resetExcluded(state);

  return { context, initialState: state, persist, loadSwap, revert, clear };
};

const PERSIST_DEBOUNCE = TimeSpan.milliseconds(250);

export interface MiddlewareConfig<S extends object> {
  engine: Engine<S>;
  getContext: (state: S) => Context;
  /**
   * Adjusts the loaded slices before they hydrate on a context switch. Used to
   * turn state that must not be merged wholesale (drift windows) into
   * dispatches instead.
   */
  onSwap?: (loaded: Partial<S>, store: MiddlewareAPI) => Partial<S>;
  debounceInterval?: CrudeTimeSpan;
}

/**
 * Creates a middleware that persists the redux store state to the provided
 * persistence engine after an action is dispatched, and swaps the L1/L2
 * partitions when the (cluster, project) context changes: the outgoing
 * context flushes under its own keys, the target's partitions load, and a
 * single hydrate action applies them.
 */
export const middleware = <S extends object>({
  engine,
  getContext,
  onSwap,
  debounceInterval = PERSIST_DEBOUNCE,
}: MiddlewareConfig<S>): Middleware<record.Unknown> => {
  let current = engine.context;
  let swapping = false;
  // Concurrent switches race: a slow stale swap hydrating last would clobber
  // the newer context's slices, so only the latest generation may hydrate.
  let swapGen = 0;
  return (store) => {
    const debouncedPersist = debounce(() => {
      if (swapping) return;
      engine
        .persist(store.getState() as S, current)
        .catch((e: unknown) => console.error("Failed to persist state", e));
    }, debounceInterval);
    return (next) => (action) => {
      const result = next(action);
      const type = (action as Action | undefined)?.type;
      const state = store.getState() as S;
      if (type === revertState.type)
        engine
          .revert(current)
          .then(() => window.location.reload())
          .catch((err: unknown) => {
            console.error("failed to revert state", err);
          });
      else if (type === clearState.type)
        engine
          .clear()
          .then(() => window.location.reload())
          .catch((err: unknown) => {
            console.error("failed to clear state", err);
          });
      else if (type === hydrate.type) {
        current = getContext(state);
        swapping = false;
        void debouncedPersist();
      } else {
        const ctx = getContext(state);
        if (!contextsEqual(ctx, current)) {
          const old = current;
          const includeL1 = ctx.cluster !== old.cluster;
          current = ctx;
          swapping = true;
          const gen = ++swapGen;
          engine
            .persist(state, old)
            .then(async () => {
              if (gen !== swapGen) return;
              const loaded = await engine.loadSwap(state, ctx, includeL1);
              if (gen !== swapGen) return;
              store.dispatch(
                hydrate((onSwap?.(loaded, store) ?? loaded) as record.Unknown),
              );
            })
            .catch((err: unknown) => {
              if (gen === swapGen) swapping = false;
              console.error("failed to swap session context", err);
            });
        } else void debouncedPersist();
      }
      return result;
    };
  };
};
