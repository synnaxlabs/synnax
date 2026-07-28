// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type destructor,
  errors,
  primitive,
  type record,
  type state,
  TimeSpan,
} from "@synnaxlabs/x";

import { NotFoundError } from "@/errors";
import { Deleted } from "@/query/deleted";
import { type Table, type TableEvent } from "@/query/table";
import { type Data, type FetchOptions, type Params } from "@/query/types";

/**
 * Deterministically serializes a query to a stable string. Keys are sorted
 * recursively so `{a: 1, b: 2}` and `{b: 2, a: 1}` collapse to the same key,
 * and explicitly-undefined fields hash like absent ones (matching JSON
 * semantics). Class instances implementing {@link primitive.Hashable}
 * delegate to their `hash()` method; plain objects and arrays recurse
 * structurally.
 */
export const hash = (query: Params): string => {
  if (query === null) return "null";
  if (query === undefined) return "undefined";
  if (typeof query === "bigint") return `${query.toString()}n`;
  if (typeof query !== "object") return JSON.stringify(query);
  if (primitive.isHashable(query)) return query.hash();
  if (Array.isArray(query)) return `[${query.map(hash).join(",")}]`;
  const entries = Object.entries(query)
    .filter(([, v]) => v !== undefined)
    .sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0));
  return `{${entries.map(([k, v]) => `${JSON.stringify(k)}:${hash(v)}`).join(",")}}`;
};

/**
 * A cached answer as delivered to consumers: the query's current data, or the
 * {@link Deleted} corpse of a record that was deleted. A corpse is never
 * delivered as bare data, so deletion cannot be mistaken for a live result.
 * Narrow with {@link Deleted.matches}.
 */
export type Cached<D extends Data> = D | Deleted<D>;

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
export interface Retrieves<Q extends Params, D extends Data> {
  retrieve: (query: Q, options?: FetchOptions) => Promise<D>;
  onChange: (query: Q, handler: ChangeHandler<D>) => destructor.Destructor;
  getCached: (query: Q) => Cached<D> | undefined;
}

/**
 * A foreign-table subscription for an answer space: projects that table's
 * events onto the space's primary keys. Build with {@link watch}.
 */
export interface WatchEntry<Q extends Params, K extends record.Key> {
  attach: (
    query: Q,
    onEvent: (result: K[] | "refetch") => void,
  ) => destructor.Destructor;
}

/**
 * Declares that events on the given table affect answers in this space.
 * `affects` maps an event to the primary keys whose membership or content it
 * touches, "refetch" to invalidate wholesale, or null when unaffected.
 */
export const watch = <
  Q extends Params,
  K extends record.Key,
  FK extends record.Key,
  FV extends state.State,
>(
  table: Table<FK, FV>,
  affects: (event: TableEvent<FK, FV>, query: Q) => K[] | "refetch" | null,
): WatchEntry<Q, K> => ({
  attach: (query, onEvent) =>
    table.subscribe((event) => {
      const result = affects(event, query);
      if (result == null) return;
      if (result === "refetch" || result.length > 0) onEvent(result);
    }),
});

/**
 * Declaration of one query space: which table owns its records, how to fetch
 * an answer's keys, how to assemble content, and how membership is maintained.
 *
 * Maintenance is chosen per query instance:
 * 1. exact-key — `keyOf` returns a key: track that row; deletion flips the
 *    answer to deleted.
 * 2. client-checkable — `matches` compares a record against the query:
 *    admit/evict exactly, no network.
 * 3. server-computed — any field named in `serverFields` is set on the query
 *    (or neither `keyOf` nor `matches` applies): debounced wholesale refetch.
 */
export interface QueriesParams<
  Q extends Params,
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
  fetch: (query: Q, options?: FetchOptions) => Promise<K[]>;
  /** Assembles the answer from resolved member records at read time. */
  compose: (records: V[], query: Q) => D;
  /** Rule 1: the single key a query addresses, or null when it doesn't. */
  keyOf?: (query: Q) => K | null;
  /** Rule 2: whether a record satisfies the query. Pure; no network. */
  matches?: (record: V, query: Q) => boolean;
  /**
   * Query fields only the server can evaluate (searchTerm, limit, offset).
   * A query instance with any of them set is maintained by rule 3.
   */
  serverFields?: readonly string[];
  /** Foreign tables whose events affect this space's answers. */
  watch?: WatchEntry<Q, K>[];
  /**
   * Marks a space whose answer addresses one record and cannot compose an
   * empty membership: rule-2 eviction of the last member flips the entry to
   * deleted (when the row tombstoned) or invalidates it, instead of
   * composing []. Leave unset for spaces where [] is a valid answer.
   */
  single?: boolean;
}

