// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  array,
  compare,
  deep,
  destructor,
  errors,
  id,
  type record,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import type z from "zod";

import { type query } from "@/query";
import { Table } from "@/query/table";

const DEFAULT_COALESCE_WINDOW = TimeSpan.milliseconds(500);
const DEFAULT_STACK_CAP = 200;
const STALE_AUTO_ADVANCE_CAP = 10;

export interface StackEntry<Action> {
  forward: Action[];
  inverse: Action[];
  kind: string;
  /** Wall-clock push time, used only for coalescing. */
  ts: TimeStamp;
  /** Tick of the earliest push, used to order the entry against remote touches. */
  tick: number;
  targets: readonly string[];
}

interface UndoState<Action> {
  undo: StackEntry<Action>[];
  redo: StackEntry<Action>[];
  remoteTouched: Record<string, number>;
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
          tick: top.tick,
          targets: top.targets,
        }
      : null;
  const out = merged != null ? [...stack.slice(0, -1), merged] : [...stack, next];
  return out.length > cap ? out.slice(out.length - cap) : out;
};

export interface Reducer<State extends query.Data, Action> {
  (
    state: State,
    actions: Action[],
  ): { next: State; inverse: Action[]; targets: readonly string[] };
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

/**
 * Network send used by the dispatch/undo/redo loops. Receives the
 * controller-generated dispatch key, held as pending before the call, so the broadcast
 * echo is recognized as own.
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
export interface Domain<Key extends record.Key, State extends query.Data, Action> {
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
 * The canonical wire shape for a broadcast action frame. The consumer's `schema` must
 * produce this shape (use a zod transform if the server emits differently-named
 * fields). `seq` is a per-key monotonic sequence stamped by the originating server node
 * and is used to skip echoes that are stale relative to the high-water mark — see
 * `applyRemote`. `dispatchKey` is the client-generated batch identifier the originator
 * holds as pending before sending, so it recognizes its own echoes.
 */
export interface Frame<Key, Action> {
  key: Key;
  dispatchKey: string;
  seq: number;
  actions: Action[];
}

export interface ControllerParams<
  Key extends record.Key,
  State extends query.Data,
  Action,
> {
  /** The table holding the documents this controller dispatches over. */
  store: Table<Key, State>;
  /** Reports listener errors that have no caller to throw to. */
  onError: (error: Error) => void;
  /**
   * Subscribes to the connection epoch of the cache that owns the store. A new epoch
   * means echoes may have been lost, and epoch 0 means the cache was reset.
   */
  onEpoch: (callback: (epoch: number) => void) => destructor.Destructor;
  reduce: Reducer<State, Action>;
  isUndoable?: (action: Action) => boolean;
  kindOf?: (actions: Action[]) => string;
  /**
   * Returns the document carried by a create action, undefined otherwise.
   * Lets remote frames for never-cached documents ingest instead of drop.
   */
  createOf?: (action: Action) => State | undefined;
  coalesceWindow?: TimeSpan;
  stackCap?: number;
  /** Clock stamping entries for the coalesce window. Defaults to TimeStamp.now. */
  now?: () => TimeStamp;
}

/** A dispatch applied to the cached doc whose echo has not arrived. */
interface Pending<Action> {
  dispatchKey: string;
  actions: Action[];
  /** The epoch in which the send started. */
  epoch: number;
  /** Whether the Core applied the dispatch. */
  accepted: boolean;
}

/** The dispatch state of a doc that has pending entries. */
interface Rebase<State, Action> {
  /** The doc as the Core holds it, without the pending entries. */
  base: State;
  pending: Array<Pending<Action>>;
  /** The doc this controller wrote last. A different doc means another writer. */
  written: State;
}

interface Applied<Action> {
  processed: Action[];
  inverse: Action[];
  targets: readonly string[];
}

/**
 * Owns the action-dispatch loop for one document table: local apply, per-doc undo/redo
 * stacks, echo recognition, and remote-frame application. The cached doc of a key with
 * pending dispatches is its base, as the Core holds it, with the pending dispatches
 * applied on top. A foreign frame applies to the base, and the pending dispatches apply
 * again on the result, so the doc stays in the order the Core applies them. A failed
 * send removes its dispatch the same way. All dispatch bookkeeping is session-local and
 * dropped when a document is deleted.
 */
export class Controller<Key extends record.Key, State extends query.Data, Action> {
  private readonly docs: Table<Key, State>;
  private readonly undos: Table<Key, UndoState<Action>>;
  private readonly params: Required<
    Omit<ControllerParams<Key, State, Action>, "store" | "onEpoch">
  >;
  // lastAppliedSeq tracks the highest broadcast sequence number applied for each key.
  // Echoes whose seq does not exceed this are stale and are dropped. A seq of 0 is
  // "unstamped" and always applies.
  private readonly lastAppliedSeq = new Map<Key, number>();
  private readonly rebases = new Map<Key, Rebase<State, Action>>();
  /** Callers of settled, waiting for a doc to hold no pending entry. */
  private readonly settling = new Map<Key, Array<() => void>>();
  private epoch = 0;
  // Monotonic counter ordering entry pushes against remote touches. Program
  // order is the ground truth here; the wall clock ties within a millisecond
  // and steps on sleep/wake or NTP resync, so it cannot be trusted for this.
  private tick = 0;

