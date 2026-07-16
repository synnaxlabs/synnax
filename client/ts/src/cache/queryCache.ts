// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, errors, primitive } from "@synnaxlabs/x";

import { type Data, type Query } from "@/cache/types";

/// Deterministically serializes a query to a stable string. Keys are sorted
/// recursively so `{a: 1, b: 2}` and `{b: 2, a: 1}` collapse to the same key.
/// Class instances implementing {@link primitive.Hashable} delegate to their
/// `hash()` method; plain objects and arrays recurse structurally.
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

/** Lifecycle state of a single query: in flight, resolved, or failed. */
export type QueryState<D extends Data> =
  | { variant: "loading"; promise: Promise<D> }
  | { variant: "success"; data: D }
  | { variant: "error"; error: Error };

/// Lifecycle cache of in-flight and settled queries for a single retrieve
/// operation. Holds no long-lived data of its own: per-record stores hold
/// canonical values keyed by record ID, while this cache holds query
/// lifecycles keyed by query shape. A loading entry auto-transitions to
/// success or error when its promise settles, gated by identity so that
/// listener-driven mid-flight updates are not clobbered by a late resolution.
///
/// One instance per retrieve operation per client, retrieved via
/// `Engine.getCache`. Methods accept the query value directly; hashing is
/// internal.
export class QueryCache<Q extends Query, D extends Data> {
  private readonly entries = new Map<string, QueryState<D>>();
  private readonly subscribers = new Map<string, Set<() => void>>();

  /// Returns the state for the given query, or undefined if absent.
  get(query: Q): QueryState<D> | undefined {
    return this.entries.get(hashQuery(query));
  }

  /// Stores a state for the given query. A loading state wires up `.then`
  /// handlers that auto-transition the entry to success or error when the
  /// promise settles.
  set(query: Q, state: QueryState<D>): void {
    const hash = hashQuery(query);
    this.entries.set(hash, state);
    this.notify(hash);
    if (state.variant !== "loading") return;
    state.promise.then(
      (data) => this.replaceIfStill(hash, state, { variant: "success", data }),
      (reason: unknown) => {
        this.replaceIfStill(hash, state, {
          variant: "error",
          error: errors.fromUnknown(reason),
        });
      },
    );
  }

  /// Removes the entry for the given query. The next read kicks off a fresh
  /// fetch.
  invalidate(query: Q): void {
    const hash = hashQuery(query);
    if (!this.entries.delete(hash)) return;
    this.notify(hash);
  }

  /// Subscribes to changes for the given query. The callback fires on every
  /// state transition for that query. Returns a destructor that unsubscribes.
  subscribe(query: Q, callback: () => void): destructor.Destructor {
    const hash = hashQuery(query);
    let set = this.subscribers.get(hash);
    if (set == null) {
      set = new Set();
      this.subscribers.set(hash, set);
    }
    set.add(callback);
    return () => {
      const subs = this.subscribers.get(hash);
      if (subs == null) return;
      subs.delete(callback);
      if (subs.size === 0) this.subscribers.delete(hash);
    };
  }

  /// Replace the entry only if it is still the same instance the caller
  /// expects. Guards against a channel listener (or another caller) replacing
  /// the entry mid-flight: a late promise resolution must not clobber a
  /// listener's update.
  private replaceIfStill(hash: string, expected: QueryState<D>, next: QueryState<D>) {
    if (this.entries.get(hash) !== expected) return;
    this.entries.set(hash, next);
    this.notify(hash);
  }

  private notify(hash: string): void {
    this.subscribers.get(hash)?.forEach((cb) => cb());
  }
}
