// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare, destructor, type record, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import type z from "zod";

import {
  type ChannelListener,
  ScopedUnaryStore,
  type Store,
  type UnaryStore,
  type UnaryStoreConfig,
} from "@/flux/base/store";
import { type Shape } from "@/flux/base/types";
import { type status } from "@/status/aether";

const DEFAULT_COALESCE_WINDOW = TimeSpan.milliseconds(500);
const DEFAULT_STACK_CAP = 200;
const STALE_AUTO_ADVANCE_CAP = 10;

export interface StackEntry<Action> {
  forward: Action[];
  inverse: Action[];
  kind: string;
  ts: TimeStamp;
  targets: readonly string[];
}

interface UndoState<Action> {
  undo: StackEntry<Action>[];
  redo: StackEntry<Action>[];
  remoteTouched: Record<string, TimeStamp>;
}

const ZERO_UNDO = <A>(): UndoState<A> => ({ undo: [], redo: [], remoteTouched: {} });

const pushOnto = <A>(
  stack: StackEntry<A>[],
  next: StackEntry<A>,
  window: TimeSpan,
  cap: number,
): StackEntry<A>[] => {
  const top = stack[stack.length - 1];
  const merged =
    top != null &&
    top.kind === next.kind &&
    next.ts.span(top.ts).lessThanOrEqual(window) &&
    compare.unorderedPrimitiveArrays(top.targets, next.targets) === compare.EQUAL
      ? {
          forward: [...top.forward, ...next.forward],
          inverse: [...next.inverse, ...top.inverse],
          kind: next.kind,
          ts: next.ts,
          targets: top.targets,
        }
      : null;
  const out = merged != null ? [...stack.slice(0, -1), merged] : [...stack, next];
  return out.length > cap ? out.slice(out.length - cap) : out;
};

export interface DispatchReducer<State extends Shape, Action> {
  (
    state: State,
    actions: Action[],
  ): { next: State; inverse: Action[]; targets: readonly string[] };
}

/** Result of replaying a set of actions locally. The caller commits the
 *  matching server send and runs `rollback` on failure. */
export interface ReplayResult<Action> {
  processed: Action[];
  inverse: Action[];
  targets: readonly string[];
  rollback: destructor.Destructor;
}

/** Result of preparing to undo or redo. The caller applies `actions` and runs
 *  `commit` on success (which itself returns a destructor for failure-rollback). */
export interface Reversal<Action> {
  actions: Action[];
  commit: () => destructor.Destructor;
}

/** A staged set of dispatches committed atomically as one undoable. */
export interface Transaction<Action> {
  add: (actions: Action | Action[]) => void;
  commit: () => Promise<boolean>;
  abort: () => void;
}

/** Sends an already-processed action list to the server. Used by transaction
 *  commit so the substrate can bind client/sessionKey at hook-call time. */
export type DispatchSend<Action> = (actions: Action[]) => Promise<void>;

/**
 * The canonical wire shape for a broadcast action frame. The consumer's
 * `schema` must produce this shape (use a zod transform if the server emits
 * differently-named fields).
 */
export interface DispatchFrame<Key, Action> {
  key: Key;
  sessionKey: string;
  actions: Action[];
}

/**
 * A UnaryStore augmented with the building blocks for an action-based
 * dispatch loop. Holds doc state alongside per-doc undo/redo stacks.
 * Deleting a document key cascades and drops its undo state.
 *
 * The store owns pure state transitions (replay, recordEntry, prepareUndo,
 * applyRemote). It does *not* own the network send — that's the substrate's
 * job. Tests can substitute their own substrate while reading the same store.
 */
export interface UndoableUnaryStore<
  Key extends record.Key,
  State extends Shape,
  Action,