  constructor(opts: ControllerParams<Key, State, Action>) {
    this.params = {
      onError: opts.onError,
      reduce: opts.reduce,
      isUndoable: opts.isUndoable ?? (() => true),
      kindOf: opts.kindOf ?? (() => "default"),
      createOf: opts.createOf ?? (() => undefined),
      coalesceWindow: opts.coalesceWindow ?? DEFAULT_COALESCE_WINDOW,
      stackCap: opts.stackCap ?? DEFAULT_STACK_CAP,
      now: opts.now ?? (() => TimeStamp.now()),
    };
    this.docs = opts.store;
    // Undo states are rebuilt objects on every update; comparing them would
    // be pure waste, so equality is disabled.
    this.undos = new Table<Key, UndoState<Action>>({
      onError: this.params.onError,
      equal: () => false,
    });
    // Stacks are session-local: when a document is deleted (tombstoned), its
    // dispatch bookkeeping goes with it.
    this.docs.subscribe((event) => {
      if (event.variant === "delete") this.drop(event.key);
    });
    opts.onEpoch((epoch) => this.changeEpoch(epoch));
  }

  private drop(key: Key): void {
    this.undos.delete(key);
    this.lastAppliedSeq.delete(key);
    this.release(key);
  }

  private changeEpoch(epoch: number): void {
    this.epoch = epoch;
    // Sequence numbers restart with the Core.
    this.lastAppliedSeq.clear();
    if (epoch === 0) {
      [...this.rebases.keys()].forEach((key) => this.release(key));
      this.undos.delete(() => true);
      return;
    }
    // The echoes of accepted dispatches may have been lost in the gap.
    for (const [key, r] of [...this.rebases])
      for (const p of r.pending.filter(({ accepted }) => accepted))
        this.settle(key, r, p.dispatchKey, p.actions);
  }

  private updateUndo(
    key: Key,
    fn: (s: UndoState<Action>) => UndoState<Action>,
  ): destructor.Destructor {
    return this.undos.set(key, (prev) => fn(prev ?? ZERO_UNDO<Action>()));
  }

  /** Takes a doc that another writer put in the table as the new base. */
  private sync(key: Key, r: Rebase<State, Action>): boolean {
    const doc = this.docs.get(key);
    if (doc == null || doc === r.written) return false;
    r.base = doc;
    return true;
  }

  /** Writes the base with the pending entries applied. */
  private write(key: Key, r: Rebase<State, Action>): State {
    const doc = r.pending.reduce(
      (state, { actions }) => this.params.reduce(state, actions).next,
      r.base,
    );
    return this.set(key, r, doc);
  }

  private set(key: Key, r: Rebase<State, Action>, doc: State): State {
    this.docs.set(key, doc);
    r.written = this.docs.get(key) ?? doc;
    return r.written;
  }

  /** Returns the cached doc with every pending entry applied. */
  private current(key: Key): State | undefined {
    const r = this.rebases.get(key);
    if (r == null || !this.sync(key, r)) return this.docs.get(key);
    return this.write(key, r);
  }

  /** Holds the entry as pending, if it is not yet, and writes the doc with it. */
  private track(key: Key, current: State, entry: Pending<Action>, next: State): void {
    let r = this.rebases.get(key);
    if (r == null) {
      r = { base: current, pending: [], written: current };
      this.rebases.set(key, r);
    }
    if (!r.pending.includes(entry)) r.pending.push(entry);
    this.set(key, r, next);
  }

  /**
   * Applies actions that the Core applied to the base, and removes the pending entry
   * with the dispatch key. Returns whether an entry had the key, and the targets.
   */
  private settle(
    key: Key,
    r: Rebase<State, Action>,
    dispatchKey: string,
    actions: Action[],
  ): { own: boolean; targets: readonly string[] } {
    const replaced = this.sync(key, r);
    const { next, targets } = this.params.reduce(r.base, actions);
    r.base = next;
    const i = r.pending.findIndex((p) => p.dispatchKey === dispatchKey);
    // The doc already holds the first pending entry. It changes only when the echo
    // carries actions that the entry does not, such as fields the Core derives.
    const held = i === 0 && !replaced && deep.equal(r.pending[0].actions, actions);
    if (i !== -1) r.pending.splice(i, 1);
    if (!held) this.write(key, r);
    if (r.pending.length === 0) this.release(key);
    return { own: i !== -1, targets };
  }

