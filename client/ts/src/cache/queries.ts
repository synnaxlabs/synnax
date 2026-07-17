// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type destructor, errors, primitive } from "@synnaxlabs/x";

import { executeSetter, type SetArg } from "@/cache/state";
import { type Data, type FetchOptions, type Query } from "@/cache/types";
import { NotFoundError } from "@/errors";

/**
 * Deterministically serializes a query to a stable string. Keys are sorted
 * recursively so `{a: 1, b: 2}` and `{b: 2, a: 1}` collapse to the same key.
 * Class instances implementing {@link primitive.Hashable} delegate to their
 * `hash()` method; plain objects and arrays recurse structurally.
 */
export const hashQuery = (query: Query): string => {
  if (query === null) return "null";
  if (query === undefined) return "undefined";
  if (typeof query === "bigint") return `${query.toString()}n`;
  if (typeof query !== "object") return JSON.stringify(query);
  if (primitive.isHashable(query)) return query.hash();
  if (Array.isArray(query)) return `[${query.map(hashQuery).join(",")}]`;
  const entries = Object.entries(query as Record<string, Query>).sort(([a], [b]) =>
    a < b ? -1 : a > b ? 1 : 0,
  );
  return `{${entries.map(([k, v]) => `${JSON.stringify(k)}:${hashQuery(v)}`).join(",")}}`;
};

/** Lifecycle state of a single query. */
export type QueryState<D extends Data> =
  | { variant: "loading"; promise: Promise<D> }
  | { variant: "success"; data: D }
  | { variant: "error"; error: Error }
  | { variant: "deleted"; corpse: D };

/**
 * A cached answer as delivered to consumers: the query's current data, or the
 * last value of a record that was deleted. A corpse is never delivered as bare
 * data, so deletion cannot be mistaken for a live result.
 */
export type Cached<D extends Data> =
  { variant: "changed"; data: D } | { variant: "deleted"; corpse: D };

/**
 * Receives the new cached answer for a query. `undefined` means the answer was
 * invalidated; the next read refetches.
 */
export interface ChangeHandler<D extends Data> {
  (result: Cached<D> | undefined): void;
}

/**
 * Handles passed to a {@link QueriesParams.mount} implementation to keep one
 * query's cached answer fresh.
 */
export interface MountParams<Q extends Query, D extends Data> {
  /** The query whose answer the mounted listeners maintain. */
  query: Q;
  /**
   * Applies a new answer. A setter receives the previous answer, or undefined
   * when none is cached. Producing null/undefined invalidates an existing
   * answer and is a no-op when none is cached.
   */
  update: (value: SetArg<D | null | undefined, D | undefined>) => void;
  /**
   * Transitions the answer to deleted, retaining the given corpse. Falls back
   * to the current answer as the corpse; with neither, invalidates.
   */
  remove: (corpse?: D) => void;
  /** Drops the entry so the next read refetches. */
  invalidate: () => void;
}

export interface QueriesParams<Q extends Query, D extends Data> {
  /** Resource name used in error messages, e.g. "range". */
  name: string;
  /** Fetches the answer from the cluster. Populates record stores itself. */
  fetch: (query: Q, options?: FetchOptions) => Promise<D>;
  /**
   * Mounts freshness listeners for a query. Called when the query gains its
   * first subscriber; returned destructors run when the last one leaves.
   */
  mount?: (
    params: MountParams<Q, D>,
  ) => destructor.Destructor | destructor.Destructor[];
  /** Started (not awaited) on reads and subscriptions to open change delivery. */
  ensureStreaming?: () => Promise<void>;
}

interface MountedListeners {
  refs: number;
  destructors: destructor.Destructor[];
}

/**
 * Cached read path for one query space of a domain client: deduped fetches,
 * lifecycle per query shape, subscriptions, and deletion delivery. Holds no
 * long-lived record data of its own; record stores hold canonical values,
 * while this holds answers keyed by query shape.
 *
 * Domain clients construct one instance per query space and delegate their
 * `retrieve`/`onChange`/`getCached` members to it. Methods accept the query
 * value directly; hashing is internal.
 */
export class Queries<Q extends Query, D extends Data> {
  private readonly entries = new Map<string, QueryState<D>>();
  private readonly handlers = new Map<string, Set<ChangeHandler<D>>>();
  private readonly queriesByHash = new Map<string, Q>();
  private readonly mounted = new Map<string, MountedListeners>();
  private readonly params: QueriesParams<Q, D>;

  constructor(params: QueriesParams<Q, D>) {
    this.params = params;
  }