> extends UnaryStore<Key, State> {
  /**
   * Run preprocess + reduce against the cached doc and write the result.
   * Returns the reducer output and a rollback that restores the prior doc
   * state. Returns null if the doc isn't cached.
   */
  replay(
    key: Key,
    actions: Action[],
    opts?: { skipPreprocess?: boolean },
  ): ReplayResult<Action> | null;
  /**
   * Append an undoable entry to the stack. Filters non-undoable actions,
   * derives the kind, and coalesces with the prior entry if it falls within
   * the configured window with matching kind/targets. Clears redo.
   */
  recordEntry(
    key: Key,
    forward: Action[],
    inverse: Action[],
    targets: readonly string[],
    kindOverride?: string,
  ): destructor.Destructor;
  /**
   * Drop stale entries from the tail and return the topmost applicable undo
   * (its inverse actions plus a commit closure that pops the entry and
   * pushes the matching redo entry).
   */
  prepareUndo(key: Key): Reversal<Action> | null;
  /** Mirror of prepareUndo for the redo stack. */
  prepareRedo(key: Key): Reversal<Action> | null;
  /**
   * Stage actions to commit atomically as one undoable. The substrate
   * supplies `send` so the store doesn't depend on client/sessionKey.
   */
  beginTransaction(
    key: Key,
    send: DispatchSend<Action>,
    kind?: string,
  ): Transaction<Action>;
  /** Apply a remote frame and stamp its targets as remote-touched. */
  applyRemote(key: Key, actions: Action[]): void;
  /** Mark internal keys as remote-touched at the given ts (default: now). */
  markRemoteTouched(
    key: Key,
    targets: readonly string[],
    ts?: TimeStamp,
  ): destructor.Destructor;
  hasUndo(key: Key): boolean;
  hasRedo(key: Key): boolean;
  onUndoStateChange(callback: () => void, key?: Key): destructor.Destructor;
}

interface StoreOpts<Key extends record.Key, State extends Shape, Action> {
  handleError: status.ErrorHandler;
  reduce: DispatchReducer<State, Action>;
  equal?: (a: State, b: State, key: Key) => boolean;
  preprocess?: (state: State, actions: Action[]) => Action[];
  isUndoable?: (action: Action) => boolean;
  kindOf?: (actions: Action[]) => string;
  coalesceWindow?: TimeSpan;
  stackCap?: number;
}

class UndoableStore<Key extends record.Key, State extends Shape, Action> {
  private readonly docs: ScopedUnaryStore<Key, State>;
  private readonly undos: ScopedUnaryStore<Key, UndoState<Action>>;
  private readonly reduce: DispatchReducer<State, Action>;
  private readonly preprocess?: (state: State, actions: Action[]) => Action[];
  private readonly isUndoable: (action: Action) => boolean;
  private readonly kindOf: (actions: Action[]) => string;
  private readonly coalesceWindow: TimeSpan;
  private readonly stackCap: number;

  constructor(opts: StoreOpts<Key, State, Action>) {
    this.docs = new ScopedUnaryStore<Key, State>(opts.handleError, opts.equal);
    this.undos = new ScopedUnaryStore<Key, UndoState<Action>>(opts.handleError);
    this.reduce = opts.reduce;
    this.preprocess = opts.preprocess;
    this.isUndoable = opts.isUndoable ?? (() => true);
    this.kindOf = opts.kindOf ?? (() => "default");
    this.coalesceWindow = opts.coalesceWindow ?? DEFAULT_COALESCE_WINDOW;
    this.stackCap = opts.stackCap ?? DEFAULT_STACK_CAP;
  }

  private updateUndo(
    scope: string,
    key: Key,
    fn: (s: UndoState<Action>) => UndoState<Action>,
  ): destructor.Destructor {
    return this.undos.set(scope, key, (prev) => fn(prev ?? ZERO_UNDO<Action>()));
  }

  replay(
    scope: string,
    key: Key,
    actions: Action[],
    opts: { skipPreprocess?: boolean } = {},
  ): ReplayResult<Action> | null {
    const current = this.docs.get(key) as State | undefined;
    if (current == null) return null;
    const processed =
      !opts.skipPreprocess && this.preprocess != null
        ? this.preprocess(current, actions)
        : actions;
    const { next, inverse, targets } = this.reduce(current, processed);
    return { processed, inverse, targets, rollback: this.docs.set(scope, key, next) };
  }

