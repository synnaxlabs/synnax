// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createAction, type Middleware } from "@reduxjs/toolkit";
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
 * The scope state is persisted under. State partitions into three scopes:
 * global, per-cluster, and per-cluster-per-project.
 */
export interface Context {
  cluster?: string;
  project?: string;
}

const contextsEqual = (a: Context, b: Context): boolean =>
  a.cluster === b.cluster && a.project === b.project;

interface StateVersionValue {
  version: number;
}

export interface KVOpener {
  (base: string): SugaredKV;
}

/** Strips state that must not reach disk, returning what gets written. */
export type ExcludeFn<S extends object> = (state: S) => S;

type SliceKey<S extends object> = keyof S & string;

export type SliceMigrators<S extends object> = {
  [K in keyof S]?: (raw: unknown) => S[K];
};

/** Slice names belonging to each partition scope. Every persisted slice
 * appears in exactly one scope. */
export interface Scopes<S extends object> {
  global: Array<SliceKey<S>>;
  cluster: Array<SliceKey<S>>;
  project: Array<SliceKey<S>>;
}

export interface Config<S extends object> {
  initial: S;
  scopes: Scopes<S>;
  /** getContext reads the partition scope from state. */
  getContext: (state: S) => Context;
  /** Per-slice migrators applied to each slice as its partition loads. */
  migrators?: SliceMigrators<S>;
  exclude?: Array<ExcludeFn<S>>;
  openKV?: KVOpener;
  debounceInterval?: CrudeTimeSpan;
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

const GLOBAL_PARTITION = "global";
const clusterPartitionBase = (cluster: string): string => `cluster.${cluster}`;
const projectPartitionBase = (cluster: string, project: string): string =>
  `project.${cluster}.${project}`;

const partitionStateKey = (base: string, version: number): string =>
  `${base}.${version}`;
const partitionVersionKey = (base: string): string => `${base}.version`;

/** Versions kept per partition before the ring wraps. */
const HISTORY_LENGTH = 4;

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

/** Persists the redux store state to disk, partitioned by session context. */
class Engine<S extends object> {
  /** The context {@link initialState} was composed for. */
  context: Context;
  /** The composed global + cluster + project state read on open. */
  initialState: S;

  private readonly db: SugaredKV;
  private readonly initial: S;
  private readonly scopes: Scopes<S>;
  private readonly getContext: (state: S) => Context;
  private readonly migrators: SliceMigrators<S>;
  private readonly exclude: Array<ExcludeFn<S>>;
  private readonly versions = new Map<string, number>();

  /**
   * Opens an engine over the persisted store and composes the state it holds for the
   * context it finds.
   * @param config - The configuration for the engine.
   */
  static async open<S extends object>(config: Config<S>): Promise<Engine<S>> {
    const engine = new Engine(config);
    await engine.compose();
    return engine;
  }

  private constructor({
    initial,
    scopes,
    getContext,
    migrators = {},
    exclude = [],
    openKV = openSugaredKV,
  }: Config<S>) {
    this.initial = deep.copy(initial);
    this.scopes = scopes;
    this.getContext = getContext;
    this.migrators = migrators;
    this.exclude = exclude;
    this.db = openKV(STORE_PATH);
    this.initialState = deep.copy(initial);
    this.context = getContext(this.initialState);
  }

  /** Persist the state's partitions under the given context's keys. */
  async persist(rawState: S, context: Context): Promise<void> {
    let state = deep.copy(rawState);
    this.exclude.forEach((exclude) => (state = exclude(state)));
    await this.writePartition(state, this.scopes.global, GLOBAL_PARTITION);
    if (context.cluster == null) return;
    await this.writePartition(
      state,
      this.scopes.cluster,
      clusterPartitionBase(context.cluster),
    );
    if (context.project == null) return;
    await this.writePartition(
      state,
      this.scopes.project,
      projectPartitionBase(context.cluster, context.project),
    );
  }

  /**
   * Load the cluster and project (or project-only) slices for the target context,
   * defaulting unvisited partitions to their initial slices. When includeCluster is
   * true the target project is re-derived from the loaded cluster partition.
   */
  async loadSwap(
    state: S,
    context: Context,
    includeCluster: boolean,
  ): Promise<Partial<S>> {
    const out: Partial<S> = {};
    let project = context.project;
    if (includeCluster) {
      const clusterSlices =
        context.cluster == null
          ? {}
          : await this.readPartition(
              this.scopes.cluster,
              clusterPartitionBase(context.cluster),
            );
      this.fill(out, this.scopes.cluster, clusterSlices);
      // The target cluster's partition records which project was last active.
      project = this.getContext({ ...state, ...out }).project;
    }
    const projectSlices =
      context.cluster == null || project == null
        ? {}
        : await this.readPartition(
            this.scopes.project,
            projectPartitionBase(context.cluster, project),
          );
    this.fill(out, this.scopes.project, projectSlices);
    return out;
  }

  /** Step every active partition back one version. */
  async revert(context: Context): Promise<void> {
    const bases = [GLOBAL_PARTITION];
    if (context.cluster != null) {
      bases.push(clusterPartitionBase(context.cluster));
      if (context.project != null)
        bases.push(projectPartitionBase(context.cluster, context.project));
    }
    for (const base of bases)
      await this.setVersion(base, this.prevVersion(await this.readVersion(base)));
  }

