// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, type Synnax as Client } from "@synnaxlabs/client";
import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { type destructor } from "@synnaxlabs/x";
import { useLayoutEffect, useRef } from "react";

export interface RetrieveParams<Query extends query.Params> {
  client: Client;
  query: Query;
}

interface UseMemoQuery {
  <Query extends query.Params>(q: Query, normalize?: (query: Query) => Query): Query;
  <Query extends query.Params>(
    q: Query | null,
    normalize?: (query: Query) => Query,
  ): Query | null;
}

/** Normalizes the query and stabilizes its identity across renders. */
export const useMemoQuery = (<Query extends query.Params>(
  q: Query | null,
  normalize?: (query: Query) => Query,
): Query | null =>
  useMemoDeepEqual(q == null || normalize == null ? q : normalize(q))) as UseMemoQuery;

/** The answer a fetch settled to: its data, or the error it failed with. */
export type Settled<Data> = { data: Data } | { error: Error };

/**
 * Fetch dedup and settled answers for queries the domain client does not cache. Entries
 * persist until the next fetch of the same query replaces them; an answer the cache
 * serves never populates `settled`. Scoped per client so a settled error never outlives
 * the client whose fetch produced it, and settled errors are dropped when the
 * connection epoch advances. Capped: write through {@link setSettled}, which evicts the
 * oldest entry past the cap.
 */
export interface LocalCache<Data> {
  epoch: number;
  inFlight: Map<string, Promise<Data>>;
  settled: Map<string, Settled<Data>>;
  /** Bumped by {@link invalidate}, so a fetch started before it settles nothing. */
  generations: Map<string, number>;
  /** Readers by query hash. Write through {@link subscribeReader}. */
  listeners: Map<string, Set<() => void>>;
}

const SETTLED_CAP = 256;

/**
 * Inserts a settled entry, evicting the oldest-inserted one once the cap is
 * reached. Settled data pins its answer in memory for the client's lifetime, so an
 * unbounded map leaks across a long session's distinct queries. An evicted entry
 * costs a refetch on the next read, never correctness.
 */
export const setSettled = <Data>(
  local: LocalCache<Data>,
  hash: string,
  entry: Settled<Data>,
): void => {
  const { settled } = local;
  settled.delete(hash);
  settled.set(hash, entry);
  if (settled.size > SETTLED_CAP) {
    const oldest = settled.keys().next().value;
    if (oldest != null) settled.delete(oldest);
  }
  notifyReaders(local, hash);
};

/**
 * Calls `notify` after each fetched answer and each {@link setSettled} for the query
 * hash.
 */
export const subscribeReader = <Data>(
  local: LocalCache<Data>,
  hash: string,
  notify: () => void,
): destructor.Destructor => {
  const readers = local.listeners.get(hash) ?? new Set<() => void>();
  local.listeners.set(hash, readers);
  readers.add(notify);
  return () => {
    readers.delete(notify);
    if (readers.size === 0 && local.listeners.get(hash) === readers)
      local.listeners.delete(hash);
  };
};

/**
 * Discards a query's settled answer and its in-flight fetch, so the next read fetches
 * again and the older fetch's answer is dropped when it lands.
 */
export const invalidate = <Data>(local: LocalCache<Data>, hash: string): void => {
  local.settled.delete(hash);
  local.inFlight.delete(hash);
  local.generations.set(hash, generationOf(local, hash) + 1);
};

/** Returns the query's generation, which {@link invalidate} advances. */
export const generationOf = <Data>(local: LocalCache<Data>, hash: string): number =>
  local.generations.get(hash) ?? 0;

/** Tells the query's readers to re-read the domain cache and the settled entry. */
export const notifyReaders = <Data>(local: LocalCache<Data>, hash: string): void =>
  local.listeners.get(hash)?.forEach((l) => l());

export const localFor = <Data>(
  locals: WeakMap<Client, LocalCache<Data>>,
  client: Client,
): LocalCache<Data> => {
  const { epoch } = client.connection.status.details;
  const existing = locals.get(client);
  if (existing == null) {
    const local: LocalCache<Data> = {
      epoch,
      inFlight: new Map(),
      settled: new Map(),
      generations: new Map(),
      listeners: new Map(),
    };
    locals.set(client, local);
    return local;
  }
  if (existing.epoch !== epoch) {
    existing.epoch = epoch;
    // A reconnected cluster may well answer a query the dead one could not.
    // Settled data survives: it is still the best answer until refetched.
    existing.settled.forEach((entry, hash) => {
      if ("error" in entry) existing.settled.delete(hash);
    });
  }
  return existing;
};

export interface FetchErrorParams<Data> {
  cause: unknown;
  error: Error;
  hash: string;
  local: LocalCache<Data>;
  /** The query's generation when the fetch started. */
  generation: number;
}

