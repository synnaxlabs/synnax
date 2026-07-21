// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  compare,
  destructor,
  errors,
  id,
  type record,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import type z from "zod";

import { type cache } from "@/cache";
import { Table } from "@/cache/table";

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

export interface Reducer<State extends cache.Data, Action> {
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

/**
 * A staged set of dispatches committed atomically as one undoable. Commit
 * resolves false when there is nothing to send; a send failure rolls back the
 * staged state and rethrows.
 */
export interface Transaction<Action> {
  add: (actions: Action | Action[]) => void;
  commit: () => Promise<boolean>;
  abort: () => void;
}

/** Sends an already-processed action list to the server. Supplied by the
 *  caller so the controller stays free of network concerns. */
export type Send<Action> = (actions: Action[]) => Promise<void>;

/**
 * Network send used by the dispatch/undo/redo loops. Receives the
 * controller-generated dispatch key that was registered as outstanding before
 * the call, so the broadcast echo is recognized as own.
 */
export type SendDispatch<Action> = (
  actions: Action[],
  dispatchKey: string,
) => Promise<void>;

/** Rewrites an action batch against the current document before replay. */
export type Preprocess<State, Action> = (state: State, actions: Action[]) => Action[];

/** Options accepted by domain-client dispatch methods. */
export interface Options<State, Action> {
  /**
   * Per-call action rewrite requiring caller context the domain client lacks
   * (e.g. diagram geometry). Runs against the cached doc before replay.
   */
  preprocess?: Preprocess<State, Action>;
}

/** The action-dispatch surface a document domain client exposes. */
export interface Domain<Key extends record.Key, State extends cache.Data, Action> {
  dispatch(
    key: Key,
    actions: Action | Action[],
    opts?: Options<State, Action>,
  ): Promise<boolean>;
  undo(key: Key): Promise<boolean>;
  redo(key: Key): Promise<boolean>;
  hasUndo(key: Key): boolean;
  hasRedo(key: Key): boolean;
  onUndoStateChange(callback: () => void, key?: Key): destructor.Destructor;
  beginTransaction(key: Key, kind?: string): Transaction<Action>;
}

/**
 * The canonical wire shape for a broadcast action frame. The consumer's
 * `schema` must produce this shape (use a zod transform if the server emits
 * differently-named fields). `seq` is a per-key monotonic sequence stamped by
 * the originating server node and is used to skip echoes that are stale
 * relative to the high-water mark — see `applyRemote`. `dispatchKey` is the
 * client-generated batch identifier the originator registered as outstanding
 * before sending; it is used to recognize own echoes race-safely so a
 * redundant reduce can be skipped when no foreign action interleaved.
 */
export interface Frame<Key, Action> {
  key: Key;
  dispatchKey: string;
  seq: number;
  actions: Action[];
}

export interface ControllerParams<
  Key extends record.Key,
  State extends cache.Data,
  Action,
> {
  /** The table holding the documents this controller dispatches over. */
  store: Table<Key, State>;
  /** Reports listener errors that have no caller to throw to. */
  onError: (error: Error) => void;
  reduce: Reducer<State, Action>;
  preprocess?: (state: State, actions: Action[]) => Action[];
  isUndoable?: (action: Action) => boolean;
  kindOf?: (actions: Action[]) => string;
  coalesceWindow?: TimeSpan;
  stackCap?: number;
}

/**
 * Owns the action-dispatch loop for one document table: local replay with
 * rollback, per-doc undo/redo stacks, echo recognition, and remote-frame
 * application. Operates on a plain cache table; all dispatch bookkeeping is
 * session-local and dropped when a document's row is deleted (tombstoned).
 */
export class Controller<Key extends record.Key, State extends cache.Data, Action> {
  private readonly docs: Table<Key, State>;
  private readonly undos: Table<Key, UndoState<Action>>;
  private readonly params: Required<
    Omit<ControllerParams<Key, State, Action>, "store">
  >;
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

  constructor(opts: ControllerParams<Key, State, Action>) {
    this.params = {
      onError: opts.onError,
      reduce: opts.reduce,
      preprocess: opts.preprocess ?? ((_, a) => a),
      isUndoable: opts.isUndoable ?? (() => true),
      kindOf: opts.kindOf ?? (() => "default"),
      coalesceWindow: opts.coalesceWindow ?? DEFAULT_COALESCE_WINDOW,
      stackCap: opts.stackCap ?? DEFAULT_STACK_CAP,
    };
    this.docs = opts.store;
    // Undo states are rebuilt objects on every update; comparing them would
    // be pure waste, so equality is disabled.
    this.undos = new Table<Key, UndoState<Action>>(this.params.onError, () => false);
    // Stacks are session-local: when a document is deleted (tombstoned), its
    // dispatch bookkeeping goes with it.
    this.docs.subscribe((event) => {
      if (event.variant === "delete") this.drop(event.key);
    });
  }

