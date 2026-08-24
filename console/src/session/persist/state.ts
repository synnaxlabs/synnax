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

import { type Entry, openSugaredKV, type SugaredKV } from "@/session/persist/kv";
import { Runtime } from "@/session/runtime";

// On desktop this names session.json inside the tauri local app data directory: on
// macOS ~/Library/Application Support/com.synnaxlabs.dev, on Windows
// %LOCALAPPDATA%/com.synnaxlabs.dev. In the browser it names an IndexedDB database.
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

/** A slice keying its state by window, which is what a lens splits. */
export interface Windowed {
  windows: Record<string, unknown>;
}

/**
 * How a window-scoped slice splits across the windows it holds state for. The store
 * supplies one, so persistence never assumes anything about a slice beyond the record
 * a window scope already implies.
 */
export interface Lens {
  /** The window keys the slice holds state for. */
  keys: (slice: Windowed) => string[];
  /** The slice narrowed to one window, which is what that window's partition stores. */
  narrow: (slice: Windowed, key: string) => Windowed;
  /** Folds a window's stored slice back into the one being composed. */
  widen: (into: Windowed, from: Windowed) => Windowed;
}

// A window scope is the store's promise that the slices in it key state by window. S
// carries no such constraint, so these two are where that promise is taken at its
// word, and the only place a slice is treated as anything but opaque.
const narrowTo = <S extends object, K extends SliceKey<S>>(
  lens: Lens,
  slice: S[K],
  window: string,
): S[K] => lens.narrow(slice as Windowed, window) as S[K];

const widenInto = <S extends object, K extends SliceKey<S>>(
  lens: Lens,
  into: S[K],
  from: S[K],
): S[K] => lens.widen(into as Windowed, from as Windowed) as S[K];

/**
 * Where each slice lives on disk. Every slice of S appears in exactly one scope or
 * in transient; {@link open} throws when one is missing, so adding a slice to the
 * store forces a decision about its durability.
 */