  recordEntry(
    scope: string,
    key: Key,
    forward: Action[],
    inverse: Action[],
    targets: readonly string[],
    kindOverride?: string,
  ): destructor.Destructor {
    if (targets.length === 0) return destructor.NOOP;
    const undoableForward = forward.filter(this.isUndoable);
    if (undoableForward.length === 0) return destructor.NOOP;
    const entry: StackEntry<Action> = {
      forward: undoableForward,
      inverse,
      kind: kindOverride ?? this.kindOf(forward),
      ts: TimeStamp.now(),
      targets,
    };
    return this.updateUndo(scope, key, (s) => ({
      ...s,
      undo: pushOnto(s.undo, entry, this.coalesceWindow, this.stackCap),
      redo: [],
    }));
  }

  prepareUndo(scope: string, key: Key): Reversal<Action> | null {
    const state = this.undos.get(key) as UndoState<Action> | undefined;
    if (state == null || state.undo.length === 0) return null;
    // Walk down the stack from the top, skipping stale entries (those
    // whose targets were touched by a remote action after the entry was
    // pushed). Bail after STALE_AUTO_ADVANCE_CAP stale entries.
    let stale = 0;
    let idx = -1;
    for (let i = state.undo.length - 1; i >= 0; i--) {
      const e = state.undo[i];
      if (e.targets.some((t) => state.remoteTouched[t]?.after(e.ts))) {
        stale++;
        if (stale >= STALE_AUTO_ADVANCE_CAP) break;
      } else {
        idx = i;
        break;
      }
    }
    if (idx === -1) {
      if (stale > 0)
        this.updateUndo(scope, key, (s) => ({
          ...s,
          undo: s.undo.slice(0, s.undo.length - stale),
        }));
      return null;
    }
    const entry = state.undo[idx];
    return {
      actions: entry.inverse,
      commit: () =>
        this.updateUndo(scope, key, (s) => ({
          ...s,
          undo: s.undo.slice(0, idx),
          redo: [...s.redo, entry],
        })),
    };
  }

  prepareRedo(scope: string, key: Key): Reversal<Action> | null {
    const state = this.undos.get(key) as UndoState<Action> | undefined;
    if (state == null || state.redo.length === 0) return null;
    const entry = state.redo[state.redo.length - 1];
    return {
      actions: entry.forward,
      commit: () =>
        this.updateUndo(scope, key, (s) => ({
          ...s,
          undo: [...s.undo, entry],
          redo: s.redo.slice(0, -1),
        })),
    };
  }

  beginTransaction(
    scope: string,
    key: Key,
    send: DispatchSend<Action>,
    kind?: string,
  ): Transaction<Action> {
    const initial = this.docs.get(key) as State | undefined;
    const accumulated: Action[] = [];
    let done = false;
    return {
      add: (actions) => {
        if (done) throw new Error("transaction finalized");
        if (initial == null) return;
        const arr = Array.isArray(actions) ? actions : [actions];
        const r = this.replay(scope, key, arr);
        if (r != null) accumulated.push(...r.processed);
      },
      commit: async () => {
        if (done) return false;
        done = true;
        if (accumulated.length === 0 || initial == null) return false;
        // Inverse must be computed against the pre-transaction snapshot so
        // undo restores to where the user started, not the last-add post-state.
        const { inverse, targets } = this.reduce(initial, accumulated);
        const stackRollback = this.recordEntry(
          scope,
          key,
          accumulated,
          inverse,
          targets,
          kind,
        );
        try {
          await send(accumulated);
          return true;
        } catch {
          stackRollback();
          this.docs.set(scope, key, initial);
          return false;
        }
      },
      abort: () => {
        if (done) return;
        done = true;
        if (initial != null) this.docs.set(scope, key, initial);
      },
    };
  }

  markRemoteTouched(
    scope: string,
    key: Key,
    targets: readonly string[],
    ts: TimeStamp = TimeStamp.now(),
  ): destructor.Destructor {
    if (targets.length === 0) return destructor.NOOP;
    return this.updateUndo(scope, key, (s) => {
      const remoteTouched = { ...s.remoteTouched };
      for (const t of targets) remoteTouched[t] = ts;
      return { ...s, remoteTouched };
    });
  }

