// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, type Synnax as Client } from "@synnaxlabs/client";
import { use, useLayoutEffect, useRef } from "react";

export interface RetrieveParams<Query extends query.Params> {
  client: Client;
  query: Query;
}

/**
 * Fetch dedup and settled answers for queries the domain client does not
 * cache. Entries persist until the next fetch of the same query replaces
 * them; an answer the cache serves never populates `settled`. Scoped per
 * client so a settled error never outlives the client whose fetch produced
 * it, and settled errors are dropped when the connection epoch advances.
 */
export interface LocalCache<Data> {
  epoch: number;
  inFlight: Map<string, Promise<Data>>;
  settled: Map<string, { data: Data } | { error: Error }>;
}

export const localFor = <Data>(
  locals: WeakMap<Client, LocalCache<Data>>,
  client: Client,
): LocalCache<Data> => {
  const { epoch } = client.connection.status.details;
  const existing = locals.get(client);
  if (existing == null) {
    const local = { epoch, inFlight: new Map(), settled: new Map() };
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
    promise = retrieve(params).then(
      (data) => {
        // An answer the domain cache serves is read from there on the next
        // render; one that never reaches it is kept locally instead.
        if (getCached?.(params) === undefined) local.settled.set(hash, { data });
        local.inFlight.delete(hash);
        return data;
      },
      (cause: unknown) => {
        const error = new Error(`Failed to retrieve ${name}`, { cause });
        const retried = onFetchError?.(params, { cause, error, hash, local });
        if (retried != null) return retried;
        // A failed fetch writes nothing getCached can serve, so the error must
        // settle locally or the next render refetches forever. A later cache
        // hit short-circuits before this entry is read. A reconnect that
        // landed mid-fetch discards the failure instead.
        if (local.epoch === epoch) local.settled.set(hash, { error });
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
 */
export const usePendingFetch = <Query extends query.Params, Data>(
  q: Query | null,
): PendingFetch<Data> => {
  const pending = useRef<{ query: Query | null; promise: Promise<Data> } | null>(null);
  useLayoutEffect(() => {
    pending.current = null;
  });
  const held = pending.current;
  return {
    promise: held != null && held.query === q ? held.promise : null,
    set: (promise) => {
      pending.current = { query: q, promise };
    },
  };
};

export const suspendOnFetch = <Query extends query.Params, Data>(
  params: RetrieveParams<Query>,
  fetchParams: EnsureFetchParams<Query, Data>,
  pending: PendingFetch<Data>,
): Data => {
  if (pending.promise != null) return use(pending.promise);
  const settled = fetchParams.local.settled.get(query.hash(params.query));
  if (settled != null) {
    if ("error" in settled) throw settled.error;
    return settled.data;
  }
  const promise = ensureFetch(params, fetchParams);
  pending.set(promise);
  return use(promise);
};