  /** Removes a pending entry that the Core did not apply. */
  private discard(key: Key, dispatchKey: string): void {
    const r = this.rebases.get(key);
    const i = r?.pending.findIndex((p) => p.dispatchKey === dispatchKey) ?? -1;
    if (r == null || i === -1) return;
    this.sync(key, r);
    r.pending.splice(i, 1);
    this.write(key, r);
    if (r.pending.length === 0) this.release(key);
  }

  /** Marks a pending entry as applied by the Core. */
  private accept(key: Key, dispatchKey: string): void {
    const r = this.rebases.get(key);
    const p = r?.pending.find((p) => p.dispatchKey === dispatchKey);
    if (r == null || p == null) return;
    p.accepted = true;
    // The echo may have been lost when the epoch changed during the send.
    if (p.epoch !== this.epoch) this.settle(key, r, dispatchKey, p.actions);
  }

  private pending(dispatchKey: string, actions: Action[] = []): Pending<Action> {
    return { dispatchKey, actions, epoch: this.epoch, accepted: false };
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
      ts: this.params.now(),
      tick: ++this.tick,
      targets,
    };
    return this.updateUndo(key, (s) => ({
      ...s,
      undo: pushOnto(s.undo, entry, this.params.coalesceWindow, this.params.stackCap),
      redo: [],
    }));
  }

