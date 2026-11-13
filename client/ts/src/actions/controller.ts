// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
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
import { type z } from "zod";

import { AccessDeniedError, isConnectionError, NotFoundError } from "@/errors";
import { type query } from "@/query";
import { keyedZ, Table } from "@/query/table";

const DEFAULT_COALESCE_WINDOW = TimeSpan.milliseconds(500);
const DEFAULT_STACK_CAP = 200;
const STALE_AUTO_ADVANCE_CAP = 10;
const ECHO_DEADLINE = TimeSpan.seconds(10);

export interface StackEntry<Action> {
  /** Stays the same when the entry coalesces or is made again after a failed send. */
  key: string;
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

// A remade entry merges only into the entry it merged into the first time.
const pushOnto = <A>(
  stack: StackEntry<A>[],
  next: StackEntry<A>,
  window: TimeSpan,
  cap: number,
  remade: boolean,
): StackEntry<A>[] => {
  const top = stack[stack.length - 1];
  const mergeable =
    top != null &&
    (remade
      ? top.key === next.key
      : top.kind === next.kind &&
        next.ts.span(top.ts).lessThanOrEqual(window) &&
        compare.unorderedPrimitiveArrays(top.targets, next.targets) === compare.EQUAL);
  const merged = mergeable
    ? {
        key: top.key,
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
 * Sends a document's processed actions to the Core. Receives the controller-generated
 * dispatch key, held as pending before the call, so the broadcast echo is recognized as
 * own. Resolves to the actions the Core added to the dispatch, which its echo carries
 * after the sent actions.
 */
export type SendDispatch<Key, Action> = (
  key: Key,
  actions: Action[],
  dispatchKey: string,
) => Promise<Action[]>;

/** Builds the send for a domain's dispatch endpoint, which posts to the target. */
export const sender =
  <Key, Action>(
    unary: UnaryClient,
    target: string,
    reqZ: z.ZodType<{ key: Key; dispatchKey: string; actions: Action[] }>,
    resZ: z.ZodType<{ actions: Action[] }>,
  ): SendDispatch<Key, Action> =>
  async (key, actions, dispatchKey) =>
    (await unary.send(target, { key, dispatchKey, actions }, reqZ, resZ)).actions;

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
  /**
   * Subscribes to the change stream of the cache going down. No echo arrives until the
   * next epoch.
   */
  onStreamDown: (callback: () => void) => destructor.Destructor;
  reduce: Reducer<State, Action>;
  /** Sends dispatches to the Core. Supplied by the caller so the controller stays free
   *  of network concerns. */
  send: SendDispatch<Key, Action>;
  /**
   * Converts an action that a caller dispatches or adds to a transaction into the form
   * the Core stores, against the cached doc that the actions before it produce. Undo
   * and redo send actions that are already in that form.
   */
  encode?: (state: State, action: Action) => Action;
  isUndoable?: (action: Action) => boolean;
  kindOf?: (actions: Action[]) => string;
  /**
   * Returns the document carried by a create action, undefined otherwise.
   * Lets remote frames for never-cached documents ingest instead of drop.
   */
  createOf?: (action: Action) => State | undefined;
  /**
   * Returns whether an action changes a field that a list reads. A remote frame with
   * such an action fetches a document the store does not hold, since a list may now
   * admit it.
   */
  listed?: (action: Action) => boolean;
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
  /**
   * The actions the Core broadcasts for the dispatch. Undefined until it applies it.
   */
  echo?: Action[];
  /** Fires when the echo is late. */
  deadline?: ReturnType<typeof setTimeout>;
}

/** The dispatch state of a doc that has pending entries. */
interface Rebase<State, Action> {
  /** The doc as the Core holds it, without the pending entries. */
  base: State;
  pending: Array<Pending<Action>>;
}

interface Applied<Action> {
  processed: Action[];
  inverse: Action[];
  targets: readonly string[];
}

/** A queued edit and the actions it sends. */
interface Queued<State, Action> {
  actions: Action[];
  /**
   * Reverts what the edit changed outside its pending entry, such as its undo entry.
   */
  rollback: destructor.Destructor;
  /** Makes the edit again on the current document. Null when nothing is left to do. */
  remake: () => Made<State, Action> | null;
}

/** An edit to queue, and the cached doc before and after it. */
interface Made<State, Action> {
  edit: Queued<State, Action>;
  /** Undefined when the doc isn't cached. */
  doc?: { current: State; next: State };
}

/** A queued edit, and the result that its caller awaits. */
interface Member<State, Action> {
  edit: Queued<State, Action>;
  result: PromiseWithResolvers<void>;
}

/** Dispatches sent together as one, held as one pending entry. */
interface Batch<State, Action> {
  entry: Pending<Action>;
  members: Array<Member<State, Action>>;
}

interface Queue<State, Action> {
  next: Batch<State, Action> | null;
}

const revert = (batch: Batch<unknown, unknown>): void =>
  batch.members.toReversed().forEach(({ edit }) => edit.rollback());

/** A prepared edit's actions and the undo stack mutation that records them. */
interface Prepared<Action> {
  actions: Action[];
  commitStack: (applied: Applied<Action>) => destructor.Destructor;
}

/**
 * Owns the action-dispatch loop for one document table: local apply, per-doc undo/redo
 * stacks, echo recognition, and remote-frame application. The cached doc of a key with
 * pending dispatches is its base, as the Core holds it, with the pending dispatches
 * applied on top. A foreign frame applies to the base, and the pending dispatches apply
 * again on the result, so the doc stays in the order the Core applies them. Another
 * writer to the table writes the base, and its rollback restores the base. A failed
 * send removes its dispatch the same way. All dispatch bookkeeping is
 * session-local and dropped when a document is deleted.
 */
export class Controller<Key extends record.Key, State extends query.Data, Action> {
  private readonly docs: Table<Key, State>;
  private readonly undos: Table<Key, UndoState<Action>>;
  private readonly params: Required<
    Omit<
      ControllerParams<Key, State, Action>,
      "store" | "onEpoch" | "onStreamDown" | "encode"
    >
  > &
    Pick<ControllerParams<Key, State, Action>, "encode">;
  // lastAppliedSeq tracks the highest broadcast sequence number applied for each key.
  // Echoes whose seq does not exceed this are stale and are dropped. A seq of 0 is
  // "unstamped" and always applies.
  private readonly lastAppliedSeq = new Map<Key, number>();
  /** By doc, the dispatch keys of entries settled at their echo deadline. */
  private readonly expired = new Map<Key, Set<string>>();
  /** Docs to fetch again once they hold no pending entry. */
  private readonly stale = new Set<Key>();
  private readonly rebases = new Map<Key, Rebase<State, Action>>();
  /** Callers of settled, waiting for a doc to hold no pending entry. */
  private readonly settling = new Map<Key, Array<() => void>>();
  private epoch = 0;
  /** Whether the change stream is live, so echoes can arrive. */
  private live = false;
  /** Whether this controller is writing the table, so its interceptor stays out. */
  private writing = false;
  // Monotonic counter ordering entry pushes against remote touches. Program
  // order is the ground truth here; the wall clock ties within a millisecond
  // and steps on sleep/wake or NTP resync, so it cannot be trusted for this.
  private tick = 0;
  // The Core applies dispatches in arrival order, so each document has at most one send
  // in flight. A key is present while its send is in flight, and holds the batch
  // queued behind it.
  private readonly queues = new Map<Key, Queue<State, Action>>();
  // Documents with a fetch in flight, each with whether a frame or a write for it came
  // during the fetch.
  private readonly fetching = new Map<Key, boolean>();

  constructor(opts: ControllerParams<Key, State, Action>) {
    this.params = {
      onError: opts.onError,
      reduce: opts.reduce,
      send: opts.send,
      encode: opts.encode,
      isUndoable: opts.isUndoable ?? (() => true),
      kindOf: opts.kindOf ?? (() => "default"),
      createOf: opts.createOf ?? (() => undefined),
      listed: opts.listed ?? (() => false),
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
    // A doc with local edits or undo history must stay cached.
    this.docs.hold({
      held: () => [...this.rebases.keys(), ...this.queues.keys(), ...this.undos.keys()],
    });
    this.docs.intercept({
      base: (key) => (this.writing ? undefined : this.rebases.get(key)?.base),
      apply: (key, doc) => this.overlay(key, doc),
    });
    opts.onEpoch((epoch) => this.changeEpoch(epoch));
    opts.onStreamDown(() => {
      this.live = false;
      this.settleAnswered();
    });
  }

  private drop(key: Key): void {
    this.undos.delete(key);
    this.lastAppliedSeq.delete(key);
    this.stale.delete(key);
    this.release(key);
  }

  private changeEpoch(epoch: number): void {
    this.epoch = epoch;
    this.live = epoch !== 0;
    // Sequence numbers restart with the Core, and no echo crosses an epoch. A new epoch
    // fetches or discards every cached doc.
    this.lastAppliedSeq.clear();
    this.expired.clear();
    this.stale.clear();
    if (epoch === 0) {
      [...this.rebases.keys()].forEach((key) => this.release(key));
      this.undos.delete(() => true);
      return;
    }
    // The echoes of applied dispatches may have been lost in the gap.
    this.settleAnswered();
  }

  /** Settles each entry that the Core applied, without its echo. */
  private settleAnswered(): void {
    for (const [key, r] of [...this.rebases])
      for (const { dispatchKey, echo } of [...r.pending])
        if (echo != null) this.settle(key, r, dispatchKey, echo);
  }

  /**
   * Sends actions in order with the document's other dispatches, and applies them to
   * the cached doc until their echo arrives. Unlike dispatch, it sends when the doc is
   * not cached and records no undo entry. When nothing is in flight for the document,
   * the actions go out at once. Otherwise they join the batch that goes out as one
   * dispatch when the in-flight send settles.
   * @throws {Error} if the cached doc cannot hold an action, before any send.
   * @throws {Error} if the send fails, after the actions roll back.
   */
  async send(key: Key, actions: Action[]): Promise<void> {
    const made = (): Made<State, Action> => {
      const current = this.docs.get(key);
      return {
        edit: { actions, rollback: destructor.NOOP, remake: made },
        doc:
          current == null
            ? undefined
            : { current, next: this.params.reduce(current, actions).next },
      };
    };
    const first = made();
    await this.enqueue(key, (queue, result) => this.join(key, queue, first, result));
  }

  /**
   * Adds an edit to the batch that sends next, and sends it at once when nothing is in
   * flight for the document.
   * @throws {Error} if the send fails.
   */
  private async enqueue(
    key: Key,
    add: (queue: Queue<State, Action>, result: PromiseWithResolvers<void>) => void,
  ): Promise<void> {
    let queue = this.queues.get(key);
    const inFlight = queue != null;
    if (queue == null) {
      queue = { next: null };
      this.queues.set(key, queue);
    }
    const result = Promise.withResolvers<void>();
    add(queue, result);
    if (!inFlight) void this.drain(key, queue);
    await result.promise;
  }

  private nextBatch(queue: Queue<State, Action>): Batch<State, Action> {
    queue.next ??= { entry: this.createPending(id.create()), members: [] };
    return queue.next;
  }

  /** Adds the edit to the batch that sends next and applies it to the cached doc. */
  private join(
    key: Key,
    queue: Queue<State, Action>,
    { edit, doc }: Made<State, Action>,
    result: PromiseWithResolvers<void>,
  ): void {
    const batch = this.nextBatch(queue);
    batch.members.push({ edit, result });
    batch.entry.actions.push(...edit.actions);
    if (doc != null) this.track(key, doc.current, batch.entry, doc.next);
  }

  /**
   * Adds the edit of a pending entry that the cached doc already holds to the batch
   * that sends next. With no batch waiting, the entry becomes the batch's entry.
   */
  private adopt(
    key: Key,
    queue: Queue<State, Action>,
    entry: Pending<Action>,
    member: Member<State, Action>,
  ): void {
    let batch = queue.next;
    if (batch == null) {
      batch = { entry, members: [] };
      queue.next = batch;
    } else this.merge(key, entry, batch.entry);
    batch.members.push(member);
  }

  /** Moves the actions of a pending entry to the end of another. */
  private merge(key: Key, from: Pending<Action>, into: Pending<Action>): void {
    into.actions.push(...from.actions);
    const r = this.rebases.get(key);
    if (r == null) return;
    r.pending = r.pending.filter((p) => p !== from && p !== into);
    r.pending.push(into);
    this.write(key, r);
  }

  private async drain(key: Key, queue: Queue<State, Action>): Promise<void> {
    while (queue.next != null) {
      const batch = queue.next;
      queue.next = null;
      const { entry } = batch;
      try {
        // A remake after a failed send can leave a batch with nothing to send.
        if (entry.actions.length > 0) {
          entry.epoch = this.epoch;
          const added = await this.params.send(key, entry.actions, entry.dispatchKey);
          this.accept(key, entry.dispatchKey, added);
        }
        batch.members.forEach(({ result }) => result.resolve());
      } catch (e) {
        this.fail(key, batch, queue);
        const error = errors.fromUnknown(e);
        batch.members.forEach(({ result }) => result.reject(error));
      }
    }
    this.queues.delete(key);
  }

  // Rolls back a failed batch. Rollbacks restore what came before their edit, so they
  // run newest first, the queued batch's before the failed one's. The queued batch has
  // not been sent, so its edits are then made again on the doc without the failed one.
  // An edit that cannot be made again rejects its caller and leaves the batch.
  private fail(
    key: Key,
    failed: Batch<State, Action>,
    queue: Queue<State, Action>,
  ): void {
    const queued = queue.next;
    if (queued != null) revert(queued);
    revert(failed);
    if (queued == null) {
      this.discard(key, failed.entry.dispatchKey);
      return;
    }
    this.discard(key, failed.entry.dispatchKey, queued.entry.dispatchKey);
    const { members } = queued;
    queued.members = [];
    queued.entry.actions = [];
    for (const { edit, result } of members) {
      let made: Made<State, Action> | null;
      try {
        made = edit.remake();
      } catch (e) {
        result.reject(errors.fromUnknown(e));
        continue;
      }
      if (made == null) result.resolve();
      else this.join(key, queue, made, result);
    }
  }

  private updateUndo(
    key: Key,
    fn: (s: UndoState<Action>) => UndoState<Action>,
  ): destructor.Destructor {
    return this.undos.set(key, (prev) => fn(prev ?? ZERO_UNDO<Action>()));
  }

  /**
   * Makes a doc that another writer puts in the table the base of its key, and returns
   * it with the pending entries on top.
   */
  private overlay(key: Key, doc: State): State {
    const r = this.rebases.get(key);
    if (r == null || this.writing) return doc;
    r.base = doc;
    return this.rebuild(r);
  }

  /** Returns the base with the pending entries applied. */
  private rebuild(r: Rebase<State, Action>): State {
    return r.pending.reduce(
      (state, { actions }) => this.params.reduce(state, actions).next,
      r.base,
    );
  }

  private write(key: Key, r: Rebase<State, Action>): void {
    this.set(key, this.rebuild(r));
  }

  private set(key: Key, doc: State): void {
    this.markFetch(key);
    this.writing = true;
    try {
      this.docs.set(key, doc);
    } finally {
      this.writing = false;
    }
  }

  /**
   * Holds the entry as the last pending entry and writes the doc with it. The next doc
   * applies the entry's latest actions last.
   */
  private track(key: Key, current: State, entry: Pending<Action>, next: State): void {
    let r = this.rebases.get(key);
    if (r == null) {
      r = { base: current, pending: [] };
      this.rebases.set(key, r);
    }
    const i = r.pending.indexOf(entry);
    if (i === -1) r.pending.push(entry);
    if (i === -1 || i === r.pending.length - 1) {
      this.set(key, next);
      return;
    }
    // An open transaction sends after the entries behind it, so it moves behind them.
    r.pending.splice(i, 1);
    r.pending.push(entry);
    this.write(key, r);
  }

  /**
   * Applies actions that the Core applied to the base, and removes the pending entry
   * with the dispatch key. When the base cannot hold the actions, it reports the error,
   * keeps the base, and fetches the doc again. Returns whether an entry had the key,
   * and the targets.
   */
  private settle(
    key: Key,
    r: Rebase<State, Action>,
    dispatchKey: string,
    actions: Action[],
  ): { own: boolean; targets: readonly string[] } {
    const i = r.pending.findIndex((p) => p.dispatchKey === dispatchKey);
    let applied: ReturnType<Reducer<State, Action>> | null = null;
    try {
      applied = this.params.reduce(r.base, actions);
    } catch (exc) {
      this.params.onError(
        new Error("failed to apply a dispatch that the Core applied", { cause: exc }),
      );
      this.refetch(key);
    }
    // The doc already holds the first pending entry. It changes only when the echo
    // carries actions that the entry does not, such as fields the Core derives.
    const held =
      applied != null && i === 0 && deep.equal(r.pending[0].actions, actions);
    if (applied != null) r.base = applied.next;
    if (i !== -1) clearTimeout(r.pending.splice(i, 1)[0].deadline);
    if (held) {
      this.markFetch(key);
      this.docs.stampWrite(key);
    } else this.write(key, r);
    if (r.pending.length === 0) this.release(key);
    return { own: i !== -1, targets: applied?.targets ?? [] };
  }

  /**
   * Removes the pending entries with the dispatch keys, which the Core did not apply.
   */
  private discard(key: Key, ...dispatchKeys: string[]): void {
    const r = this.rebases.get(key);
    if (r == null) return;
    const kept = r.pending.filter((p) => !dispatchKeys.includes(p.dispatchKey));
    if (kept.length === r.pending.length) return;
    r.pending = kept;
    this.write(key, r);
    if (r.pending.length === 0) this.release(key);
  }

  /** Marks a pending entry as applied by the Core, which added the given actions. */
  private accept(key: Key, dispatchKey: string, added: Action[]): void {
    const r = this.rebases.get(key);
    const p = r?.pending.find((p) => p.dispatchKey === dispatchKey);
    if (r == null || p == null) return;
    const echo = [...p.actions, ...added];
    p.echo = echo;
    // No echo arrives while the stream is down. The echo may also have been lost when
    // the epoch changed during the send.
    if (!this.live || p.epoch !== this.epoch) {
      this.settle(key, r, dispatchKey, echo);
      return;
    }
    p.deadline = setTimeout(
      () => this.expire(key, r, dispatchKey, echo),
      ECHO_DEADLINE.milliseconds,
    );
  }

  /**
   * Settles an entry whose echo is late, and fetches the doc again, since the stream
   * may have lost other changes to it.
   */
  private expire(
    key: Key,
    r: Rebase<State, Action>,
    dispatchKey: string,
    echo: Action[],
  ): void {
    const expired = this.expired.get(key) ?? new Set<string>();
    expired.add(dispatchKey);
    this.expired.set(key, expired);
    this.refetch(key);
    this.settle(key, r, dispatchKey, echo);
  }

  /**
   * Fetches the doc again, once it holds no pending entry: a fetched doc can already
   * hold a pending entry, which would then apply twice.
   */
  private refetch(key: Key): void {
    if (this.rebases.has(key)) this.stale.add(key);
    else void this.fetch(key);
  }

  private createPending(dispatchKey: string, actions: Action[] = []): Pending<Action> {
    return { dispatchKey, actions, epoch: this.epoch };
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
    return this.record(key, forward, inverse, targets, kindOverride).rollback;
  }

  // Pushes an entry as recordEntry does. With remakeOf, makes the entry with that key
  // again after a failed send. Returns the key of the entry that holds the push.
  private record(
    key: Key,
    forward: Action[],
    inverse: Action[],
    targets: readonly string[],
    kindOverride?: string,
    remakeOf?: string,
  ): { rollback: destructor.Destructor; entryKey?: string } {
    if (targets.length === 0) return { rollback: destructor.NOOP };
    const undoableForward = forward.filter(this.params.isUndoable);
    if (undoableForward.length === 0) return { rollback: destructor.NOOP };
    const entry: StackEntry<Action> = {
      key: remakeOf ?? id.create(),
      forward: undoableForward,
      inverse,
      kind: kindOverride ?? this.params.kindOf(forward),
      ts: this.params.now(),
      tick: ++this.tick,
      targets,
    };
    const rollback = this.updateUndo(key, (s) => ({
      ...s,
      undo: pushOnto(
        s.undo,
        entry,
        this.params.coalesceWindow,
        this.params.stackCap,
        remakeOf != null,
      ),
      redo: [],
    }));
    return { rollback, entryKey: this.undos.get(key)?.undo.at(-1)?.key };
  }

  // Records the entry for a dispatch or transaction, and records it again under the
  // same key when a failed send makes the edit again.
  private stackRecorder(
    key: Key,
    kind?: string,
  ): (applied: Applied<Action>) => destructor.Destructor {
    let entryKey: string | undefined;
    return ({ processed, inverse, targets }) => {
      const recorded = this.record(key, processed, inverse, targets, kind, entryKey);
      entryKey ??= recorded.entryKey;
      return recorded.rollback;
    };
  }

  // Walks the chosen stack from the top, skipping entries whose targets were
  // remote-touched after the entry was pushed. Caps the walk at
  // STALE_AUTO_ADVANCE_CAP and drops a fully stale tail. On hit, returns the
  // entry's reversal actions (inverse for undo, forward for redo) and a
  // commit that moves the entry to the opposite stack.
  private prepareReversal(
    key: Key,
    side: "undo" | "redo",
  ): (Reversal<Action> & { entryKey: string }) | null {
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
    return this.reversal(key, side, stack, idx);
  }

  private reversal(
    key: Key,
    side: "undo" | "redo",
    stack: StackEntry<Action>[],
    idx: number,
  ): Reversal<Action> & { entryKey: string } {
    const entry = stack[idx];
    const other = side === "undo" ? "redo" : "undo";
    return {
      entryKey: entry.key,
      actions: side === "undo" ? entry.inverse : entry.forward,
      commit: () =>
        this.updateUndo(key, (s) => ({
          ...s,
          [side]: s[side].slice(0, idx),
          [other]: [...s[other], entry],
        })),
    };
  }

  // Makes the edit on the cached doc and queues its send. Returns false when there is
  // nothing to prepare or the doc isn't cached.
  private async applySend(
    key: Key,
    prepare: (current: State) => Prepared<Action> | null,
  ): Promise<boolean> {
    const made = this.make(key, prepare);
    if (made == null) return false;
    // A no-op has nothing for the server. The stack mutation still runs so undo/redo
    // consume their entry instead of pinning it.
    if (made.doc.next === made.doc.current) return true;
    await this.enqueue(key, (queue, result) => this.join(key, queue, made, result));
    return true;
  }

  // Prepares an edit against the cached doc and runs its stack mutation. Returns null
  // when there is nothing to prepare or the doc isn't cached.
  private make(
    key: Key,
    prepare: (current: State) => Prepared<Action> | null,
  ): Required<Made<State, Action>> | null {
    const current = this.docs.get(key);
    if (current == null) return null;
    const prepared = prepare(current);
    if (prepared == null) return null;
    const { actions } = prepared;
    const { next, inverse, targets } = this.params.reduce(current, actions);
    const rollback = prepared.commitStack({ processed: actions, inverse, targets });
    return {
      edit: { actions, rollback, remake: () => this.make(key, prepare) },
      doc: { current, next },
    };
  }

  // Prepares an undo or redo. Made again after a failed send, it reverses the same
  // entry, or nothing when that entry is gone.
  private reversalPreparer(
    key: Key,
    side: "undo" | "redo",
  ): () => Prepared<Action> | null {
    let entryKey: string | undefined;
    return () => {
      let prepared: (Reversal<Action> & { entryKey: string }) | null = null;
      if (entryKey == null) prepared = this.prepareReversal(key, side);
      else {
        const stack = this.undos.get(key)?.[side] ?? [];
        const idx = stack.findLastIndex((e) => e.key === entryKey);
        if (idx !== -1) prepared = this.reversal(key, side, stack, idx);
      }
      if (prepared == null) return null;
      entryKey = prepared.entryKey;
      return { actions: prepared.actions, commitStack: prepared.commit };
    };
  }

  /**
   * Apply an action batch to the cached doc and send it to the server,
   * recording an undoable entry. Returns false without side effects when the
   * doc isn't cached. Removes the local apply and rethrows on send failure.
   */
  async dispatch(
    key: Key,
    actions: Action[],
    opts: Options<State, Action> = {},
  ): Promise<boolean> {
    if (actions.length === 0) return true;
    const { preprocess } = opts;
    const commitStack = this.stackRecorder(key);
    return await this.applySend(key, (current) => ({
      actions: this.encode(
        current,
        preprocess == null ? actions : preprocess(current, actions),
      ),
      commitStack,
    }));
  }

  /** Encodes each action against the doc that the actions before it produce. */
  private encode(state: State, actions: Action[]): Action[] {
    const { encode, reduce } = this.params;
    if (encode == null) return actions;
    let doc = state;
    return actions.map((action, i) => {
      const encoded = encode(doc, action);
      if (i < actions.length - 1) doc = reduce(doc, [encoded]).next;
      return encoded;
    });
  }

  /**
   * Apply and send the top live undo entry, moving it to the redo stack. Returns false
   * when nothing is undoable. Rolls back and rethrows on send failure.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.applySend(key, this.reversalPreparer(key, "undo"));
  }

  /** Mirror of {@link undo} for the redo stack. */
  async redo(key: Key): Promise<boolean> {
    return await this.applySend(key, this.reversalPreparer(key, "redo"));
  }

  /**
   * Sends a create for the given keys. The cached docs among them hold their create as
   * pending, so its echo does not revert edits made before the echo arrives. Dispatches
   * to the keys during the send wait for it, then go out in order.
   * @returns what the send returns.
   * @throws {Error} if the send fails. The waiting dispatches then roll back and reject
   * with the same error.
   */
  async create<T>(keys: Key[], send: (dispatchKey: string) => Promise<T>): Promise<T> {
    const dispatchKey = id.create();
    const held = keys
      .filter((key) => !this.queues.has(key))
      .map((key) => {
        const queue: Queue<State, Action> = { next: null };
        this.queues.set(key, queue);
        return { key, queue };
      });
    const tracked: Key[] = [];
    for (const key of keys) {
      const current = this.docs.get(key);
      if (current == null) continue;
      this.track(key, current, this.createPending(dispatchKey), current);
      tracked.push(key);
    }
    let res: T;
    try {
      res = await send(dispatchKey);
    } catch (e) {
      const err = errors.fromUnknown(e);
      held.forEach(({ key, queue }) => {
        this.queues.delete(key);
        const batch = queue.next;
        if (batch == null) return;
        revert(batch);
        this.discard(key, batch.entry.dispatchKey);
        batch.result.reject(err);
      });
      tracked.forEach((key) => this.discard(key, dispatchKey));
      throw err;
    }
    tracked.forEach((key) => this.accept(key, dispatchKey, []));
    held.forEach(({ key, queue }) => void this.drain(key, queue));
    return res;
  }

  /**
   * Stage actions committed atomically as one undoable entry. Each add applies to the
   * cached doc at once, and the staged actions stay pending until the commit settles.
   */
  transaction(key: Key, kind?: string): Transaction<Action> {
    const initial = this.docs.get(key);
    const entry = this.createPending(id.create());
    let done = false;
    return {
      add: (actions) => {
        if (done) throw new Error("transaction finalized");
        if (initial == null) return;
        const current = this.docs.get(key);
        if (current == null) return;
        const arr = this.encode(current, array.toArray(actions));
        const { next } = this.params.reduce(current, arr);
        entry.actions.push(...arr);
        this.track(key, current, entry, next);
      },
      commit: async () => {
        if (done) return false;
        done = true;
        if (entry.actions.length === 0 || initial == null) return false;
        // The batch that adopts the entry adds to its actions.
        const actions = [...entry.actions];
        // Inverse must be computed against the pre-transaction snapshot so
        // undo restores to where the user started, not the last-add post-state.
        const { inverse, targets } = this.params.reduce(initial, actions);
        const commitStack = this.stackRecorder(key, kind);
        const edit: Queued<State, Action> = {
          actions,
          rollback: commitStack({ processed: actions, inverse, targets }),
          remake: () => this.make(key, () => ({ actions, commitStack })),
        };
        await this.enqueue(key, (queue, result) =>
          this.adopt(key, queue, entry, { edit, result }),
        );
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

  /**
   * Whether the cached doc holds a dispatch that the Core has not echoed, a create
   * that it has not echoed, or an open transaction.
   */
  hasPending(key: Key): boolean {
    return this.rebases.has(key);
  }

  /**
   * Resolves once the cached doc holds no dispatch that the Core has not echoed. A
   * dispatch that the Core applied stops waiting for its echo when the change stream
   * goes down, or when the echo is 10 seconds late. An open transaction is pending
   * until it commits or aborts.
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
    this.rebases.get(key)?.pending.forEach(({ deadline }) => clearTimeout(deadline));
    this.rebases.delete(key);
    if (this.stale.delete(key)) void this.fetch(key);
    const waiting = this.settling.get(key);
    if (waiting == null) return;
    this.settling.delete(key);
    waiting.forEach((resolve) => resolve());
  }

  /**
   * Apply a remote frame. The frame is dropped if its seq does not exceed the
   * high-water mark previously applied for this key. A frame for an uncached doc
   * applies only when a create heads it. Otherwise the doc is fetched when the frame
   * changes a field that a list reads. A frame whose dispatch key matches a pending
   * entry is that entry's echo: it moves into the base, and the doc changes only if
   * other frames came first. An echo that comes after its deadline drops. Any other
   * frame applies to the base, under the pending entries, and marks its targets as
   * remote-touched. A frame that fails to apply reports the error and fetches the doc
   * again.
   */
  applyRemote(key: Key, seq: number, dispatchKey: string, actions: Action[]): void {
    this.markFetch(key);
    // Until the fetch caches the doc, there is nothing to apply the frame to.
    if (this.fetching.has(key) && this.docs.get(key) == null) return;
    if (seq !== 0 && seq <= (this.lastAppliedSeq.get(key) ?? 0)) return;
    try {
      this.applyFrame(key, dispatchKey, actions);
    } catch (exc) {
      this.params.onError(new Error("failed to apply a remote frame", { cause: exc }));
      this.refetch(key);
      return;
    }
    if (seq !== 0) this.lastAppliedSeq.set(key, seq);
  }

  private applyFrame(key: Key, dispatchKey: string, actions: Action[]): void {
    const expired = this.expired.get(key);
    if (expired?.delete(dispatchKey) === true) {
      if (expired.size === 0) this.expired.delete(key);
      return;
    }
    const doc = this.docs.get(key);
    if (doc == null) {
      // Anything but a create predates our knowledge of the doc, so it drops.
      const created = actions.length > 0 ? this.params.createOf(actions[0]) : undefined;
      if (created != null) {
        this.docs.set(key, this.params.reduce(created, actions.slice(1)).next);
        return;
      }
      if (actions.some((action) => this.params.listed(action))) void this.fetch(key);
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

  // A fetch in flight may answer from before a frame or a write, so it runs again.
  private markFetch(key: Key): void {
    if (this.fetching.has(key)) this.fetching.set(key, true);
  }

  /**
   * Fetches a doc until no frame or write for it comes during the fetch. A doc that
   * holds a pending entry after a fetch is fetched again once it holds none. Never
   * rejects.
   */
  private async fetch(key: Key): Promise<void> {
    if (this.fetching.has(key)) {
      this.fetching.set(key, true);
      return;
    }
    try {
      do {
        this.fetching.set(key, false);
        await this.docs.retrieve([key], { refresh: true });
        if (this.rebases.has(key)) {
          this.stale.add(key);
          return;
        }
      } while (this.fetching.get(key) === true);
    } catch (exc) {
      // A deleted or denied doc joins no list, and the reconcile after a reconnect
      // fetches every doc again.
      if (
        NotFoundError.matches(exc) ||
        AccessDeniedError.matches(exc) ||
        isConnectionError(exc)
      )
        return;
      this.params.onError(new Error("failed to fetch a document", { cause: exc }));
    } finally {
      this.fetching.delete(key);
    }
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
      onInvalid: (raw) => {
        const keyed = keyedZ.safeParse(raw);
        if (keyed.success) this.refetch(keyed.data.key as Key);
      },
    };
  }
}