  /** Clear the entire store. */
  async clear(): Promise<void> {
    await this.db.clear();
    this.versions.clear();
  }

  // The global partition names the selected cluster, whose partition names the active
  // project, whose partition holds the workspace.
  private async compose(): Promise<void> {
    const state = deep.copy(this.initial);
    Object.assign(
      state,
      await this.readPartition(this.scopes.global, GLOBAL_PARTITION),
    );
    const { cluster } = this.getContext(state);
    if (cluster != null)
      Object.assign(
        state,
        await this.readPartition(this.scopes.cluster, clusterPartitionBase(cluster)),
      );
    const context = this.getContext(state);
    if (context.cluster != null && context.project != null)
      Object.assign(
        state,
        await this.readPartition(
          this.scopes.project,
          projectPartitionBase(context.cluster, context.project),
        ),
      );
    this.initialState = state;
    this.context = context;
  }

  private nextVersion(v: number): number {
    return (v + 1) % HISTORY_LENGTH;
  }

  private prevVersion(v: number): number {
    return (v - 1 + HISTORY_LENGTH) % HISTORY_LENGTH;
  }

  private async readVersion(base: string): Promise<number> {
    const cached = this.versions.get(base);
    if (cached != null) return cached;
    const stored = (await this.db.get(
      partitionVersionKey(base),
    )) as StateVersionValue | null;
    const version = stored?.version ?? 0;
    this.versions.set(base, version);
    return version;
  }

  private async setVersion(base: string, version: number): Promise<void> {
    this.versions.set(base, version);
    await this.db.set(partitionVersionKey(base), { version }).catch((err: unknown) => {
      console.error(`failed to bump version of partition ${base}`, err);
    });
  }

  private migrateSlice(key: SliceKey<S>, raw: unknown): unknown {
    const migrator = this.migrators[key];
    if (migrator == null || raw == null) return raw;
    try {
      return migrator(raw);
    } catch (err) {
      console.error(`failed to migrate slice ${key}. using its initial state.`, err);
      return null;
    }
  }

  private async readPartition(
    slices: Array<SliceKey<S>>,
    base: string,
  ): Promise<Partial<S>> {
    const version = await this.readVersion(base);
    const data = (await this.db.get(
      partitionStateKey(base, version),
    )) as Partial<S> | null;
    const out: Partial<S> = {};
    if (data == null) return out;
    slices.forEach((key) => {
      const migrated = this.migrateSlice(key, data[key]);
      if (migrated != null) out[key] = migrated as S[typeof key];
    });
    return out;
  }

  private async writePartition(
    state: S,
    slices: Array<SliceKey<S>>,
    base: string,
  ): Promise<void> {
    const version = this.nextVersion(await this.readVersion(base));
    const data: Partial<S> = {};
    slices.forEach((key) => {
      data[key] = state[key];
    });
    await this.db.set(partitionStateKey(base, version), data).catch((err: unknown) => {
      console.error(`failed to write partition ${base} at version ${version}`, err);
    });
    await this.setVersion(base, version);
  }

  private fill(out: Partial<S>, slices: Array<SliceKey<S>>, data: Partial<S>): void {
    slices.forEach((key) => {
      out[key] = data[key] ?? deep.copy(this.initial[key]);
    });
  }
}

const PERSIST_DEBOUNCE = TimeSpan.milliseconds(250);

/**
 * Creates a middleware that persists the redux store state to the given engine after an
 * action is dispatched, and swaps the cluster and project partitions when the (cluster,
 * project) context changes: the outgoing context flushes under its own keys, the
 * target's partitions load, and a single hydrate action applies them.
 */
const createMiddleware = <S extends object>(
  engine: Engine<S>,
  getContext: (state: S) => Context,
  debounceInterval: CrudeTimeSpan = PERSIST_DEBOUNCE,
): Middleware<record.Unknown> => {
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
          const includeCluster = ctx.cluster !== old.cluster;
          current = ctx;
          swapping = true;
          const gen = ++swapGen;
          engine
            .persist(state, old)
            .then(async () => {
              if (gen !== swapGen) return;
              const loaded = await engine.loadSwap(state, ctx, includeCluster);
              if (gen !== swapGen) return;
              store.dispatch(hydrate(loaded as record.Unknown));
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

const passThrough: Middleware<record.Unknown> = () => (next) => (action) =>
  next(action);

/**
 * Opens persistence over the redux store: composes the state held for the session
 * context found on disk, and builds the middleware that keeps it written. Only the main
 * window persists; every other window gets no state and a pass-through middleware.
 * @param config - The configuration for the engine.
 * @returns The composed state to preload the store with, and the middleware to install
 * on it.
 */
export const open = async <S extends object>(
  config: Config<S>,
): Promise<{ initialState?: S; middleware: Middleware<record.Unknown> }> => {
  if (!Runtime.isMainWindow()) return { middleware: passThrough };
  const engine = await Engine.open(config);
  return {
    initialState: engine.initialState,
    middleware: createMiddleware(engine, config.getContext, config.debounceInterval),
  };
};
