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

/** Versions kept per partition before the ring wraps. */
const HISTORY_LENGTH = 4;

const nextVersion = (v: number): number => (v + 1) % HISTORY_LENGTH;
const prevVersion = (v: number): number => (v - 1 + HISTORY_LENGTH) % HISTORY_LENGTH;

/**
 * A group of slices stored under one key prefix, with a bounded ring of versions
 * behind a pointer key. Owns how the group's bytes round-trip: ring advancement,
 * migration on read, and stepping the pointer back on revert.
 */
class Partition<S extends object> {
  private readonly db: SugaredKV;
  private readonly base: string;
  private readonly slices: Array<SliceKey<S>>;
  private readonly migrators: SliceMigrators<S>;

  constructor(
    db: SugaredKV,
    base: string,
    slices: Array<SliceKey<S>>,
    migrators: SliceMigrators<S>,
  ) {
    this.db = db;
    this.base = base;
    this.slices = slices;
    this.migrators = migrators;
  }

  /** Read the slices at the current ring version, migrated. */
  async read(): Promise<Partial<S>> {
    const version = await this.readVersion();
    const data = (await this.db.get(this.stateKey(version))) as Partial<S> | null;
    const out: Partial<S> = {};
    if (data == null) return out;
    this.slices.forEach((key) => {
      const migrated = this.migrateSlice(key, data[key]);
      if (migrated != null) out[key] = migrated as S[typeof key];
    });
    return out;
  }

  /** Write the state's slices into the next ring slot and advance the pointer. */
  async write(state: S): Promise<void> {
    const version = nextVersion(await this.readVersion());
    const data: Partial<S> = {};
    this.slices.forEach((key) => {
      data[key] = state[key];
    });
    await this.db.set(this.stateKey(version), data).catch((err: unknown) => {
      console.error(
        `failed to write partition ${this.base} at version ${version}`,
        err,
      );
    });
    await this.setVersion(version);
  }

  /** Step the ring pointer back one version. */
  async revert(): Promise<void> {
    await this.setVersion(prevVersion(await this.readVersion()));
  }

  private stateKey(version: number): string {
    return `${this.base}.${version}`;
  }

  private versionKey(): string {
    return `${this.base}.version`;
  }

  private async readVersion(): Promise<number> {
    const stored = (await this.db.get(this.versionKey())) as StateVersionValue | null;
    return stored?.version ?? 0;
  }

  private async setVersion(version: number): Promise<void> {
    await this.db.set(this.versionKey(), { version }).catch((err: unknown) => {
      console.error(`failed to bump version of partition ${this.base}`, err);
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
}

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
    for (const partition of this.activePartitions(context))
      await partition.write(state);
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
        context.cluster == null ? {} : await this.cluster(context.cluster).read();
      this.fill(out, this.scopes.cluster, clusterSlices);
      // The target cluster's partition records which project was last active.
      project = this.getContext({ ...state, ...out }).project;
    }
    const projectSlices =
      context.cluster == null || project == null
        ? {}
        : await this.project(context.cluster, project).read();
    this.fill(out, this.scopes.project, projectSlices);
    return out;
  }

  /** Step every active partition back one version. */
  async revert(context: Context): Promise<void> {
    for (const partition of this.activePartitions(context)) await partition.revert();
  }

  /** Clear the entire store. */
  async clear(): Promise<void> {
    await this.db.clear();
  }

  // The global partition names the selected cluster, whose partition names the active
  // project, whose partition holds the workspace.
  private async compose(): Promise<void> {
    const state = deep.copy(this.initial);
    Object.assign(state, await this.global().read());
    const { cluster } = this.getContext(state);
    if (cluster != null) Object.assign(state, await this.cluster(cluster).read());
    const context = this.getContext(state);
    if (context.cluster != null && context.project != null)
      Object.assign(state, await this.project(context.cluster, context.project).read());
    this.initialState = state;
    this.context = context;
  }

  private global(): Partition<S> {
    return new Partition(this.db, "global", this.scopes.global, this.migrators);
  }

  private cluster(key: string): Partition<S> {
    return new Partition(
      this.db,
      `cluster.${key}`,
      this.scopes.cluster,
      this.migrators,
    );
  }

  private project(cluster: string, project: string): Partition<S> {
    return new Partition(
      this.db,
      `project.${cluster}.${project}`,
      this.scopes.project,
      this.migrators,
    );
  }

  /** The partitions the given context reaches: global, then cluster, then project. */
  private activePartitions({ cluster, project }: Context): Array<Partition<S>> {
    const out = [this.global()];
    if (cluster == null) return out;
    out.push(this.cluster(cluster));
    if (project != null) out.push(this.project(cluster, project));
    return out;
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
