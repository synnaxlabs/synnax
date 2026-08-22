// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createAction,
  type Middleware,
  type MiddlewareAPI,
  type PayloadAction,
} from "@reduxjs/toolkit";
import {
  type CrudeTimeSpan,
  debounce,
  deep,
  type record,
  TimeSpan,
} from "@synnaxlabs/x";
import { z } from "zod";

import { openSugaredKV, type SugaredKV } from "@/session/persist/kv";
import { Runtime } from "@/session/runtime";

// On desktop this names session.json inside the tauri app data directory: on macOS
// ~/Library/Application Support/com.synnaxlabs.dev, on Windows
// %APPDATA%/com.synnaxlabs.dev. In the browser it names an IndexedDB database.
export const STORE_NAME = "session";

/**
 * The scope state is persisted under. State partitions into three scopes:
 * global, per-Core, and per-Core-per-project.
 */
export interface Context {
  core?: string;
  project?: string;
}

const contextsEqual = (a: Context, b: Context): boolean =>
  a.core === b.core && a.project === b.project;

export interface KVOpener {
  (base: string): SugaredKV;
}

/** Strips state that must not reach disk, returning what gets written. */
export type ExcludeFn<S extends object> = (state: S) => S;

type SliceKey<S extends object> = keyof S & string;

/**
 * The slices of one partition scope, each under the schema its bytes are parsed
 * through on read. Declaring the schema beside the scope rather than in a separate
 * table is what makes an unvalidated persisted slice unrepresentable.
 */
export type SliceSchemas<S extends object> = {
  [K in SliceKey<S>]?: z.ZodType<S[K]>;
};

/**
 * Where each slice lives on disk. Every slice of S appears in exactly one scope or
 * in transient; {@link open} throws when one is missing, so adding a slice to the
 * store forces a decision about its durability.
 */
export interface Scopes<S extends object> {
  global: SliceSchemas<S>;
  core: SliceSchemas<S>;
  project: SliceSchemas<S>;
  /** Slices deliberately never written. */
  transient: Array<SliceKey<S>>;
}

const scopeKeys = <S extends object>(scope: SliceSchemas<S>): Array<SliceKey<S>> =>
  Object.keys(scope) as Array<SliceKey<S>>;

/**
 * @throws {Error} if a slice of S is in no scope and not declared transient, or is
 * in more than one scope.
 */
const validateScopes = <S extends object>(initial: S, scopes: Scopes<S>): void => {
  const { global, core, project, transient } = scopes;
  const declared = [
    ...scopeKeys(global),
    ...scopeKeys(core),
    ...scopeKeys(project),
    ...transient,
  ];
  const seen = new Set<string>();
  const duplicated = declared.filter((key) => {
    if (seen.has(key)) return true;
    seen.add(key);
    return false;
  });
  if (duplicated.length > 0)
    throw new Error(`slices declared in more than one scope: ${duplicated.join(", ")}`);
  const missing = Object.keys(initial).filter((key) => !seen.has(key));
  if (missing.length > 0)
    throw new Error(
      `slices in no persistence scope: ${missing.join(", ")}. Add each to a scope with its schema, or to transient.`,
    );
};

export interface Config<S extends object> {
  initial: S;
  scopes: Scopes<S>;
  /** getContext reads the partition scope from state. */
  getContext: (state: S) => Context;
  exclude?: Array<ExcludeFn<S>>;
  /**
   * Slices to start from the first time the store is opened, carried over from
   * whatever the previous release left on disk. Consulted only while the store is
   * still empty, so it never overwrites what this release has written.
   */
  seed?: () => Promise<Partial<S>>;
  openKV?: KVOpener;
  debounceInterval?: CrudeTimeSpan;
}

export const revertState = createAction("persist/revertState");
export const clearState = createAction("persist/clearState");
/**
 * Replaces the swapped slices wholesale when the session context changes.
 * Dispatched by the persistence middleware, applied by a root reducer wrapper.
 */
export const hydrate = createAction<object>("persist/hydrate");

/**
 * Narrows an action to a hydrate carrying S's persisted slices. The payload is
 * whatever {@link open} loaded for the same S, so the root reducer wrapper for
 * a store reads its own slices back.
 */
export const matchHydrate = <S extends object>(
  action: unknown,
): action is PayloadAction<Partial<S>> => hydrate.match(action);
/**
 * Marks the start of a partition swap. Between this and the closing hydrate
 * (or endSwap on failure) the store holds the outgoing context's slices.
 */
export const beginSwap = createAction("persist/beginSwap");
/** Clears the swap mark when a failed swap will never hydrate. */
export const endSwap = createAction("persist/endSwap");
/**
 * Marks the store unwritable. The platform refuses it outright — a cross-origin
 * frame, private browsing on an older engine, an exhausted quota — so the session
 * runs but will not survive a reload.
 */
