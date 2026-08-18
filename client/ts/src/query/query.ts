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
  type destructor,
  errors,
  primitive,
  type record,
  type state,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { AccessDeniedError, NotFoundError } from "@/errors";
import { Deleted } from "@/query/deleted";
import { type Table, type TableEvent } from "@/query/table";
import { type Data, type FetchOptions, type Params } from "@/query/types";

const hashes = new WeakMap<object, string>();

/**
 * Deterministically serializes query params to a stable string. Keys are sorted
 * recursively so `{a: 1, b: 2}` and `{b: 2, a: 1}` collapse to the same key,
 * and explicitly-undefined fields hash like absent ones (matching JSON
 * semantics). Class instances implementing {@link primitive.Hashable}
 * delegate to their `hash()` method; plain objects and arrays recurse
 * structurally, memoized per object identity ({@link Params} is readonly, so
 * an object's hash never changes).
 */
export const hash = (params: Params): string => {
  if (params === null) return "null";
  if (params === undefined) return "undefined";
  if (typeof params === "bigint") return `${params.toString()}n`;
  if (typeof params !== "object") return JSON.stringify(params);
  if (primitive.isHashable(params)) return params.hash();
  const held = hashes.get(params);
  if (held !== undefined) return held;
  let result: string;
  if (Array.isArray(params)) result = `[${params.map(hash).join(",")}]`;
  else {
    const fields = Object.entries(params)
      .filter(([, v]) => v !== undefined)
      .sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0));
    result = `{${fields.map(([k, v]) => `${JSON.stringify(k)}:${hash(v)}`).join(",")}}`;
  }
  hashes.set(params, result);
  return result;
};

/**
 * A cached answer as delivered to consumers: the query's current data, or the
 * {@link Deleted} corpse of a record that was deleted. A corpse is never
 * delivered as bare data, so deletion cannot be mistaken for a live result.
 * Narrow with {@link isLive}, or {@link Deleted.matches} to reach the corpse.
 */
export type Cached<D extends Data> = D | Deleted<D>;

/** True when a cached answer is live: cached at all, and not a deletion. */
export const isLive = <D extends Data>(value: Cached<D> | undefined): value is D =>
  value != null && !Deleted.matches<D>(value);

/**
 * Returns the corpse of a deleted answer, for callers that need the record's
 * last value (restoring it, say) and have no use for any other outcome.
 * @throws {NotFoundError} if the answer is live or nothing is cached.
 */
export const requireCorpse = <D extends Data>(value: Cached<D> | undefined): D => {
  if (!Deleted.matches<D>(value))
    throw new NotFoundError("the deleted record is no longer cached");
  return value.corpse;
};

/**
 * Receives the new cached answer for a query. `undefined` means the answer was
 * invalidated; the next read refetches.
 */
export interface ChangeHandler<D extends Data> {
  (result: Cached<D> | undefined): void;
}

/**
 * The read surface of one query space: fetch, subscribe, snapshot. The public
 * face of every answer space, including the named spaces domain clients
 * expose (children, parents, kv).
 */
export interface Retrieves<P extends Params, D extends Data> {
  retrieve: (params: P, options?: FetchOptions) => Promise<D>;
  onChange: (params: P, handler: ChangeHandler<D>) => destructor.Destructor;
  getCached: (params: P) => Cached<D> | undefined;
}

/**
 * A foreign-table subscription for an answer space: projects that table's
 * events onto the space's primary keys. Build with {@link watch}.
 */
export interface Watch<P extends Params, K extends record.Key> {
  attach: (
    params: P,
    onEvent: (result: K[] | "refetch") => void,
  ) => destructor.Destructor;
}

/**
 * Declares that events on the given table affect answers in this space.
 * `affects` maps an event to the primary keys whose membership or content it
 * touches, "refetch" to invalidate wholesale, or null when unaffected.
 */
export const watch = <
  P extends Params,
  K extends record.Key,
  ForeignKey extends record.Key,
  ForeignValue extends state.State,
>(
  table: Table<ForeignKey, ForeignValue>,
  affects: (
    event: TableEvent<ForeignKey, ForeignValue>,
    params: P,
  ) => K[] | "refetch" | null,
): Watch<P, K> => ({
  attach: (params, onEvent) =>
    // Batched so one foreign write fires onEvent once: a refetch verdict for any event
    // supersedes the batch's keys, which otherwise union.
    table.subscribeBatch((events) => {
      let keys: K[] = [];
      for (const event of events) {
        const result = affects(event, params);
        if (result == null) continue;
        if (result === "refetch") return onEvent("refetch");
        keys = keys.concat(result);
      }
      if (keys.length > 0) onEvent(keys);
    }),
});