/** Wiring an {@link Queries} space receives from the cache that owns it. */
export interface AnswersHooks {
  /** Started (not awaited) on reads and subscriptions to open change delivery. */
  ensureStreaming?: () => Promise<void>;
  /** Subscribes to connection-epoch changes; maintained answers refetch on bump. */
  onEpoch?: (callback: (epoch: number) => void) => destructor.Destructor;
  /** Reports maintenance errors that have no caller to throw to. */
  onError?: (error: Error) => void;
}

const DEFAULT_DEBOUNCE = TimeSpan.milliseconds(100);

type EntryState<K extends record.Key, D extends Data> =
  | { variant: "unfetched" }
  | { variant: "loading"; promise: Promise<D> }
  | { variant: "ready"; keys: K[] }
  | { variant: "error"; error: Error }
  | { variant: "deleted"; key: K };

interface Entry<Q extends Params, K extends record.Key, D extends Data> {
  query: Q;
  state: EntryState<K, D>;
  handlers: Set<ChangeHandler<D>>;
  /** Rule teardown; present exactly while the entry is maintained. */
  teardown?: destructor.Destructor[];
  refetchTimer?: ReturnType<typeof setTimeout>;
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
export class Queries<
  Q extends Params,
  D extends Data,
  K extends record.Key = record.Key,
  V extends state.State = state.State,
> {
  private readonly entries = new Map<string, Entry<Q, K, D>>();
  /** Composed corpses interned per tombstone and query, so deleted answers
   *  stay referentially stable. Identity composes reuse the tombstone. */
  private readonly corpses = new WeakMap<Deleted<V>, Map<string, Deleted<D>>>();
  private readonly params: QueriesParams<Q, D, K, V>;
  private readonly hooks: AnswersHooks;
  private readonly detachEpoch?: destructor.Destructor;

  constructor(params: QueriesParams<Q, D, K, V>, hooks: AnswersHooks = {}) {
    this.params = params;
    this.hooks = hooks;
    this.detachEpoch = hooks.onEpoch?.((epoch) => {
      // 0 is a return to cold (cluster replacement): the fresh stream's own
      // epoch bump refetches, not the reset itself.
      if (epoch === 0) return;
      this.entries.forEach((entry) => {
        if (entry.teardown != null) this.refetch(entry);
      });
    });
  }

  /**
   * Returns the answer to the query: instantly when cached and subscribed,
   * joining the in-flight fetch when one exists, fetching otherwise. Settled
   * answers are kept fresh only while subscribed, so an unsubscribed read
   * always refetches. A previously failed query refetches.
   * @throws {NotFoundError} if the queried record was deleted.
   */
  retrieve(query: Q, options?: FetchOptions): Promise<D> {
    const entry = this.ensure(query);
    const { state } = entry;
    if (state.variant === "loading") return state.promise;
    if (entry.teardown != null)
      switch (state.variant) {
        case "ready":
          return Promise.resolve(this.compose(state.keys, entry.query));
        case "deleted":
          return Promise.reject(new NotFoundError(`${this.params.name} was deleted`));
      }
    this.startStreaming();
    return this.fetch(entry, options);
  }

  /**
   * Returns the cached answer without touching the network: the current data,
   * the corpse of a deleted record, or undefined when nothing is cached.
   * Exact-key queries also resolve from a table row cached by any other
   * query. May be stale for queries nothing subscribes to.
   */
  getCached(query: Q): Cached<D> | undefined {
    const entry = this.entries.get(hash(query));
    if (entry != null) {
      const cached = this.cachedOf(entry);
      if (cached != null) return cached;
    }
    // Rule-1 queries read straight through to the table: a row (or tombstone)
    // answers the query even before any fetch or subscription exists.
    const key = this.params.keyOf?.(query);
    if (key == null) return undefined;
    const { table } = this.params;
    const row = table.get(key);
    if (row != null) return this.params.compose([row], query);
    const tombstone = table.getTombstone(key);
    if (tombstone == null) return undefined;
    return this.deletedOf(tombstone, query);
  }

  /**
   * Subscribes to changes in the query's cached answer. The handler fires with
   * the new answer on every change or deletion. Maintenance for the query runs
   * while at least one subscriber exists. Returns a destructor that
   * unsubscribes.
   */
  onChange(query: Q, handler: ChangeHandler<D>): destructor.Destructor {
    this.startStreaming();
    const entry = this.ensure(query);
    entry.handlers.add(handler);
    if (entry.teardown == null) this.maintain(entry);
    return () => {
      entry.handlers.delete(handler);
      if (entry.handlers.size > 0) return;
      entry.teardown?.forEach((d) => d());
      entry.teardown = undefined;
      if (entry.refetchTimer != null) {
        clearTimeout(entry.refetchTimer);
        entry.refetchTimer = undefined;
      }
    };
  }

  /**
   * Resets every answer to unfetched and notifies subscribers that their
   * answer was invalidated, so the next read refetches. Maintenance
   * subscriptions stay mounted. Called by the cache when the cluster behind
   * the connection is replaced.
   */
  reset(): void {
    this.entries.forEach((entry) => {
      if (entry.refetchTimer != null) {
        clearTimeout(entry.refetchTimer);
        entry.refetchTimer = undefined;
      }
      entry.state = { variant: "unfetched" };
      entry.handlers.forEach((handler) => {
        try {
          handler(undefined);
        } catch (exc) {
          this.report(exc, `failed to notify ${this.params.name} answer subscriber`);
        }
      });
    });
  }

  /** Detaches the epoch subscription. Entries and handlers are dropped. */
  close(): void {
    this.detachEpoch?.();
    this.entries.forEach((entry) => {
      entry.teardown?.forEach((d) => d());
      if (entry.refetchTimer != null) clearTimeout(entry.refetchTimer);
    });
    this.entries.clear();
  }

  private ensure(query: Q): Entry<Q, K, D> {
    const hashed = hash(query);
    let entry = this.entries.get(hashed);
    if (entry == null) {
      entry = { query, state: { variant: "unfetched" }, handlers: new Set() };
      this.entries.set(hashed, entry);
    }
    return entry;
  }

  private startStreaming(): void {
    // Streaming failure (e.g. no streamer permission) must never block reads,
    // so the change stream opens in the background rather than being awaited.
    void this.hooks.ensureStreaming?.().catch((exc: unknown) => this.report(exc));
  }

  private report(
    exc: unknown,
    message = `failed to maintain ${this.params.name} answers`,
  ): void {
    const error = new Error(message, { cause: exc });
    if (this.hooks.onError != null) this.hooks.onError(error);
    else console.error(error);
  }

  private compose(keys: K[], query: Q): D {
    const records = this.params.table.get(keys);
    return this.params.compose(records, query);
  }

  private cachedOf(entry: Entry<Q, K, D>): Cached<D> | undefined {
    const { state } = entry;
    if (state.variant === "ready") return this.compose(state.keys, entry.query);
    if (state.variant === "deleted") {
      const tombstone = this.params.table.getTombstone(state.key);
      if (tombstone == null) return undefined;
      return this.deletedOf(tombstone, entry.query);
    }
    return undefined;
  }

  private deletedOf(tombstone: Deleted<V>, query: Q): Deleted<D> {
    const hashed = hash(query);
    let perTombstone = this.corpses.get(tombstone);
    if (perTombstone == null) {
      perTombstone = new Map();
      this.corpses.set(tombstone, perTombstone);
    }
    let deleted = perTombstone.get(hashed);
    if (deleted == null) {
      deleted = new Deleted(
        this.params.compose([tombstone.corpse], query),
        tombstone.deletedAt,
      );
      perTombstone.set(hashed, deleted);
    }
    return deleted;
  }

  /** Notifies every subscriber with the entry's current answer. */
  private touch(entry: Entry<Q, K, D>): void {
    if (entry.handlers.size === 0) return;
    const result = this.cachedOf(entry);
    entry.handlers.forEach((handler) => {
      try {
        handler(result);
      } catch (exc) {
        this.report(exc, `failed to notify ${this.params.name} answer subscriber`);
      }
    });
  }

  private settle(
    entry: Entry<Q, K, D>,
    expected: EntryState<K, D>,
    next: EntryState<K, D>,
  ): void {
    // A late promise resolution must not clobber a maintenance update.
    if (entry.state !== expected) return;
    entry.state = next;
    if (next.variant !== "error") this.touch(entry);
  }

  private fetch(entry: Entry<Q, K, D>, options?: FetchOptions): Promise<D> {
    const promise = this.params.fetch(entry.query, options).then(
      (keys) => {
        this.settle(entry, loading, { variant: "ready", keys });
        return this.compose(keys, entry.query);
      },
      (reason: unknown) => {
        const error = errors.fromUnknown(reason);
        this.settle(entry, loading, { variant: "error", error });
        throw error;
      },
    );
    const loading: EntryState<K, D> = { variant: "loading", promise };
    entry.state = loading;
    return promise;
  }

  private refetch(entry: Entry<Q, K, D>): void {
    if (entry.refetchTimer != null) {
      clearTimeout(entry.refetchTimer);
      entry.refetchTimer = undefined;
    }
    const before = entry.state;
    void this.params.fetch(entry.query).then(
      (keys) => {
        // Only apply if maintenance hasn't been torn down and no newer fetch
        // superseded this one.
        if (entry.state !== before) return;
        entry.state = { variant: "ready", keys };
        this.touch(entry);
      },
      (reason: unknown) => {
        if (entry.state !== before) return;
        entry.state = { variant: "error", error: errors.fromUnknown(reason) };
      },
    );
  }

  private scheduleRefetch(entry: Entry<Q, K, D>): void {
    if (entry.refetchTimer != null) clearTimeout(entry.refetchTimer);
    const wait = DEFAULT_DEBOUNCE.milliseconds;
    entry.refetchTimer = setTimeout(() => {
      entry.refetchTimer = undefined;
      this.refetch(entry);
    }, wait);
  }

  private isServerComputed(query: Q): boolean {
    const { serverFields } = this.params;
    if (serverFields == null || serverFields.length === 0) return false;
    if (typeof query !== "object" || query === null) return false;
    return Object.entries(query).some(
      ([field, value]) => value != null && serverFields.includes(field),
    );
  }

  private maintain(entry: Entry<Q, K, D>): void {
    const teardown: destructor.Destructor[] = [];
    entry.teardown = teardown;
    const { table, keyOf, matches, watch: watches } = this.params;
    const query = entry.query;

    if (this.isServerComputed(query) || (keyOf?.(query) == null && matches == null)) {
      // Rule 3: server-computed — any relevant event triggers a debounced
      // wholesale refetch; membership is never patched locally.
      teardown.push(table.subscribe(() => this.scheduleRefetch(entry)));
      watches?.forEach((w) =>
        teardown.push(w.attach(query, () => this.scheduleRefetch(entry))),
      );
      return;
    }

    const key = keyOf?.(query);
    if (key != null) {
      // Rule 1: exact-key — track one row; deletion flips the answer. A row
      // already in the table answers the query without a fetch.
      if (entry.state.variant === "unfetched" || entry.state.variant === "error") {
        const status = table.status(key);
        if (status === "present") {
          entry.state = { variant: "ready", keys: [key] };
          this.touch(entry);
        } else if (status === "tombstoned") {
          entry.state = { variant: "deleted", key };
          this.touch(entry);
        }
      }
      teardown.push(
        table.subscribe((event) => {
          if (event.variant === "set") entry.state = { variant: "ready", keys: [key] };
          else entry.state = { variant: "deleted", key };
          this.touch(entry);
        }, key),
      );
      watches?.forEach((w) =>
        teardown.push(
          w.attach(query, (result) => {
            if (result === "refetch") return this.scheduleRefetch(entry);
            if (result.includes(key) && entry.state.variant === "ready")
              this.touch(entry);
          }),
        ),
      );
      return;
    }

    // Rule 2: client-checkable — admit/evict exactly against `matches`.
    teardown.push(table.subscribe((event) => this.recheck(entry, event.key)));
    watches?.forEach((w) =>
      teardown.push(
        w.attach(query, (result) => {
          if (result === "refetch") return this.scheduleRefetch(entry);
          this.recheckMany(entry, result);
        }),
      ),
    );
  }

  private evict(entry: Entry<Q, K, D>, keys: K[], key: K): void {
    const next = keys.filter((k) => k !== key);
    if (this.params.single === true && next.length === 0)
      if (this.params.table.status(key) === "tombstoned")
        entry.state = { variant: "deleted", key };
      else entry.state = { variant: "unfetched" };
    else entry.state = { variant: "ready", keys: next };
    this.touch(entry);
  }

  private recheck(entry: Entry<Q, K, D>, key: K): void {
    if (entry.state.variant !== "ready") return;
    const { table, matches } = this.params;
    const keys = entry.state.keys;
    const member = keys.includes(key);
    const rec = table.get(key);
    if (rec == null) {
      if (!member) return;
      return this.evict(entry, keys, key);
    }
    const m = matches!(rec, entry.query);
    if (m && !member) {
      entry.state = { variant: "ready", keys: [...keys, key] };
      return this.touch(entry);
    }
    if (!m && member) return this.evict(entry, keys, key);
    if (m && member) this.touch(entry);
  }

  private recheckMany(entry: Entry<Q, K, D>, keys: K[]): void {
    const { table } = this.params;
    keys.forEach((key) => this.recheck(entry, key));
    const missing = keys.filter((key) => table.status(key) === "unknown");
    if (missing.length === 0) return;
    // Backfill through the table's fetch so membership can be rechecked;
    // fetch-less tables serve cached rows only and the recheck is a no-op.
    table
      .retrieve(missing)
      .then(() => missing.forEach((key) => this.recheck(entry, key)))
      .catch((exc: unknown) =>
        this.report(exc, `failed to hydrate ${this.params.name} answers`),
      );
  }
}
