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
import { type Data } from "@/flux/base/types";
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

export interface DispatchReducer<State extends Data, Action> {
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
  // changed reports whether the replay produced a different state. Reducers
  // return the input state by reference when no action touched it, so a false
  // value means the entire vector was a no-op and need not reach the server.
  changed: boolean;
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
 *  commit so the substrate can bind client/dispatchKey at hook-call time. */
export type DispatchSend<Action> = (actions: Action[]) => Promise<void>;

/**
 * The canonical wire shape for a broadcast action frame. The consumer's
 * `schema` must produce this shape (use a zod transform if the server emits
 * differently-named fields). `seq` is a per-key monotonic sequence stamped by
 * the originating server node and is used by the store to skip echoes that
 * are stale relative to its high-water mark — see `applyRemote`. `dispatchKey`
 * is the client-generated batch identifier the originator registered as
 * outstanding before sending; the substrate uses it to recognize own echoes
 * race-safely so it can skip a redundant reduce when no foreign action
 * interleaved.
 */
export interface DispatchFrame<Key, Action> {
  key: Key;
  dispatchKey: string;
  seq: number;
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
  State extends Data,
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
   * Return the top live undo, dropping stale entries from the tail as it
   * walks past them. Null when no live entry exists.
   */
  prepareUndo(key: Key): Reversal<Action> | null;
  /** Mirror of prepareUndo for the redo stack. */
  prepareRedo(key: Key): Reversal<Action> | null;
  /**
   * Stage actions to commit atomically as one undoable. The substrate
   * supplies `send` so the store doesn't depend on the Synnax client.
   */
  beginTransaction(
    key: Key,
    send: DispatchSend<Action>,
    kind?: string,
  ): Transaction<Action>;
  /**
   * Register a dispatch as outstanding so when its broadcast echo arrives the
   * substrate can recognize it as own and skip a redundant reduce. Called by
   * the dispatcher *before* the network send so the registration always wins
   * the race against a fast echo. A registration whose echo has already been
   * applied (seq advanced past it) is silently dropped; that's the safe
   * fallback when the echo arrives before the dispatch promise resolves.
   */
  registerOutstandingDispatch(key: Key, dispatchKey: string): void;
  /**
   * Apply a remote frame. The frame is dropped if its seq does not exceed the
   * high-water mark previously applied for this key. If `dispatchKey` matches
   * an entry the originator registered as outstanding, the substrate has
   * already applied these actions via replay: if no foreign action
   * interleaved during the outstanding window, skip the reduce; otherwise
   * re-run it to recover from the foreign overwrite. A foreign frame (one
   * whose dispatchKey is unknown) always reduces and marks every outstanding
   * own dispatch with a greater seq as needing recovery.
   */
  applyRemote(key: Key, seq: number, dispatchKey: string, actions: Action[]): void;
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

export interface UndoableStoreConfig<
  Key extends record.Key,
  State extends Data,
  Action,
> {
  handleError: status.ErrorHandler;
  reduce: DispatchReducer<State, Action>;
  equal?: (a: State, b: State, key: Key) => boolean;
  preprocess?: (state: State, actions: Action[]) => Action[];
  isUndoable?: (action: Action) => boolean;
  kindOf?: (actions: Action[]) => string;
  coalesceWindow?: TimeSpan;
  stackCap?: number;
}

class UndoableStore<Key extends record.Key, State extends Data, Action> {
  private readonly docs: ScopedUnaryStore<Key, State>;
  private readonly undos: ScopedUnaryStore<Key, UndoState<Action>>;
  private readonly config: Required<UndoableStoreConfig<Key, State, Action>>;
  // lastAppliedSeq tracks the highest broadcast sequence number applied for
  // each key. Echoes whose seq does not exceed this are stale (a fresher
  // value already overwrote them on the server) and are dropped. A seq of 0
  // is treated as "unstamped" and always applies — that covers frames from
  // servers that predate the field.
  private readonly lastAppliedSeq = new Map<Key, number>();
  // outstandingDispatches tracks per-key dispatches the originator has
  // applied locally via replay but whose broadcast echo has not yet arrived.
  // Each entry's `disturbed` flag is flipped to true when a foreign echo
  // lands during the outstanding window; on own-echo arrival, an entry that
  // is not disturbed skips the reduce (the local replay was authoritative)
  // and an entry that is disturbed re-runs to recover from the foreign
  // overwrite.
  private readonly outstandingDispatches = new Map<
    Key,
    Map<string, { disturbed: boolean }>
  >();