  private drop(key: Key): void {
    this.undos.delete(key);
    this.lastAppliedSeq.delete(key);
    this.outstandingDispatches.delete(key);
  }

  private updateUndo(
    key: Key,
    fn: (s: UndoState<Action>) => UndoState<Action>,
  ): destructor.Destructor {
    return this.undos.set(key, (prev) => fn(prev ?? ZERO_UNDO<Action>()));
  }

  /**
   * Run preprocess + reduce against the cached doc and write the result.
   * Returns the reducer output and a rollback that restores the prior doc
   * state. Returns null if the doc isn't cached.
   */
  replay(
    key: Key,
    actions: Action[],
    opts: { skipPreprocess?: boolean } = {},
  ): ReplayResult<Action> | null {
    const current = this.docs.get(key);
    if (current == null) return null;
    const processed = opts.skipPreprocess
      ? actions
      : this.params.preprocess(current, actions);
    const { next, inverse, targets } = this.params.reduce(current, processed);
    return {
      processed,
      inverse,
      targets,
      changed: next !== current,
      rollback: this.docs.set(key, next),
    };
  }

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
  ): destructor.Destructor {
    if (targets.length === 0) return destructor.NOOP;
    const undoableForward = forward.filter(this.params.isUndoable);
    if (undoableForward.length === 0) return destructor.NOOP;
    const entry: StackEntry<Action> = {
      forward: undoableForward,
      inverse,
      kind: kindOverride ?? this.params.kindOf(forward),
      ts: TimeStamp.now(),
      targets,
    };
    return this.updateUndo(key, (s) => ({
      ...s,
      undo: pushOnto(s.undo, entry, this.params.coalesceWindow, this.params.stackCap),
      redo: [],
    }));
  }