/**
 * Declaration of one query space: which table owns its records, how to fetch
 * an answer's keys, how to assemble content, and how membership is maintained.
 *
 * Maintenance is chosen per query instance:
 * 1. exact-key — `keyOf` returns a key: track that entry; deletion flips the
 *    answer to deleted.
 * 2. client-checkable — `matches` compares a record against the query:
 *    admit/evict exactly, no network.
 * 3. server-computed — any field named in `serverFields` is set on the query
 *    (or neither `keyOf` nor `matches` applies): debounced wholesale refetch.
 */
export interface SpaceConfig<
  P extends Params,
  D extends Data,
  K extends record.Key = record.Key,
  V extends state.State = state.State,
> {
  /** Resource name used in error messages, e.g. "range". */
  name: string;
  /** The table owning this space's record content. */
  table: Table<K, V>;
  /**
   * Fetches the answer from the cluster: writes fetched records into their
   * tables and returns the member keys.
   */
  fetch: (params: P, options?: FetchOptions) => Promise<K[]>;
  /** Assembles the answer from resolved member records at read time. */
  compose: (records: V[], params: P) => D;
  /** Rule 1: the single key a query addresses, or null when it doesn't. */
  keyOf?: (params: P) => K | null;
  /**
   * The exact key set a keys-only query addresses, or null when it doesn't. Lets
   * getCached compose the answer straight from the table (params order, deduped,
   * tombstoned members omitted) and lets maintenance seed a ready answer on
   * subscribe, so a fully cached key set never fetches.
   */
  keysOf?: (params: P) => K[] | null;
  /** Rule 2: whether a record satisfies the query. Pure; no network. */
  matches?: (record: V, params: P) => boolean;
  /**
   * Query fields only the server can evaluate (searchTerm, limit, offset).
   * A query instance with any of them set is maintained by rule 3.
   */
  serverFields?: readonly string[];
  /** Foreign tables whose events affect this space's answers. */
  watch?: Watch<P, K>[];
  /**
   * Marks a space whose answer addresses one record and cannot compose an
   * empty membership: rule-2 eviction of the last member flips the query to
   * deleted (when the entry tombstoned) or invalidates it, instead of
   * composing []. Leave unset for spaces where [] is a valid answer.
   */
  single?: boolean;
}

/** Wiring a {@link Space} receives from the cache that owns it. */
export interface SpaceHooks {
  /** Started (not awaited) on reads and subscriptions to open change delivery. */
  ensureStreaming?: () => Promise<void>;
  /** Subscribes to connection-epoch changes; maintained answers refetch on bump. */
  onEpoch?: (callback: (epoch: number) => void) => destructor.Destructor;
  /**
   * Whether change delivery keeps subscribed answers current. False when the
   * cache is detached or the cluster refused the stream. Defaults to true.
   */
  maintained?: () => boolean;
  /** Reports maintenance errors that have no caller to throw to. Defaults to
   *  console logging. */
  onError?: (error: Error) => void;
  /**
   * How long maintenance survives the last unsubscribe, so a quick remount finds
   * the answer still maintained and skips the reconfirm refetch. Zero tears down
   * synchronously. Defaults to 5 seconds.
   */
  teardownGrace?: TimeSpan;
}

const DEFAULT_DEBOUNCE = TimeSpan.milliseconds(100);
const DEFAULT_TEARDOWN_GRACE = TimeSpan.seconds(5);

type QueryState<K extends record.Key, D extends Data> =
  | { variant: "unfetched" }
  | { variant: "loading"; promise: Promise<D> }
  | { variant: "ready"; keys: K[] }
  | { variant: "error"; error: Error }
  | { variant: "deleted"; key: K };

interface Query<
  P extends Params,
  K extends record.Key,
  D extends Data,
  V extends state.State,