  applyRemote(scope: string, key: Key, actions: Action[]): void {
    const current = this.docs.get(key) as State | undefined;
    if (current == null) return;
    const { next, targets } = this.reduce(current, actions);
    this.docs.set(scope, key, next);
    this.markRemoteTouched(scope, key, targets);
  }

  cascadeDelete(
    scope: string,
    key: Key | Key[] | ((value: State, k: Key) => boolean),
  ): destructor.Destructor {
    let arg: typeof key;
    let cascadeKeys: Key[];
    if (typeof key === "function") {
      cascadeKeys = [];
      arg = (value: State, k: Key) => {
        const matched = key(value, k);
        if (matched) cascadeKeys.push(k);
        return matched;
      };
    } else {
      cascadeKeys = Array.isArray(key) ? key : [key];
      arg = key;
    }
    const docRollback = this.docs.delete(scope, arg);
    const undoRollback = this.undos.delete(scope, cascadeKeys);
    return () => {
      undoRollback();
      docRollback();
    };
  }

  hasUndo(key: Key): boolean {
    const s = this.undos.get(key) as UndoState<Action> | undefined;
    return (s?.undo.length ?? 0) > 0;
  }

  hasRedo(key: Key): boolean {
    const s = this.undos.get(key) as UndoState<Action> | undefined;
    return (s?.redo.length ?? 0) > 0;
  }

  onUndoStateChange(
    scope: string,
    callback: () => void,
    key?: Key,
  ): destructor.Destructor {
    const offSet = this.undos.onSet(scope, () => callback(), key);
    const offDelete = this.undos.onDelete(scope, () => callback(), key);
    return () => {
      offSet();
      offDelete();
    };
  }

  scope(scope: string): UndoableUnaryStore<Key, State, Action> {
    return {
      ...this.docs.scope(scope),
      delete: (key) => this.cascadeDelete(scope, key),
      replay: (key, actions, opts) => this.replay(scope, key, actions, opts),
      recordEntry: (key, forward, inverse, targets, kindOverride) =>
        this.recordEntry(scope, key, forward, inverse, targets, kindOverride),
      prepareUndo: (key) => this.prepareUndo(scope, key),
      prepareRedo: (key) => this.prepareRedo(scope, key),
      beginTransaction: (key, send, kind) =>
        this.beginTransaction(scope, key, send, kind),
      applyRemote: (key, actions) => this.applyRemote(scope, key, actions),
      markRemoteTouched: (key, targets, ts) =>
        this.markRemoteTouched(scope, key, targets, ts),
      hasUndo: (key) => this.hasUndo(key),
      hasRedo: (key) => this.hasRedo(key),
      onUndoStateChange: (callback, key) =>
        this.onUndoStateChange(scope, callback, key),
    };
  }
}

export interface CreateUndoableStoreParams<
  Key extends record.Key,
  State extends Shape,
  Action,
  SK extends string,
> {
  storeKey: SK;
  reduce: DispatchReducer<State, Action>;
  preprocess?: (state: State, actions: Action[]) => Action[];
  channel: string;
  schema: z.ZodType<DispatchFrame<Key, Action>>;
  isUndoable?: (action: Action) => boolean;
  kindOf?: (actions: Action[]) => string;
  coalesceWindow?: TimeSpan;
  stackCap?: number;
  equal?: (a: State, b: State, key: Key) => boolean;
}

export const createUndoableStore = <
  Key extends record.Key,
  State extends Shape,
  Action,
  SK extends string,
  ScopedStore extends Store & Record<SK, UndoableUnaryStore<Key, State, Action>>,
>(
  config: CreateUndoableStoreParams<Key, State, Action, SK>,
): UnaryStoreConfig<ScopedStore> => {
  const { storeKey, channel, schema, ...storeOpts } = config;
  const remoteListener: ChannelListener<
    ScopedStore,
    z.ZodType<DispatchFrame<Key, Action>>
  > = {
    channel,
    schema,
    onChange: ({ changed, store, client }) => {
      if (changed.sessionKey === client?.key) return;
      store[storeKey].applyRemote(changed.key, changed.actions);
    },
  };
  return {
    listeners: [remoteListener],
    factory: (handleError) =>
      new UndoableStore<Key, State, Action>({ ...storeOpts, handleError }),
  };
};