  /**
   * Returns the answer to the query: instantly when cached and subscribed,
   * joining the in-flight fetch when one exists, fetching otherwise. Settled
   * answers are kept fresh only while subscribed, so an unsubscribed read
   * always refetches. A previously failed query refetches.
   * @throws {NotFoundError} if the queried record was deleted.
   */
  retrieve(query: Q, options?: FetchOptions): Promise<D> {
    const hash = hashQuery(query);
    const entry = this.entries.get(hash);
    if (entry?.variant === "loading") return entry.promise;
    if (this.handlers.has(hash))
      switch (entry?.variant) {
        case "success":
          return Promise.resolve(entry.data);
        case "deleted":
          return Promise.reject(new NotFoundError(`${this.params.name} was deleted`));
      }
    // Streaming failure (e.g. no streamer permission) must never block reads,
    // so the change stream opens in the background rather than being awaited.
    void this.params.ensureStreaming?.().catch(console.error);
    const promise = this.params.fetch(query, options);
    // Set synchronously so a same-tick concurrent retrieve joins this fetch.
    const loading: QueryState<D> = { variant: "loading", promise };
    this.entries.set(hash, loading);
    promise.then(
      (data) => this.replaceIfStill(hash, loading, { variant: "success", data }),
      (reason: unknown) =>
        this.replaceIfStill(hash, loading, {
          variant: "error",
          error: errors.fromUnknown(reason),
        }),
    );
    return promise;
  }

  /**
   * Returns the cached answer without touching the network: the current data,
   * the corpse of a deleted record, or undefined when unfetched, in flight, or
   * failed. May be stale for queries nothing subscribes to.
   */
  getCached(query: Q): Cached<D> | undefined {
    return toCached(this.entries.get(hashQuery(query)));
  }

  /**
   * Subscribes to changes in the query's cached answer. The handler fires with
   * the new answer on every change or deletion, and with undefined when the
   * answer is invalidated. Freshness listeners for the query are mounted while
   * at least one subscriber exists. Returns a destructor that unsubscribes.
   */
  onChange(query: Q, handler: ChangeHandler<D>): destructor.Destructor {
    const hash = hashQuery(query);
    void this.params.ensureStreaming?.().catch(console.error);
    let set = this.handlers.get(hash);
    if (set == null) {
      set = new Set();
      this.handlers.set(hash, set);
    }
    set.add(handler);
    this.mount(hash, query);
    return () => {
      const handlers = this.handlers.get(hash);
      if (handlers == null) return;
      handlers.delete(handler);
      if (handlers.size === 0) this.handlers.delete(hash);
      this.unmount(hash);
    };
  }

  private mount(hash: string, query: Q): void {
    const existing = this.mounted.get(hash);
    if (existing != null) {
      existing.refs++;
      return;
    }
    const { mount } = this.params;
    const destructors =
      mount == null
        ? []
        : array.toArray(
            mount({
              query,
              update: (value) => this.update(hash, value),
              remove: (corpse) => this.remove(hash, corpse),
              invalidate: () => this.invalidate(hash),
            }),
          );
    this.mounted.set(hash, { refs: 1, destructors });
    this.queriesByHash.set(hash, query);
  }

  private unmount(hash: string): void {
    const mounted = this.mounted.get(hash);
    if (mounted == null) return;
    mounted.refs--;
    if (mounted.refs > 0) return;
    mounted.destructors.forEach((d) => d());
    this.mounted.delete(hash);
    this.queriesByHash.delete(hash);
  }

  private update(hash: string, value: SetArg<D | null | undefined, D | undefined>) {
    const entry = this.entries.get(hash);
    const prev = entry?.variant === "success" ? entry.data : undefined;
    const next = executeSetter(value, prev);
    if (next == null) {
      // No cached answer means nothing to invalidate; deleting here would
      // discard an in-flight fetch's settle via replaceIfStill.
      if (prev == null) return;
      return this.invalidate(hash);
    }
    this.entries.set(hash, { variant: "success", data: next });
    this.notify(hash);
  }

  private remove(hash: string, corpse?: D) {
    const entry = this.entries.get(hash);
    const prev = entry?.variant === "success" ? entry.data : undefined;
    const body = corpse ?? prev;
    if (body == null) return this.invalidate(hash);
    this.entries.set(hash, { variant: "deleted", corpse: body });
    this.notify(hash);
  }

  private invalidate(hash: string) {
    if (!this.entries.delete(hash)) return;
    this.notify(hash);
  }

  /**
   * Replace the entry only if it is still the same instance the caller
   * expects: a late promise resolution must not clobber a listener's update.
   */
  private replaceIfStill(hash: string, expected: QueryState<D>, next: QueryState<D>) {
    if (this.entries.get(hash) !== expected) return;
    this.entries.set(hash, next);
    if (next.variant !== "error") this.notify(hash);
  }

  private notify(hash: string): void {
    const handlers = this.handlers.get(hash);
    if (handlers == null) return;
    const result = toCached(this.entries.get(hash));
    handlers.forEach((handler) => handler(result));
  }
}

// Memoized per entry so getCached is referentially stable between changes,
// making it usable as a useSyncExternalStore snapshot.
const cachedReprs = new WeakMap<object, Cached<any>>();

const toCached = <D extends Data>(
  entry: QueryState<D> | undefined,
): Cached<D> | undefined => {
  if (entry == null) return undefined;
  if (entry.variant !== "success" && entry.variant !== "deleted") return undefined;
  let repr = cachedReprs.get(entry);
  if (repr == null) {
    repr =
      entry.variant === "success"
        ? { variant: "changed", data: entry.data }
        : { variant: "deleted", corpse: entry.corpse };
    cachedReprs.set(entry, repr);
  }
  return repr;
};