> {
  params: P;
  state: QueryState<K, D>;
  handlers: Set<ChangeHandler<D>>;
  /** Rule teardown; present exactly while the query is maintained. */
  teardown?: destructor.Destructor[];
  /** Set when maintenance ends: changes since then went unobserved. */
  unmaintained?: boolean;
  /** Keys whose membership a fetch in flight deferred; drained on settle. */
  pendingRechecks?: Set<K>;
  /** Set when a fetch in flight deferred a refetch; honored on settle. */
  refetchOnSettle?: boolean;
  refetchTimer?: ReturnType<typeof setTimeout>;
  /** When the last subscriber left; the sweep tears the query down once this
   *  outlives the grace window. Cleared on resubscribe. */
  idleSince?: TimeStamp;
  /** Answer interned against the records it was composed from, so repeated
   *  reads stay referentially stable for useSyncExternalStore consumers. A
   *  table replaces an entry only when its content changes, so a surviving entry
   *  set means a surviving answer. */
  composed?: { records: V[]; value: D };
}

/**
 * Cached read path for one query space of a domain client: deduped fetches,
 * per-query lifecycle, subscriptions, and deletion delivery. Answers hold
 * only key lists; record content lives in tables and is assembled at read
 * time, so answer content cannot diverge from the record cache.
 *
 * Answers are maintained only while subscribed. An unsubscribed read
 * refetches; an unsubscribed getCached serves the retained (possibly stale)
 * answer, recomposed against live tables.
 */
export class Space<
  P extends Params,
  D extends Data,
  K extends record.Key = record.Key,
  V extends state.State = state.State,