export const storeUnavailable = createAction("persist/storeUnavailable");
export type Action = ReturnType<
  | typeof revertState
  | typeof clearState
  | typeof hydrate
  | typeof beginSwap
  | typeof endSwap
  | typeof storeUnavailable
>;

/** Slots kept per partition before the ring wraps. */
const HISTORY_LENGTH = 4;

const nextSlot = (s: number): number => (s + 1) % HISTORY_LENGTH;
const prevSlot = (s: number): number => (s - 1 + HISTORY_LENGTH) % HISTORY_LENGTH;

const slotPointerZ = z.object({
  slot: z
    .number()
    .int()
    .min(0)
    .max(HISTORY_LENGTH - 1),
});

/**
 * A group of slices stored under one key prefix, with a bounded ring of slots
 * behind a pointer key. Owns how the group's bytes round-trip: ring advancement,
 * parsing on read, and stepping the pointer back on revert. The slot pointer is
 * unrelated to the schema version each slice carries in its own value.
 */
class Partition<S extends object> {
  private readonly db: SugaredKV;
  private readonly base: string;
  private readonly schemas: SliceSchemas<S>;

  constructor(db: SugaredKV, base: string, schemas: SliceSchemas<S>) {
    this.db = db;
    this.base = base;
    this.schemas = schemas;
  }

  /**
   * Read the slices at the current ring slot. A slice whose stored bytes fail its
   * schema is dropped, leaving the caller to fall back to its initial state.
   */
  async read(): Promise<Partial<S>> {
    const out: Partial<S> = {};
    const slot = await this.readSlot();
    const data = (await this.db.get(this.stateKey(slot))) as Partial<S> | null;
    if (data == null) return out;
    this.keys().forEach((key) => {
      const raw = data[key];
      if (raw == null) return;
      const parsed = this.schemas[key]?.safeParse(raw);
      if (parsed?.success !== true)
        return console.error(
          `discarding stored slice ${key}: it does not match its schema`,
          parsed?.error,
        );
      out[key] = parsed.data;
    });
    return out;
  }

  /**
   * Write the state's slices into the next ring slot and advance the pointer onto it,
   * as one unit.
   * @throws {Error} if the store rejects the write.
   */
  async write(state: S): Promise<void> {
    const slot = nextSlot(await this.readSlot());
    const data: Partial<S> = {};
    this.keys().forEach((key) => {
      data[key] = state[key];
    });
    try {
      await this.db.setMany([
        { key: this.stateKey(slot), value: data },
        { key: this.slotKey(), value: { slot } },
      ]);
    } catch (err) {
      throw new Error(`failed to write partition ${this.base} at slot ${slot}`, {
        cause: err,
      });
    }
  }

  /** Step the ring pointer back one slot. */
  async revert(): Promise<void> {
    const slot = prevSlot(await this.readSlot());
    await this.db.setMany([{ key: this.slotKey(), value: { slot } }]);
  }

  private keys(): Array<SliceKey<S>> {
    return Object.keys(this.schemas) as Array<SliceKey<S>>;
  }

  private stateKey(slot: number): string {
    return `${this.base}.${slot}`;
  }

  private slotKey(): string {
    return `${this.base}.slot`;
  }

  /** Bad or absent bytes read as slot zero; a store that refuses the read throws. */
  private async readSlot(): Promise<number> {
    const stored = slotPointerZ.safeParse(await this.db.get(this.slotKey()));
    return stored.success ? stored.data.slot : 0;
  }
}

/** Clear the entire store and reload the page. */
export const hardClearAndReload = () => {
  if (!Runtime.isMainWindow()) return;
  openSugaredKV(STORE_NAME)
    .clear()
    .catch((err: unknown) => {
      console.error("failed to clear store during hard reload", err);
    })
    .finally(() => window.location.reload());
};

/** Persists the redux store state to disk, partitioned by session context. */
class Engine<S extends object> {
  /** The context {@link initialState} was composed for. */
  context: Context;
  /** The composed global + Core + project state read on open. */
  initialState: S;
  /** Whether the store refused every read on open. */
  unusable = false;

  private readonly db: SugaredKV;
  private readonly initial: S;
  private readonly scopes: Scopes<S>;
  private readonly getContext: (state: S) => Context;
  private readonly exclude: Array<ExcludeFn<S>>;
  private readonly seed?: () => Promise<Partial<S>>;