  constructor(opts: UndoableStoreConfig<Key, State, Action>) {
    this.config = {
      handleError: opts.handleError,
      reduce: opts.reduce,
      equal: opts.equal ?? (() => false),
      preprocess: opts.preprocess ?? ((_, a) => a),
      isUndoable: opts.isUndoable ?? (() => true),
      kindOf: opts.kindOf ?? (() => "default"),
      coalesceWindow: opts.coalesceWindow ?? DEFAULT_COALESCE_WINDOW,
      stackCap: opts.stackCap ?? DEFAULT_STACK_CAP,
    };
    const { handleError, equal } = this.config;
    this.docs = new ScopedUnaryStore<Key, State>(handleError, equal);
    this.undos = new ScopedUnaryStore<Key, UndoState<Action>>(handleError);
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
    const current = this.docs.get(key);
    if (current == null) return null;
    const processed = opts.skipPreprocess
      ? actions
      : this.config.preprocess(current, actions);
    const { next, inverse, targets } = this.config.reduce(current, processed);
    return {
      processed,
      inverse,
      targets,
      changed: next !== current,
      rollback: this.docs.set(scope, key, next),
    };
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
    const undoableForward = forward.filter(this.config.isUndoable);
    if (undoableForward.length === 0) return destructor.NOOP;
    const entry: StackEntry<Action> = {
      forward: undoableForward,
      inverse,
      kind: kindOverride ?? this.config.kindOf(forward),
      ts: TimeStamp.now(),
      targets,
    };
    return this.updateUndo(scope, key, (s) => ({
      ...s,
      undo: pushOnto(s.undo, entry, this.config.coalesceWindow, this.config.stackCap),
      redo: [],
    }));
  }

  // Walks the chosen stack from the top, skipping entries whose targets were
  // remote-touched after the entry's push ts. Caps the walk at
  // STALE_AUTO_ADVANCE_CAP and drops a fully stale tail. On hit, returns the
  // entry's reversal actions (inverse for undo, forward for redo) and a
  // commit that moves the entry to the opposite stack.
  private prepareReversal(
    scope: string,
    key: Key,
    side: "undo" | "redo",
  ): Reversal<Action> | null {
    const state = this.undos.get(key);
    if (state == null) return null;
    const stack = state[side];
    if (stack.length === 0) return null;
    let stale = 0;
    let idx = -1;
    for (let i = stack.length - 1; i >= 0; i--) {
      const e = stack[i];
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
          [side]: s[side].slice(0, s[side].length - stale),
        }));
      return null;
    }
    const entry = stack[idx];
    const other = side === "undo" ? "redo" : "undo";
    return {
      actions: side === "undo" ? entry.inverse : entry.forward,
      commit: () =>
        this.updateUndo(scope, key, (s) => ({
          ...s,
          [side]: s[side].slice(0, idx),
          [other]: [...s[other], entry],
        })),
    };
  }

  prepareUndo(scope: string, key: Key): Reversal<Action> | null {
    return this.prepareReversal(scope, key, "undo");
  }

  prepareRedo(scope: string, key: Key): Reversal<Action> | null {
    return this.prepareReversal(scope, key, "redo");
  }

  beginTransaction(
    scope: string,
    key: Key,
    send: DispatchSend<Action>,
    kind?: string,
  ): Transaction<Action> {
    const initial = this.docs.get(key);
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
        const { inverse, targets } = this.config.reduce(initial, accumulated);
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

  registerOutstandingDispatch(key: Key, dispatchKey: string): void {
    let bucket = this.outstandingDispatches.get(key);
    if (bucket == null) {
      bucket = new Map();
      this.outstandingDispatches.set(key, bucket);
    }
    bucket.set(dispatchKey, { disturbed: false });
  }

  applyRemote(
    scope: string,
    key: Key,
    seq: number,
    dispatchKey: string,
    actions: Action[],
  ): void {
    if (seq !== 0) {
      const last = this.lastAppliedSeq.get(key) ?? 0;
      if (seq <= last) return;
      this.lastAppliedSeq.set(key, seq);
    }
    const bucket = this.outstandingDispatches.get(key);
    const ownEntry = bucket?.get(dispatchKey);
    const isOwnEcho = ownEntry != null;
    if (ownEntry != null) {
      bucket?.delete(dispatchKey);
      // Local replay was authoritative; nothing interleaved.
      if (!ownEntry.disturbed) return;
      // Fall through to reduce: a foreign echo overwrote state during the
      // outstanding window, so we must re-apply to recover.
    } else if (bucket != null)
      // Foreign echo: any outstanding own dispatch represents a local replay
      // that the server has not yet broadcast back. We don't know each
      // entry's server-stamped seq yet (the echo carrying it hasn't arrived),
      // so we can't tell whether the foreign was applied before or after it
      // on the server — mark every outstanding entry as disturbed. Entries
      // whose seq turns out to be greater than this foreign's seq genuinely
      // need recovery; entries whose seq was lower will re-run idempotently
      // and converge to the same value.
      for (const entry of bucket.values()) entry.disturbed = true;
    const current = this.docs.get(key);
    if (current == null) return;
    const { next, targets } = this.config.reduce(current, actions);
    this.docs.set(scope, key, next);
    if (!isOwnEcho) this.markRemoteTouched(scope, key, targets);
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
    const s = this.undos.get(key);
    return (s?.undo.length ?? 0) > 0;
  }

  hasRedo(key: Key): boolean {
    const s = this.undos.get(key);
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
      registerOutstandingDispatch: (key, dispatchKey) =>
        this.registerOutstandingDispatch(key, dispatchKey),
      applyRemote: (key, seq, dispatchKey, actions) =>
        this.applyRemote(scope, key, seq, dispatchKey, actions),
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
  State extends Data,
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
  State extends Data,
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
    onChange: ({ changed, store }) => {
      store[storeKey].applyRemote(
        changed.key,
        changed.seq,
        changed.dispatchKey,
        changed.actions,
      );
    },
  };
  return {
    listeners: [remoteListener],
    factory: (handleError) =>
      new UndoableStore<Key, State, Action>({ ...storeOpts, handleError }),
  };
};