export interface Scopes<S extends object> {
  global: SliceSchemas<S>;
  core: SliceSchemas<S>;
  project: SliceSchemas<S>;
  /** Slices split one partition per window, through {@link Config.lens}. */
  window: SliceSchemas<S>;
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
  const { global, core, project, window, transient } = scopes;
  const declared = [
    ...scopeKeys(global),
    ...scopeKeys(core),
    ...scopeKeys(project),
    ...scopeKeys(window),
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
  /** getWindows names the windows the session is running, one partition each. */
  getWindows?: (state: S) => string[];
  /** How window-scoped slices split across windows. Required when that scope is used. */
  lens?: Lens | null;
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
 * Deletes everything stored under a Core partition key: the partition itself and the
 * project partitions beneath it. Dispatched when nothing can reach that state again.
 */
export const purge = createAction<string>("persist/purge");
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
  | typeof purge
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
  /** The window this partition holds, when it holds one window's slices. */
  private readonly window: { key: string; lens: Lens } | null;
  /** The slices last known to be in the ring, so an idle partition is left alone. */
  private committed: Partial<S> | null = null;
  private staged: Partial<S> | null = null;

  constructor(
    db: SugaredKV,
    base: string,
    schemas: SliceSchemas<S>,
    window: { key: string; lens: Lens } | null = null,
  ) {
    this.db = db;
    this.base = base;
    this.schemas = schemas;
    this.window = window;
  }

  /**
   * Read the slices at the current ring slot. A slice whose stored bytes fail its
   * schema is dropped, leaving the caller to fall back to its initial state.
   */
  async read(): Promise<Partial<S>> {
    const out: Partial<S> = {};
    const slot = (await this.readSlot()) ?? 0;
    const data = (await this.db.get(this.stateKey(slot))) as Partial<S> | null;
    this.committed = out;
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
   * The entries that put the state's slices in the next ring slot and move the pointer
   * onto it. The caller writes them, then calls {@link commit}.
   * @returns null when the slices match the ones already committed, so an idle
   * partition is left alone and the ring holds sessions rather than the last second of
   * writes.
   * @throws {Error} if the store refuses the pointer read.
   */
  async stage(state: S): Promise<Entry[] | null> {
    const data: Partial<S> = {};
    this.keys().forEach((key) => {
      const slice = state[key];
      data[key] =
        this.window == null
          ? slice
          : narrowTo<S, typeof key>(this.window.lens, slice, this.window.key);
    });
    if (this.committed != null && deep.equal(this.committed, data)) return null;
    const stored = await this.readSlot();
    const slot = stored == null ? 0 : nextSlot(stored);
    this.staged = data;
    return [
      { key: this.stateKey(slot), value: data },
      { key: this.slotKey(), value: { slot } },
    ];
  }

  /** Marks what {@link stage} returned as the slices now in the ring. */
  commit(): void {
    if (this.staged == null) return;
    this.committed = this.staged;
    this.staged = null;
  }

  /**
   * Step the ring pointer back one slot.
   * @returns false when the slot behind holds nothing, leaving the pointer alone.
   */
  async revert(): Promise<boolean> {
    const slot = prevSlot((await this.readSlot()) ?? 0);
    if ((await this.db.get(this.stateKey(slot))) == null) return false;
    this.committed = null;
    await this.db.setMany([{ key: this.slotKey(), value: { slot } }]);
    return true;
  }

  private keys(): Array<SliceKey<S>> {
    return Object.keys(this.schemas) as Array<SliceKey<S>>;
  }

  /** Every key the partition occupies, for deleting it whole. */
  occupied(): string[] {
    const slots = Array.from({ length: HISTORY_LENGTH }, (_, slot) =>
      this.stateKey(slot),
    );
    return [...slots, this.slotKey()];
  }

  private stateKey(slot: number): string {
    return `${this.base}.${slot}`;
  }

  private slotKey(): string {
    return `${this.base}.slot`;
  }

  /**
   * The slot the pointer names, or null when the ring has never been written or the
   * pointer is corrupt.
   * @throws {Error} if the store refuses the read.
   */
  private async readSlot(): Promise<number | null> {
    const stored = slotPointerZ.safeParse(await this.db.get(this.slotKey()));
    return stored.success ? stored.data.slot : null;
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
  private readonly partitions = new Map<string, Partition<S>>();
  private readonly initial: S;
  private readonly scopes: Scopes<S>;
  private readonly getContext: (state: S) => Context;
  private readonly getWindows: (state: S) => string[];
  private readonly lens: Lens | null;
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
    getWindows = () => [],
    lens = null,
    exclude = [],
    seed,
    openKV = openSugaredKV,
  }: Config<S>) {
    this.initial = deep.copy(initial);
    this.scopes = scopes;
    this.getContext = getContext;
    this.getWindows = getWindows;
    this.lens = lens;
    this.exclude = exclude;
    this.seed = seed;
    this.db = openKV(STORE_NAME);
    this.initialState = deep.copy(initial);
    this.context = getContext(this.initialState);
  }

  /**
   * Persist the state's partitions under the given context's keys, as one write. Every
   * partition the context reaches commits together, so a reader never sees one scope
   * of a session without the others.
   * @throws {Error} if the store rejects the write.
   */
  async persist(rawState: S, context: Context): Promise<void> {
    let state = deep.copy(rawState);
    this.exclude.forEach((exclude) => (state = exclude(state)));
    const partitions = [
      ...this.activePartitions(context),
      ...this.windowPartitions(context, state),
    ];
    const entries: Entry[] = [];
    for (const partition of partitions)
      entries.push(...((await partition.stage(state)) ?? []));
    const closed = this.closedWindows(context, state);
    if (entries.length === 0 && closed.length === 0) return;
    try {
      if (entries.length > 0) await this.db.setMany(entries);
      if (closed.length > 0) await this.db.deleteMany(closed);
    } catch (err) {
      throw new Error("failed to write the session state", { cause: err });
    }
    partitions.forEach((partition) => partition.commit());
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
    const target = { ...state, ...out };
    const windowSlices =
      project == null ? {} : await this.readWindows({ ...context, project }, target);
    this.fill(out, this.scopes.window, windowSlices);
    return out;
  }

  /**
   * Step the innermost active partition holding history back one version. Only that
   * one moves: the partitions outside it name the session's context, and stepping
   * those back would move the session somewhere else instead of undoing its state.
   */
  async revert(context: Context, state: S): Promise<void> {
    let reverted = false;
    for (const partition of this.windowPartitions(context, state))
      reverted = (await partition.revert()) || reverted;
    if (reverted) return;
    for (const partition of this.activePartitions(context).reverse())
      if (await partition.revert()) return;
  }

  /** Clear the entire store. */
  async clear(): Promise<void> {
    this.partitions.clear();
    await this.db.clear();
  }

  /**
   * Delete the Core partition stored under the key, and every project partition under
   * it.
   * @throws {Error} if the store refuses the read or a delete.
   */
  async purge(core: string): Promise<void> {
    const owned = (base: string): boolean =>
      base === `core.${core}` ||
      base.startsWith(`project.${core}.`) ||
      base.startsWith(`window.${core}.`);
    const stale = (await this.db.keys()).filter((key) =>
      owned(key.slice(0, key.lastIndexOf("."))),
    );
    await this.db.deleteMany(stale);
    for (const base of this.partitions.keys())
      if (owned(base)) this.partitions.delete(base);
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
      if (context.core != null && context.project != null) {
        Object.assign(state, await this.project(context.core, context.project).read());
        // The project partition holds the window bookkeeping, so the windows a session
        // is running are only known once it has been read.
        Object.assign(state, await this.readWindows(context, state));
      }
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
    return this.partition("global", this.scopes.global);
  }

  private core(key: string): Partition<S> {
    return this.partition(`core.${key}`, this.scopes.core);
  }

  private project(core: string, project: string): Partition<S> {
    return this.partition(`project.${core}.${project}`, this.scopes.project);
  }

  private window(core: string, project: string, key: string): Partition<S> {
    if (this.lens == null)
      throw new Error("the window scope needs a lens to split its slices");
    return this.partition(`window.${core}.${project}.${key}`, this.scopes.window, {
      key,
      lens: this.lens,
    });
  }

  /**
   * The keys of window partitions this engine wrote whose windows are gone. Window
   * keys are minted fresh per open, so a partition left behind is one the store
   * carries forever.
   */
  private closedWindows(context: Context, state: S): string[] {
    const { core, project } = context;
    if (core == null || project == null) return [];
    const prefix = `window.${core}.${project}.`;
    const live = new Set(this.getWindows(state).map((key) => `${prefix}${key}`));
    const stale: string[] = [];
    for (const [base, partition] of this.partitions)
      if (base.startsWith(prefix) && !live.has(base)) {
        stale.push(...partition.occupied());
        this.partitions.delete(base);
      }
    return stale;
  }

  /** One partition per window the session is running, under the given context. */
  private windowPartitions(context: Context, state: S): Array<Partition<S>> {
    const { core, project } = context;
    if (core == null || project == null) return [];
    return this.getWindows(state).map((key) => this.window(core, project, key));
  }

  /** Reads every window partition the state names, folding them into one set of slices. */
  private async readWindows(context: Context, state: S): Promise<Partial<S>> {
    const out: Partial<S> = {};
    for (const partition of this.windowPartitions(context, state)) {
      const read = await partition.read();
      scopeKeys(this.scopes.window).forEach((key) => {
        const value = read[key];
        if (value == null || this.lens == null) return;
        out[key] = widenInto<S, typeof key>(
          this.lens,
          out[key] ?? deep.copy(this.initial[key]),
          value,
        );
      });
    }
    return out;
  }

  // Partitions are held rather than rebuilt so each one remembers what it committed
  // and can skip a write that would change nothing.
  private partition(
    base: string,
    schemas: SliceSchemas<S>,
    window: { key: string; lens: Lens } | null = null,
  ): Partition<S> {
    let partition = this.partitions.get(base);
    if (partition == null) {
      partition = new Partition(this.db, base, schemas, window);
      this.partitions.set(base, partition);
    }
    return partition;
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
  // A swap flushes the partition it leaves, so a purge of that partition has to
  // wait behind it or the keys come back.
  let swap: Promise<unknown> = Promise.resolve();
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
          .revert(current, state)
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
      else if (type === purge.type) {
        const { payload } = action as PayloadAction<string>;
        swap = swap
          .then(async () => await engine.purge(payload))
          .catch((err: unknown) => {
            console.error("failed to purge stored Core state", err);
          });
      } else if (type === hydrate.type) {
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
          swap = engine
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