  /**
   * Opens an engine over the persisted store and composes the state it holds for the
   * context it finds.
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
    exclude = [],
    seed,
    openKV = openSugaredKV,
  }: Config<S>) {
    this.initial = deep.copy(initial);
    this.scopes = scopes;
    this.getContext = getContext;
    this.exclude = exclude;
    this.seed = seed;
    this.db = openKV(STORE_NAME);
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
   * Load the Core and project (or project-only) slices for the target context,
   * defaulting unvisited partitions to their initial slices. When includeCore is
   * true the target project is re-derived from the loaded Core partition.
   */
  async loadSwap(
    state: S,
    context: Context,
    includeCore: boolean,
  ): Promise<Partial<S>> {
    const out: Partial<S> = {};
    let project = context.project;
    if (includeCore) {
      const coreSlices =
        context.core == null ? {} : await this.core(context.core).read();
      this.fill(out, this.scopes.core, coreSlices);
      // The target Core's partition records which project was last active.
      project = this.getContext({ ...state, ...out }).project;
    }
    const projectSlices =
      context.core == null || project == null
        ? {}
        : await this.project(context.core, project).read();
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

  // The global partition names the selected Core, whose partition names the active
  // project, whose partition holds the workspace.
  private async compose(): Promise<void> {
    const state = deep.copy(this.initial);
    try {
      Object.assign(state, await this.readSeed());
      Object.assign(state, await this.global().read());
      const { core } = this.getContext(state);
      if (core != null) Object.assign(state, await this.core(core).read());
      const context = this.getContext(state);
      if (context.core != null && context.project != null)
        Object.assign(state, await this.project(context.core, context.project).read());
      this.context = context;
    } catch (err) {
      // A platform that refuses storage outright must not stop the app booting. The
      // session runs from its initial state; the middleware tells the user it will
      // not be saved.
      console.error("failed to read the session store", err);
      this.unusable = true;
    }
    this.initialState = state;
  }

  /** The seed's slices, or nothing when the store already holds a session. */
  private async readSeed(): Promise<Partial<S>> {
    if (this.seed == null) return {};
    try {
      if ((await this.db.length()) > 0) return {};
      return await this.seed();
    } catch (err) {
      console.error("failed to seed the session store", err);
      return {};
    }
  }

  private global(): Partition<S> {
    return new Partition(this.db, "global", this.scopes.global);
  }

  private core(key: string): Partition<S> {
    return new Partition(this.db, `core.${key}`, this.scopes.core);
  }

  private project(core: string, project: string): Partition<S> {
    return new Partition(this.db, `project.${core}.${project}`, this.scopes.project);
  }

  /** The partitions the given context reaches: global, then Core, then project. */
  private activePartitions({ core, project }: Context): Array<Partition<S>> {
    const out = [this.global()];
    if (core == null) return out;
    out.push(this.core(core));
    if (project != null) out.push(this.project(core, project));
    return out;
  }

  private fill(out: Partial<S>, scope: SliceSchemas<S>, data: Partial<S>): void {
    scopeKeys(scope).forEach((key) => {
      out[key] = data[key] ?? deep.copy(this.initial[key]);
    });
  }
}

const PERSIST_DEBOUNCE = TimeSpan.milliseconds(250);

/**
 * Creates a middleware that persists the redux store state to the given engine after an
 * action is dispatched, and swaps the Core and project partitions when the (Core,
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
  let announced = false;
  const announceUnavailable = (store: MiddlewareAPI) => {
    if (announced) return;
    announced = true;
    store.dispatch(storeUnavailable());
  };
  // Concurrent switches race: a slow stale swap hydrating last would clobber
  // the newer context's slices, so only the latest generation may hydrate.
  let swapGen = 0;
  return (store) => {
    if (engine.unusable) announceUnavailable(store);
    const debouncedPersist = debounce.debounce(() => {
      if (swapping) return;
      engine.persist(store.getState() as S, current).catch((e: unknown) => {
        console.error("failed to persist state", e);
        announceUnavailable(store);
      });
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
        debouncedPersist();
      } else {
        const ctx = getContext(state);
        if (!contextsEqual(ctx, current)) {
          const old = current;
          const includeCore = ctx.core !== old.core;
          current = ctx;
          swapping = true;
          const gen = ++swapGen;
          store.dispatch(beginSwap());
          engine
            .persist(state, old)
            .then(async () => {
              if (gen !== swapGen) return;
              const loaded = await engine.loadSwap(state, ctx, includeCore);
              if (gen !== swapGen) return;
              store.dispatch(hydrate(loaded));
            })
            .catch((err: unknown) => {
              if (gen === swapGen) {
                swapping = false;
                store.dispatch(endSwap());
              }
              console.error("failed to swap session context", err);
            });
        } else debouncedPersist();
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
 * @throws {Error} if a slice is in no scope and not declared transient.
 */
export const open = async <S extends object>(
  config: Config<S>,
): Promise<{ initialState?: S; middleware: Middleware<record.Unknown> }> => {
  validateScopes(config.initial, config.scopes);
  if (!Runtime.isMainWindow()) return { middleware: passThrough };
  const engine = await Engine.open(config);
  return {
    initialState: engine.initialState,
    middleware: createMiddleware(engine, config.getContext, config.debounceInterval),
  };
};