> {
  private readonly queries = new Map<string, Query<P, K, D, V>>();
  /** Composed corpses interned per tombstone and params, so deleted answers
   *  stay referentially stable. Identity composes reuse the tombstone. */
  private readonly corpses = new WeakMap<Deleted<V>, Map<string, Deleted<D>>>();
  /** Answers for reads with no live query, interned per entry and params. */
  private readonly entryAnswers = new WeakMap<object, Map<string, D>>();
  private readonly config: SpaceConfig<P, D, K, V>;
  private readonly hooks: SpaceHooks & {
    onError: NonNullable<SpaceHooks["onError"]>;
  };
  private readonly detachEpoch?: destructor.Destructor;
  private readonly grace: TimeSpan;
  private sweepTimer?: ReturnType<typeof setTimeout>;

  constructor(config: SpaceConfig<P, D, K, V>, hooks: SpaceHooks = {}) {
    this.config = config;
    this.hooks = { ...hooks, onError: hooks.onError ?? console.error };
    this.grace = hooks.teardownGrace ?? DEFAULT_TEARDOWN_GRACE;
    this.detachEpoch = hooks.onEpoch?.((epoch) => {
      // 0 is a return to cold (cluster replacement): the fresh stream's own
      // epoch bump refetches, not the reset itself.
      if (epoch === 0) return;
      this.queries.forEach((query) => {
        if (query.teardown != null) void this.refetch(query);
      });
    });
  }

  /**
   * Returns the answer to the query: instantly when cached and subscribed,
   * joining the in-flight fetch when one exists, fetching otherwise. Settled
   * answers are kept fresh only while maintained (subscribed, or within the teardown
   * grace window after the last unsubscribe, and only while change delivery is on),
   * so an unmaintained read always refetches. A previously failed query refetches.
   * @throws {NotFoundError} if the queried record was deleted.
   */
  retrieve(params: P, options?: FetchOptions): Promise<D> {
    const query = this.ensure(params);
    const { state } = query;
    if (state.variant === "loading") return state.promise;
    if (query.teardown != null && this.hooks.maintained?.() !== false)
      switch (state.variant) {
        case "ready":
          return Promise.resolve(this.composedOf(query, state.keys));
        case "deleted":
          return Promise.reject(new NotFoundError(`${this.config.name} was deleted`));
      }
    void this.startStreaming();
    return this.fetch(query, options);
  }

  /**
   * Returns the cached answer without touching the network: the current data,
   * the corpse of a deleted record, or undefined when nothing is cached.
   * Exact-key queries also resolve from a table entry cached by any other
   * query. May be stale for queries nothing subscribes to.
   */
  getCached(params: P): Cached<D> | undefined {
    const query = this.queries.get(hash(params));
    if (query != null) {
      const cached = this.cachedOf(query);
      if (cached != null) return cached;
    }
    // Rule-1 queries read straight through to the table: an entry (or tombstone)
    // answers the query even before any fetch or subscription exists.
    const key = this.config.keyOf?.(params);
    if (key != null) {
      const { table } = this.config;
      const entry = table.get(key);
      if (entry != null) return this.entryAnswerOf(entry, params);
      const tombstone = table.getTombstone(key);
      if (tombstone == null) return undefined;
      return this.deletedOf(tombstone, params);
    }
    // Keys-only queries compose from the table the same way, when every key resolves.
    // The answer matches what the fetch would return.
    const keys = this.config.keysOf?.(params);
    if (keys == null) return undefined;
    const members = this.composableMembersOf(keys);
    if (members == null) return undefined;
    return this.composedOf(this.ensure(params), members);
  }

  /**
   * Resolves a keys-only query's key list against the table: deduped, in params
   * order, tombstoned keys omitted to match the fetch answer. Returns null when any
   * key is unknown, since only a fetch can tell an uncached record from a
   * nonexistent one.
   */
  private composableMembersOf(keys: K[]): K[] | null {
    const { table } = this.config;
    const members: K[] = [];
    const seen = new Set<K>();
    for (const key of keys) {
      if (seen.has(key)) continue;
      seen.add(key);
      const status = table.status(key);
      if (status === "unknown") return null;
      if (status === "present") members.push(key);
    }
    return members;
  }

  /**
   * Subscribes to changes in the query's cached answer. The handler fires with
   * the new answer on every change or deletion. Maintenance for the query runs
   * while at least one subscriber exists and survives the last unsubscribe by
   * the grace window, so a quick remount skips the reconfirm refetch. Returns
   * a destructor that unsubscribes.
   */
  onChange(params: P, handler: ChangeHandler<D>): destructor.Destructor {
    void this.startStreaming();
    const query = this.ensure(params);
    query.handlers.add(handler);
    query.idleSince = undefined;
    if (query.teardown == null) this.maintain(query);
    return () => {
      query.handlers.delete(handler);
      if (query.handlers.size > 0) return;
      if (this.grace.isZero) return this.teardownQuery(query);
      query.idleSince = TimeStamp.now();
      this.armSweep();
    };
  }

  /**
   * Resets every answer to unfetched and notifies subscribers that their
   * answer was invalidated, so the next read refetches. Maintenance
   * subscriptions stay mounted. Called by the cache when the cluster behind
   * the connection is replaced.
   */
  reset(): void {
    this.queries.forEach((query) => {
      if (query.refetchTimer != null) {
        clearTimeout(query.refetchTimer);
        query.refetchTimer = undefined;
      }
      query.state = { variant: "unfetched" };
      query.handlers.forEach((handler) => {
        try {
          handler(undefined);
        } catch (exc) {
          this.report(exc, `failed to notify ${this.config.name} answer subscriber`);
        }
      });
    });
  }

  /** Detaches the epoch subscription. Queries and handlers are dropped. */
  close(): void {
    this.detachEpoch?.();
    if (this.sweepTimer != null) {
      clearTimeout(this.sweepTimer);
      this.sweepTimer = undefined;
    }
    this.queries.forEach((query) => {
      query.teardown?.forEach((d) => d());
      if (query.refetchTimer != null) clearTimeout(query.refetchTimer);
    });
    this.queries.clear();
  }

  private teardownQuery(query: Query<P, K, D, V>): void {
    query.idleSince = undefined;
    query.teardown?.forEach((d) => d());
    query.teardown = undefined;
    query.unmaintained = true;
    query.pendingRechecks = undefined;
    query.refetchOnSettle = false;
    if (query.refetchTimer != null) {
      clearTimeout(query.refetchTimer);
      query.refetchTimer = undefined;
    }
  }

  /** Arms the sweep while any query sits in the grace window. One timer per
   *  space, no matter how many queries go idle at once. */
  private armSweep(): void {
    this.sweepTimer ??= setTimeout(() => {
      this.sweepTimer = undefined;
      this.sweep();
    }, this.grace.milliseconds / 2);
  }

  private sweep(): void {
    let anyIdle = false;
    const now = TimeStamp.now();
    this.queries.forEach((query) => {
      if (query.idleSince == null) return;
      if (query.handlers.size > 0) {
        query.idleSince = undefined;
        return;
      }
      if (now.span(query.idleSince).lessThan(this.grace)) {
        anyIdle = true;
        return;
      }
      this.teardownQuery(query);
    });
    if (anyIdle) this.armSweep();
  }

  private ensure(params: P): Query<P, K, D, V> {
    const hashed = hash(params);
    let query = this.queries.get(hashed);
    if (query == null) {
      query = { params, state: { variant: "unfetched" }, handlers: new Set() };
      this.queries.set(hashed, query);
    }
    return query;
  }

  // Streaming failure must never block reads, so the change stream opens in
  // the background rather than being awaited. A denial belongs to the
  // connection, which reports it once, not to every query that reads.
  // Never rejects.
  private async startStreaming(): Promise<void> {
    try {
      await this.hooks.ensureStreaming?.();
    } catch (exc) {
      if (AccessDeniedError.matches(exc)) return;
      this.report(exc);
    }
  }

  private report(
    exc: unknown,
    message = `failed to maintain ${this.config.name} answers`,
  ): void {
    this.hooks.onError(new Error(message, { cause: exc }));
  }

  /** Composes a ready answer, interned against the records it was built from. */
  private composedOf(query: Query<P, K, D, V>, keys: K[]): D {
    const records = this.config.table.get(keys);
    const { composed } = query;
    if (composed != null && compare.arraysEqual(composed.records, records))
      return composed.value;
    const value = this.config.compose(records, query.params);
    query.composed = { records, value };
    return value;
  }

  private cachedOf(query: Query<P, K, D, V>): Cached<D> | undefined {
    const { state } = query;
    if (state.variant === "ready") return this.composedOf(query, state.keys);
    if (state.variant === "deleted") {
      const tombstone = this.config.table.getTombstone(state.key);
      if (tombstone == null) return undefined;
      return this.deletedOf(tombstone, query.params);
    }
    return undefined;
  }

  /** Composes a rule-1 answer for a read with no live query, interned per entry
   *  and params. The memo dies with the entry it is keyed on, which a table
   *  replaces on every content change. */
  private entryAnswerOf(entry: V, params: P): D {
    if (typeof entry !== "object" || entry === null)
      return this.config.compose([entry], params);
    const hashed = hash(params);
    let perEntry = this.entryAnswers.get(entry);
    if (perEntry == null) {
      perEntry = new Map();
      this.entryAnswers.set(entry, perEntry);
    }
    if (perEntry.has(hashed)) return perEntry.get(hashed) as D;
    const value = this.config.compose([entry], params);
    perEntry.set(hashed, value);
    return value;
  }

  private deletedOf(tombstone: Deleted<V>, params: P): Deleted<D> {
    const hashed = hash(params);
    let perTombstone = this.corpses.get(tombstone);
    if (perTombstone == null) {
      perTombstone = new Map();
      this.corpses.set(tombstone, perTombstone);
    }
    let deleted = perTombstone.get(hashed);
    if (deleted == null) {
      deleted = new Deleted(
        this.config.compose([tombstone.corpse], params),
        tombstone.deletedAt,
      );
      perTombstone.set(hashed, deleted);
    }
    return deleted;
  }

  /** Notifies subscribers when any of the keys is a member of the ready
   *  answer; otherwise just drops the memo so the next read recomposes. */
  private touchMembers(query: Query<P, K, D, V>, keys: K[]): void {
    query.composed = undefined;
    if (query.state.variant !== "ready") return;
    const members = new Set(query.state.keys);
    if (keys.some((key) => members.has(key))) this.touch(query);
  }

  /** Notifies every subscriber with the query's current answer. */
  private touch(query: Query<P, K, D, V>): void {
    // A watched foreign table is an input the records do not carry, so an
    // event is the only signal that a composed answer changed.
    query.composed = undefined;
    if (query.handlers.size === 0) return;
    const result = this.cachedOf(query);
    query.handlers.forEach((handler) => {
      try {
        handler(result);
      } catch (exc) {
        this.report(exc, `failed to notify ${this.config.name} answer subscriber`);
      }
    });
  }

  private settle(
    query: Query<P, K, D, V>,
    expected: QueryState<K, D>,
    next: QueryState<K, D>,
  ): void {
    // A late promise resolution must not clobber a maintenance update.
    if (query.state !== expected) return;
    query.state = next;
    this.drainRechecks(query);
    if (next.variant === "error") {
      query.refetchOnSettle = false;
      return;
    }
    if (query.refetchOnSettle === true) {
      query.refetchOnSettle = false;
      if (query.teardown != null) this.scheduleRefetch(query);
    }
    this.touch(query);
  }

  /**
   * Replays membership changes a fetch in flight deferred. The fetch answers the query
   * as of when it ran, so a change that raced it would otherwise be lost under the keys
   * it publishes. Applies quietly: the settle that calls this notifies once for the
   * whole answer.
   */
  private drainRechecks(query: Query<P, K, D, V>): void {
    const pending = query.pendingRechecks;
    if (pending == null) return;
    query.pendingRechecks = undefined;
    this.applyRechecks(query, pending);
  }

  private fetch(query: Query<P, K, D, V>, options?: FetchOptions): Promise<D> {
    const promise = this.config.fetch(query.params, options).then(
      (keys) => {
        this.settle(query, loading, { variant: "ready", keys });
        return this.composedOf(query, keys);
      },
      (reason: unknown) => {
        const error = errors.fromUnknown(reason);
        this.settle(query, loading, { variant: "error", error });
        throw error;
      },
    );
    const loading: QueryState<K, D> = { variant: "loading", promise };
    query.state = loading;
    return promise;
  }

  /** Never rejects: a failed refetch leaves the previous answer in place. */
  private async refetch(query: Query<P, K, D, V>): Promise<void> {
    if (query.refetchTimer != null) {
      clearTimeout(query.refetchTimer);
      query.refetchTimer = undefined;
    }
    const before = query.state;
    let keys: K[];
    try {
      keys = await this.config.fetch(query.params);
    } catch (reason) {
      if (query.state !== before) return;
      // A refetch revalidates an answer the caller already has. Failing to
      // confirm it is not grounds for taking it away.
      if (before.variant === "ready") return;
      query.state = { variant: "error", error: errors.fromUnknown(reason) };
      return;
    }
    // Only apply if maintenance hasn't been torn down and no newer fetch
    // superseded this one.
    if (query.state !== before) return;
    query.state = { variant: "ready", keys };
    this.touch(query);
  }

  private scheduleRefetch(query: Query<P, K, D, V>): void {
    // A fetch in flight publishes its own keys on settle, and a refetch racing
    // it resolves into a state its result no longer matches, so it is dropped.
    if (query.state.variant === "loading") {
      query.refetchOnSettle = true;
      return;
    }
    if (query.refetchTimer != null) clearTimeout(query.refetchTimer);
    const wait = DEFAULT_DEBOUNCE.milliseconds;
    query.refetchTimer = setTimeout(() => {
      query.refetchTimer = undefined;
      void this.refetch(query);
    }, wait);
  }

  private isServerComputed(params: P): boolean {
    const { serverFields } = this.config;
    if (serverFields == null || serverFields.length === 0) return false;
    if (typeof params !== "object" || params === null) return false;
    return Object.entries(params).some(
      ([field, value]) => value != null && serverFields.includes(field),
    );
  }

  private maintain(query: Query<P, K, D, V>): void {
    const teardown: destructor.Destructor[] = [];
    query.teardown = teardown;
    const { table, keyOf, matches, watch: watches } = this.config;
    const { params } = query;
    // Rules 2 and 3 build membership from events, and an unmaintained query
    // received none, so its answer must be reconfirmed. Rule 1 re-seeds from the
    // table below and needs no network.
    if (
      query.unmaintained === true &&
      query.state.variant === "ready" &&
      keyOf?.(params) == null
    )
      this.scheduleRefetch(query);
    query.unmaintained = false;

    // A keys-only query whose members the table already holds seeds a ready answer:
    // rules 2 and 3 gate notifications on a ready state, so without the seed an
    // answer getCached composes would look live while never updating.
    const composableKeys = this.config.keysOf?.(params);
    if (
      composableKeys != null &&
      query.state.variant !== "loading" &&
      query.state.variant !== "ready"
    ) {
      const members = this.composableMembersOf(composableKeys);
      if (members != null) {
        query.state = { variant: "ready", keys: members };
        this.touch(query);
      }
    }

    if (this.isServerComputed(params) || (keyOf?.(params) == null && matches == null)) {
      // Rule 3: server-computed — any relevant event triggers a debounced
      // wholesale refetch; membership is never patched locally. A change to a
      // current member's entry notifies immediately, so optimistic writes render
      // without waiting on the refetch.
      teardown.push(
        table.subscribeBatch((events) => {
          this.scheduleRefetch(query);
          this.touchMembers(
            query,
            events.map((event) => event.key),
          );
        }),
      );
      watches?.forEach((w) =>
        teardown.push(
          w.attach(params, (result) => {
            this.scheduleRefetch(query);
            if (result !== "refetch") this.touchMembers(query, result);
          }),
        ),
      );
      return;
    }

    const key = keyOf?.(params);
    if (key != null) {
      // Rule 1: exact-key — track one entry; deletion flips the answer. An entry
      // already in the table answers the query without a fetch. The state is a
      // cache of the table's presence for this key and goes stale while
      // unsubscribed (a delete observed, then a recreate), so maintenance
      // re-seeds from the table instead of trusting the last answer. A loading
      // query owns an in-flight promise its own settle will resolve.
      if (query.state.variant !== "loading") {
        const status = table.status(key);
        if (status === "present" && query.state.variant !== "ready") {
          query.state = { variant: "ready", keys: [key] };
          this.touch(query);
        } else if (status === "tombstoned" && query.state.variant !== "deleted") {
          query.state = { variant: "deleted", key };
          this.touch(query);
        }
      }
      teardown.push(
        table.subscribe((event) => {
          if (event.variant === "set") query.state = { variant: "ready", keys: [key] };
          else query.state = { variant: "deleted", key };
          this.touch(query);
        }, key),
      );
      watches?.forEach((w) =>
        teardown.push(
          w.attach(params, (result) => {
            if (result === "refetch") return this.scheduleRefetch(query);
            if (result.includes(key) && query.state.variant === "ready")
              this.touch(query);
          }),
        ),
      );
      return;
    }

    // Rule 2: client-checkable — admit/evict exactly against `matches`.
    teardown.push(
      table.subscribeBatch((events) =>
        this.recheckKeys(
          query,
          events.map((event) => event.key),
        ),
      ),
    );
    watches?.forEach((w) =>
      teardown.push(
        w.attach(params, (result) => {
          if (result === "refetch") return this.scheduleRefetch(query);
          void this.recheckMany(query, result);
        }),
      ),
    );
  }

  /**
   * Applies membership rechecks for the given keys against the table's current entries,
   * without notifying. Admissions append in iteration order; evicting a single space's
   * last member flips the query to deleted or unfetched. Returns whether the answer
   * changed: membership moved, or a member's content was touched.
   */
  private applyRechecks(query: Query<P, K, D, V>, keys: Iterable<K>): boolean {
    if (query.state.variant === "loading") {
      const pending = (query.pendingRechecks ??= new Set<K>());
      for (const key of keys) pending.add(key);
      return false;
    }
    if (query.state.variant !== "ready") return false;
    const { table, matches, single } = this.config;
    const memberSet = new Set(query.state.keys);
    const admitted: K[] = [];
    let lastEvicted: K | null = null;
    let touched = false;
    for (const key of new Set(keys)) {
      const entry = table.get(key);
      const matched = entry != null && matches!(entry, query.params);
      if (matched === memberSet.has(key)) {
        touched ||= matched;
        continue;
      }
      if (matched) {
        memberSet.add(key);
        admitted.push(key);
      } else {
        memberSet.delete(key);
        lastEvicted = key;
      }
    }
    if (admitted.length === 0 && lastEvicted == null) return touched;
    const next = [...query.state.keys.filter((k) => memberSet.has(k)), ...admitted];
    if (single === true && next.length === 0 && lastEvicted != null)
      query.state =
        table.status(lastEvicted) === "tombstoned"
          ? { variant: "deleted", key: lastEvicted }
          : { variant: "unfetched" };
    else query.state = { variant: "ready", keys: next };
    return true;
  }

  /** Applies rechecks for the keys, then notifies once when anything changed. */
  private recheckKeys(query: Query<P, K, D, V>, keys: Iterable<K>): void {
    if (this.applyRechecks(query, keys)) this.touch(query);
  }

  /** Never rejects: a failed backfill is reported and the recheck is skipped. */
  private async recheckMany(query: Query<P, K, D, V>, keys: K[]): Promise<void> {
    const { table } = this.config;
    this.recheckKeys(query, keys);
    const missing = keys.filter((key) => table.status(key) === "unknown");
    if (missing.length === 0) return;
    // Backfill through the table's fetch so membership can be rechecked;
    // fetch-less tables serve cached entries only and the recheck is a no-op.
    try {
      await table.retrieve(missing);
      this.recheckKeys(query, missing);
    } catch (exc) {
      this.report(exc, `failed to hydrate ${this.config.name} answers`);
    }
  }
}
