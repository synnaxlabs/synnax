// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, primitive, type record, type state } from "@synnaxlabs/x";

import { NotFoundError } from "@/errors";
import { Deleted } from "@/query/deleted";
import type { Table, TableEvent } from "@/query/table";
import type { Data, FetchOptions, Params } from "@/query/types";

const hashes = new WeakMap<object, string>();

/**
 * Deterministically serializes query params to a stable string. Keys are sorted
 * recursively so `{a: 1, b: 2}` and `{b: 2, a: 1}` collapse to the same key, and
 * explicitly-undefined fields hash like absent ones (matching JSON semantics). Class
 * instances implementing {@link primitive.Hashable} delegate to their `hash()` method;
 * plain objects and arrays recurse structurally, memoized per object identity ({@link
 * Params} is readonly, so an object's hash never changes).
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

/** The answer of one query, kept current while a handler is subscribed. */
export interface Live<D extends Data> {
  /**
   * Returns the answer. Serves the cache while the answer is fresh, and fetches
   * otherwise.
   */
  retrieve: (options?: FetchOptions) => Promise<D>;
  /**
   * Returns the cached answer without touching the network, or undefined when nothing
   * is cached. May be stale while no handler is subscribed.
   */
  getCached: () => Cached<D> | undefined;
  /**
   * Calls the handler with each new answer. The answer stays current while a handler is
   * subscribed, and for a grace period after the last one leaves. Returns a destructor
   * that unsubscribes.
   */
  onChange: (handler: ChangeHandler<D>) => destructor.Destructor;
}

/**
 * The read surface of one query space: fetch once, or watch. The public face of every
 * answer space, including the named spaces domain clients expose (parent, kv).
 */
export interface Retrieves<P extends Params, D extends Data> {
  retrieve: (params: P, options?: FetchOptions) => Promise<D>;
  watch: (params: P) => Live<D>;
}

/**
 * A foreign-table subscription for an answer space: projects that table's
 * events onto the space's primary keys. Build with {@link trigger}.
 */
export interface Trigger<P extends Params, K extends record.Key> {
  /**
   * Starts delivering the table's events to a maintained answer.
   * @param members - Returns the current members of the answer.
   * @returns a destructor that stops delivery when the answer stops being maintained.
   */
  attach: (
    params: P,
    onEvent: (result: K[] | "refetch") => void,
    members: () => readonly K[],
  ) => destructor.Destructor;
  /** The foreign table's {@link Table.version}. */
  version: () => number;
  /**
   * Whether the table's changes leave the matches of every query as they were, so a
   * change composes the members again instead of fetching the matches.
   */
  contentOnly?: boolean;
  /**
   * Whether the table holds what an answer with the given members composes. Defaults
   * to true. A cached answer that a trigger does not cover is not served, so its query
   * fetches.
   */
  covers?: (params: P, members: readonly K[]) => boolean;
}

/**
 * Declares that events on the given table affect answers in this space.
 * `affects` maps an event to the primary keys whose membership or content it
 * touches, "refetch" to invalidate wholesale, or null when unaffected. Set
 * `contentOnly` when no query matches on the table's records.
 */
export const trigger = <
  P extends Params,
  K extends record.Key,
  ForeignKey extends record.Key,
  ForeignValue extends state.State,
>(
  table: Pick<Table<ForeignKey, ForeignValue>, "subscribeBatch" | "version">,
  affects: (
    event: TableEvent<ForeignKey, ForeignValue>,
    params: P,
  ) => K[] | "refetch" | null,
  { contentOnly = false }: { contentOnly?: boolean } = {},
): Trigger<P, K> => ({
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
  version: () => table.version(),
  contentOnly,
});