  // Walks the chosen stack from the top, skipping entries whose targets were
  // remote-touched after the entry's push ts. Caps the walk at
  // STALE_AUTO_ADVANCE_CAP and drops a fully stale tail. On hit, returns the
  // entry's reversal actions (inverse for undo, forward for redo) and a
  // commit that moves the entry to the opposite stack.
  private prepareReversal(key: Key, side: "undo" | "redo"): Reversal<Action> | null {
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
        this.updateUndo(key, (s) => ({
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
        this.updateUndo(key, (s) => ({
          ...s,
          [side]: s[side].slice(0, idx),
          [other]: [...s[other], entry],
        })),
    };
  }

  // Reduce + send with rollback: replay locally, run the per-op stack
  // mutation, register the batch as outstanding *before* the network send
  // (the broadcast echo can race the HTTP response), and roll back stack and
  // state when the send fails. Returns false when the doc isn't cached.
  private async applySend(
    key: Key,
    actions: Action[],
    commitStack: (r: ReplayResult<Action>) => destructor.Destructor,
    send: SendDispatch<Action>,
    preprocess?: Preprocess<State, Action>,
  ): Promise<boolean> {
    const current = this.docs.get(key);
    if (current == null) return false;
    const processed = preprocess == null ? actions : preprocess(current, actions);
    const r = this.replay(key, processed, { skipPreprocess: true });
    if (r == null) return false;
    const stackRollback = commitStack(r);
    // A replay that left the state untouched has nothing for the server: no
    // document change, no broadcast worth echoing. The stack mutation still
    // runs above so undo/redo consume their entry instead of pinning it.
    if (!r.changed) return true;
    const dispatchKey = id.create();
    this.registerOutstandingDispatch(key, dispatchKey);
    try {
      await send(r.processed, dispatchKey);
      return true;
    } catch (e) {
      stackRollback();
      r.rollback();
      throw errors.fromUnknown(e);
    }
  }

  /**
   * Apply an action batch to the cached doc and send it to the server,
   * recording an undoable entry. Returns false without side effects when the
   * doc isn't cached. Rolls back the local apply and rethrows on send failure.
   */
  async dispatch(
    key: Key,
    actions: Action[],
    send: SendDispatch<Action>,
    preprocess?: Preprocess<State, Action>,
  ): Promise<boolean> {
    if (actions.length === 0) return true;
    return await this.applySend(
      key,
      actions,
      ({ processed, inverse, targets }) =>
        this.recordEntry(key, processed, inverse, targets),
      send,
      preprocess,
    );
  }

  /**
   * Apply and send the top live undo entry, moving it to the redo stack.
   * Returns false when nothing is undoable. Rolls back and rethrows on send
   * failure.
   */
  async undo(key: Key, send: SendDispatch<Action>): Promise<boolean> {
    const prepared = this.prepareUndo(key);
    if (prepared == null) return false;
    return await this.applySend(key, prepared.actions, () => prepared.commit(), send);
  }

  /** Mirror of {@link undo} for the redo stack. */
  async redo(key: Key, send: SendDispatch<Action>): Promise<boolean> {
    const prepared = this.prepareRedo(key);
    if (prepared == null) return false;
    return await this.applySend(key, prepared.actions, () => prepared.commit(), send);
  }

  /**
   * Stage actions committed atomically as one undoable entry, registering the
   * batch as outstanding before the send so the broadcast echo is recognized
   * as own.
   */
  transaction(
    key: Key,
    send: SendDispatch<Action>,
    kind?: string,
  ): Transaction<Action> {
    return this.beginTransaction(
      key,
      async (actions) => {
        const dispatchKey = id.create();
        this.registerOutstandingDispatch(key, dispatchKey);
        await send(actions, dispatchKey);
      },
      kind,
    );
  }

  prepareUndo(key: Key): Reversal<Action> | null {
    return this.prepareReversal(key, "undo");
  }

  prepareRedo(key: Key): Reversal<Action> | null {
    return this.prepareReversal(key, "redo");
  }

  beginTransaction(key: Key, send: Send<Action>, kind?: string): Transaction<Action> {
    const initial = this.docs.get(key);
    const accumulated: Action[] = [];
    let done = false;
    return {
      add: (actions) => {
        if (done) throw new Error("transaction finalized");
        if (initial == null) return;
        const arr = Array.isArray(actions) ? actions : [actions];
        const r = this.replay(key, arr);
        if (r != null) accumulated.push(...r.processed);
      },
      commit: async () => {
        if (done) return false;
        done = true;
        if (accumulated.length === 0 || initial == null) return false;
        // Inverse must be computed against the pre-transaction snapshot so
        // undo restores to where the user started, not the last-add post-state.
        const { inverse, targets } = this.params.reduce(initial, accumulated);
        const stackRollback = this.recordEntry(
          key,
          accumulated,
          inverse,
          targets,
          kind,
        );
        try {
          await send(accumulated);
          return true;
        } catch (e) {
          stackRollback();
          this.docs.set(key, initial);
          throw errors.fromUnknown(e);
        }
      },
      abort: () => {
        if (done) return;
        done = true;
        if (initial != null) this.docs.set(key, initial);
      },
    };
  }

  markRemoteTouched(
    key: Key,
    targets: readonly string[],
    ts: TimeStamp = TimeStamp.now(),
  ): destructor.Destructor {
    if (targets.length === 0) return destructor.NOOP;
    return this.updateUndo(key, (s) => {
      const remoteTouched = { ...s.remoteTouched };
      for (const t of targets) remoteTouched[t] = ts;
      return { ...s, remoteTouched };
    });
  }

  /** Whether any locally replayed dispatch for the key still awaits its echo. */
  hasOutstanding(key: Key): boolean {
    return (this.outstandingDispatches.get(key)?.size ?? 0) > 0;
  }

  /**
   * Register a dispatch as outstanding so when its broadcast echo arrives it
   * can be recognized as own and a redundant reduce skipped. Called by the
   * dispatcher *before* the network send so the registration always wins the
   * race against a fast echo. A registration whose echo has already been
   * applied (seq advanced past it) is silently dropped; that's the safe
   * fallback when the echo arrives before the dispatch promise resolves.
   */
  registerOutstandingDispatch(key: Key, dispatchKey: string): void {
    let bucket = this.outstandingDispatches.get(key);
    if (bucket == null) {
      bucket = new Map();
      this.outstandingDispatches.set(key, bucket);
    }
    bucket.set(dispatchKey, { disturbed: false });
  }

  /**
   * Apply a remote frame. The frame is dropped if its seq does not exceed the
   * high-water mark previously applied for this key. If `dispatchKey` matches
   * an entry the originator registered as outstanding, these actions were
   * already applied via replay: if no foreign action interleaved during the
   * outstanding window, skip the reduce; otherwise re-run it to recover from
   * the foreign overwrite. A foreign frame (one whose dispatchKey is unknown)
   * always reduces and marks every outstanding own dispatch with a greater
   * seq as needing recovery.
   */
  applyRemote(key: Key, seq: number, dispatchKey: string, actions: Action[]): void {
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
    const { next, targets } = this.params.reduce(current, actions);
    this.docs.set(key, next);
    if (!isOwnEcho) this.markRemoteTouched(key, targets);
  }

  hasUndo(key: Key): boolean {
    const s = this.undos.get(key);
    return (s?.undo.length ?? 0) > 0;
  }

  hasRedo(key: Key): boolean {
    const s = this.undos.get(key);
    return (s?.redo.length ?? 0) > 0;
  }

  onUndoStateChange(callback: () => void, key?: Key): destructor.Destructor {
    return this.undos.subscribe(() => callback(), key);
  }

  /**
   * Builds a channel listener that applies broadcast action frames from the
   * given channel to this controller. Registered onto the owning table.
   */
  listener(
    channel: string,
    schema: z.ZodType<Frame<Key, Action>>,
  ): cache.ChannelListener<z.ZodType<Frame<Key, Action>>> {
    return {
      channel,
      schema,
      onChange: ({ key, seq, dispatchKey, actions }) => {
        this.applyRemote(key, seq, dispatchKey, actions);
      },
    };
  }
}