export interface EnsureFetchParams<Query extends query.Params, Data> {
  name: string;
  retrieve: (params: RetrieveParams<Query>) => Promise<Data>;
  /**
   * Reports whether the fetched answer lands somewhere the next render reads.
   * An answer that does not is kept in `settled` instead.
   */
  getCached?: (params: RetrieveParams<Query>) => unknown;
  /**
   * Offers a failed fetch a second chance, returning a promise that supersedes
   * the failure or null to settle it.
   */
  onFetchError?: (
    params: RetrieveParams<Query>,
    errorParams: FetchErrorParams<Data>,
  ) => Promise<Data> | null;
  local: LocalCache<Data>;
}

/** Joins the query's in-flight fetch, starting one when none exists. */
export const ensureFetch = <Query extends query.Params, Data>(
  params: RetrieveParams<Query>,
  { name, retrieve, getCached, onFetchError, local }: EnsureFetchParams<Query, Data>,
): Promise<Data> => {
  const hash = query.hash(params.query);
  let promise = local.inFlight.get(hash);
  if (promise == null) {
    const epoch = local.epoch;
    const generation = generationOf(local, hash);
    promise = retrieve(params).then(
      (data) => {
        if (generationOf(local, hash) !== generation) return data;
        // An answer the domain cache serves is read from there; one that never
        // reaches it is kept locally instead.
        if (getCached?.(params) === undefined) setSettled(local, hash, { data });
        else notifyReaders(local, hash);
        local.inFlight.delete(hash);
        return data;
      },
      (cause: unknown) => {
        const error = new Error(`Failed to retrieve ${name}`, { cause });
        if (generationOf(local, hash) !== generation) throw error;
        const retried = onFetchError?.(params, {
          cause,
          error,
          hash,
          local,
          generation,
        });
        if (retried != null) return retried;
        // A failed fetch writes nothing getCached can serve, so the error must
        // settle locally or the next render refetches forever. A later cache
        // hit short-circuits before this entry is read. A reconnect that
        // landed mid-fetch discards the failure instead.
        if (local.epoch === epoch) setSettled(local, hash, { error });
        local.inFlight.delete(hash);
        throw error;
      },
    );
    local.inFlight.set(hash, promise);
  }
  return promise;
};

export interface PendingFetch<Data> {
  /** The promise the attempt being replayed suspended on, if there is one. */
  promise: Promise<Data> | null;
  /** Records the promise this attempt suspends on. */
  set: (promise: Promise<Data>) => void;
}

/**
 * Remembers the promise an attempt suspends on until the render commits. React ends
 * a replayed component's recorded hook list at its `use` call, so an attempt that
 * suspended has to resume through the same promise. Reaching the answer any other
 * way leaves every hook after the read reading a list that has already run out.
 * @returns a new object on each render, so a compiled caller runs again any call that
 * takes it, such as {@link promiseFor}.
 */
export const usePendingFetch = <Query extends query.Params, Data>(
  q: Query | null,
): PendingFetch<Data> => {
  // A replay must read the promise its suspended attempt wrote, which only a ref holds
  // across the replay.
  "use no memo";
  const pending = useRef<{ query: Query | null; promise: Promise<Data> } | null>(null);
  useLayoutEffect(() => {
    pending.current = null;
  });
  /* eslint-disable synnax/react-compiler */
  const held = pending.current;
  return {
    promise: held != null && held.query === q ? held.promise : null,
    set: (promise) => {
      pending.current = { query: q, promise };
    },
  };
  /* eslint-enable synnax/react-compiler */
};

/**
 * @returns a promise already fulfilled with the value, so `use` reads it without
 * suspending. A hook that suspended must call `use` again when it finishes, or React
 * flags the read as a conditional `use`. Call `use` in the hook itself: the compiler
 * caches a call to a plain function, which would skip a `use` inside it.
 */
export const fulfilled = <Data>(value: Data): Promise<Data> =>
  Object.assign(Promise.resolve(value), { status: "fulfilled" as const, value });

/**
 * @returns the promise a suspending read passes to `use`: the one the replayed attempt
 * suspended on, the settled answer, or a new fetch.
 * @throws the error the query last settled to.
 */
export const promiseFor = <Query extends query.Params, Data>(
  params: RetrieveParams<Query>,
  fetchParams: EnsureFetchParams<Query, Data>,
  pending: PendingFetch<Data>,
): Promise<Data> => {
  if (pending.promise != null) return pending.promise;
  const settled = fetchParams.local.settled.get(query.hash(params.query));
  if (settled != null) {
    if ("error" in settled) throw settled.error;
    return fulfilled(settled.data);
  }
  const promise = ensureFetch(params, fetchParams);
  pending.set(promise);
  return promise;
};