  // Walks the chosen stack from the top, skipping entries whose targets were
  // remote-touched after the entry was pushed. Caps the walk at
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
      if (e.targets.some((t) => (state.remoteTouched[t] ?? 0) > e.tick)) {
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

  // Applies the actions to the cached doc, runs the per-op stack mutation, adds a
  // pending entry *before* the network send (the broadcast echo can race the
  // response), and removes the entry and rolls back the stack when the send fails.
  // Returns false when the doc isn't cached.
  private async applySend(
    key: Key,
    actions: Action[],
    commitStack: (applied: Applied<Action>) => destructor.Destructor,
    send: SendDispatch<Action>,
    preprocess?: Preprocess<State, Action>,
  ): Promise<boolean> {
    const current = this.current(key);
    if (current == null) return false;
    const processed = preprocess == null ? actions : preprocess(current, actions);
    const { next, inverse, targets } = this.params.reduce(current, processed);
    const stackRollback = commitStack({ processed, inverse, targets });
    // A no-op has nothing for the server. The stack mutation still runs above so
    // undo/redo consume their entry instead of pinning it.
    if (next === current) return true;
    const dispatchKey = id.create();
    this.track(key, current, this.pending(dispatchKey, processed), next);
    try {
      await send(processed, dispatchKey);
    } catch (e) {
      stackRollback();
      this.discard(key, dispatchKey);
      throw errors.fromUnknown(e);
    }
    this.accept(key, dispatchKey);
    return true;
  }

  /**
   * Apply an action batch to the cached doc and send it to the server,
   * recording an undoable entry. Returns false without side effects when the
   * doc isn't cached. Removes the local apply and rethrows on send failure.
   */
  async dispatch(
    key: Key,
    actions: Action[],
    send: SendDispatch<Action>,
    opts: Options<State, Action> = {},
  ): Promise<boolean> {
    if (actions.length === 0) return true;
    return await this.applySend(
      key,
      actions,
      ({ processed, inverse, targets }) =>
        this.recordEntry(key, processed, inverse, targets),
      send,
      opts.preprocess,
    );
  }

  /**
   * Apply and send the top live undo entry, moving it to the redo stack. Returns false
   * when nothing is undoable. Rolls back and rethrows on send failure.
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
   * Sends a create for the given keys. The cached docs among them hold their create as
   * pending, so its echo does not revert edits made before the echo arrives.
   * @returns what the send returns.
   * @throws the error of the send.
   */
  async create<T>(keys: Key[], send: (dispatchKey: string) => Promise<T>): Promise<T> {
    const dispatchKey = id.create();
    const tracked = keys.filter((key) => {
      const current = this.current(key);
      if (current == null) return false;
      this.track(key, current, this.pending(dispatchKey), current);
      return true;
    });
    let res: T;
    try {
      res = await send(dispatchKey);
    } catch (e) {
      tracked.forEach((key) => this.discard(key, dispatchKey));
      throw errors.fromUnknown(e);
    }
    tracked.forEach((key) => this.accept(key, dispatchKey));
    return res;
  }

  /**
   * Stage actions committed atomically as one undoable entry. Each add applies to the
   * cached doc at once, and the staged actions stay pending until the commit settles.
   */
  transaction(
    key: Key,
    send: SendDispatch<Action>,
    kind?: string,
  ): Transaction<Action> {
    const initial = this.current(key);
    const entry = this.pending(id.create());
    let done = false;
    return {
      add: (actions) => {
        if (done) throw new Error("transaction finalized");
        if (initial == null) return;
        const current = this.current(key);
        if (current == null) return;
        const arr = array.toArray(actions);
        entry.actions.push(...arr);
        this.track(key, current, entry, this.params.reduce(current, arr).next);
      },
      commit: async () => {
        if (done) return false;
        done = true;
        if (entry.actions.length === 0 || initial == null) return false;
        // Inverse must be computed against the pre-transaction snapshot so
        // undo restores to where the user started, not the last-add post-state.
        const { inverse, targets } = this.params.reduce(initial, entry.actions);
        const stackRollback = this.recordEntry(
          key,
          entry.actions,
          inverse,
          targets,
          kind,
        );
        entry.epoch = this.epoch;
        try {
          await send(entry.actions, entry.dispatchKey);
        } catch (e) {
          stackRollback();
          this.discard(key, entry.dispatchKey);
          throw errors.fromUnknown(e);
        }
        this.accept(key, entry.dispatchKey);
        return true;
      },
      abort: () => {
        if (done) return;
        done = true;
        this.discard(key, entry.dispatchKey);
      },
    };
  }

  prepareUndo(key: Key): Reversal<Action> | null {
    return this.prepareReversal(key, "undo");
  }

  prepareRedo(key: Key): Reversal<Action> | null {
    return this.prepareReversal(key, "redo");
  }

  markRemoteTouched(key: Key, targets: readonly string[]): destructor.Destructor {
    if (targets.length === 0) return destructor.NOOP;
    const tick = ++this.tick;
    return this.updateUndo(key, (s) => {
      const remoteTouched = { ...s.remoteTouched };
      for (const t of targets) remoteTouched[t] = tick;
      return { ...s, remoteTouched };
    });
  }

  /** Whether the cached doc holds a dispatch that the Core has not echoed. */
  hasPending(key: Key): boolean {
    return this.rebases.has(key);
  }

  /**
   * Resolves once the cached doc holds no dispatch that the Core has not echoed. An
   * open transaction is pending until it commits or aborts.
   */
  async settled(key: Key): Promise<void> {
    if (!this.rebases.has(key)) return;
    await new Promise<void>((resolve) => {
      const waiting = this.settling.get(key);
      if (waiting == null) this.settling.set(key, [resolve]);
      else waiting.push(resolve);
    });
  }

  private release(key: Key): void {
    this.rebases.delete(key);
    const waiting = this.settling.get(key);
    if (waiting == null) return;
    this.settling.delete(key);
    waiting.forEach((resolve) => resolve());
  }

  /**
   * Apply a remote frame. The frame is dropped if its seq does not exceed the
   * high-water mark previously applied for this key. A frame for an uncached doc
   * applies only when a create heads it. A frame whose dispatch key matches a pending
   * entry is that entry's echo: it moves into the base, and the doc changes only if
   * other frames came first. Any other frame applies to the base, under the pending
   * entries, and marks its targets as remote-touched.
   */
  applyRemote(key: Key, seq: number, dispatchKey: string, actions: Action[]): void {
    if (seq !== 0) {
      const last = this.lastAppliedSeq.get(key) ?? 0;
      if (seq <= last) return;
      this.lastAppliedSeq.set(key, seq);
    }
    const doc = this.docs.get(key);
    if (doc == null) {
      // Anything but a create predates our knowledge of the doc, so it drops.
      const created = actions.length > 0 ? this.params.createOf(actions[0]) : undefined;
      if (created == null) return;
      this.docs.set(key, this.params.reduce(created, actions.slice(1)).next);
      return;
    }
    const r = this.rebases.get(key);
    if (r == null) {
      const { next, targets } = this.params.reduce(doc, actions);
      this.docs.set(key, next);
      this.markRemoteTouched(key, targets);
      return;
    }
    const { own, targets } = this.settle(key, r, dispatchKey, actions);
    if (!own) this.markRemoteTouched(key, targets);
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
   * given channel to this controller. Registered via {@link query.Cache.listen}.
   */
  listener(
    channel: string,
    schema: z.ZodType<Frame<Key, Action>>,
  ): query.Listener<z.ZodType<Frame<Key, Action>>> {
    return {
      channel,
      schema,
      onChange: (frames) => {
        frames.forEach(({ key, seq, dispatchKey, actions }) =>
          this.applyRemote(key, seq, dispatchKey, actions),
        );
      },
    };
  }
}
