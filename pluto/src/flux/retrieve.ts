// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DisconnectedError,
  isConnectionError,
  NotFoundError,
  query,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { type destructor, state, TimeSpan } from "@synnaxlabs/x";
import { use, useCallback, useRef, useState, useSyncExternalStore } from "react";

import { DeletedError } from "@/flux/errors";
import {
  errorResult,
  loadingResult,
  nullClientResult,
  type Result,
  successResult,
} from "@/flux/result";
import { useAsyncEffect } from "@/hooks";
import { useDestructors } from "@/hooks/useDestructors";
import { useMemoDeepEqual } from "@/memo";
import { useAdder } from "@/status/base/Aggregator";
import { Synnax } from "@/synnax";

// Bound at module scope: hooks bind `query` to the caller's params object.
const { Deleted, isLive } = query;

export interface RetrieveParams<Query extends query.Params> {
  client: Client;
  query: Query;
}

/**
 * Binds a query definition onto its domain client's read surface: fetch =
 * `retrieve`, live updates = `subscribe` (the client's onChange), snapshot =
 * `getCached`. Flux holds no cache of its own; queries without `subscribe`
 * and `getCached` fetch on every mount and never receive live updates.
 */
export interface CreateRetrieveParams<
  Query extends query.Params,
  Data extends query.Data,
> {
  name: string;
  retrieve: (params: RetrieveParams<Query>) => Promise<Data>;
  subscribe?: (
    params: RetrieveParams<Query>,
    handler: query.ChangeHandler<Data>,
  ) => destructor.Destructor;
  getCached?: (params: RetrieveParams<Query>) => query.Cached<Data> | undefined;
  /**
   * Builds the answer synchronously from records already cached under other
   * queries. Consulted only when `getCached` misses, so suspending reads
   * resolve without a fetch. Returns undefined to fall through to `retrieve`.
   */
  deriveCached?: (params: RetrieveParams<Query>) => Data | undefined;
  /**
   * Holds the previous answer whenever the next one compares equal, so suspending
   * readers re-render only on changes the answer expresses. Required when `getCached`
   * builds a value rather than returning the domain client's own cached reference:
   * a fresh object every call breaks `useSyncExternalStore`.
   */
  equal?: (prev: Data, next: Data) => boolean;
}

export interface BeforeRetrieveParams<Query extends query.Params> {
  query: Query;
}

export interface UseObservableBaseRetrieveParams<
  Query extends query.Params,
  Data extends state.State,
> {
  addStatusOnFailure?: boolean;
  beforeRetrieve?: (params: BeforeRetrieveParams<Query>) => Data | boolean;
  onChange: (result: state.SetArg<Result<Data>>, query: Query) => void;
}

export interface UseRetrieveObservableParams<
  Query extends query.Params,
  Data extends state.State,
> extends Omit<UseObservableBaseRetrieveParams<Query, Data>, "onChange"> {
  onChange: (result: Result<Data>, query: Query) => void;
}

export interface UseRetrieveObservableReturn<Query extends query.Params> {
  retrieve: (
    query: state.SetArg<Query, Partial<Query>>,
    options?: query.FetchOptions,
  ) => void;
  retrieveAsync: (
    query: state.SetArg<Query, Partial<Query>>,
    options?: query.FetchOptions,
  ) => Promise<void>;
}

export type UseRetrieveStatefulReturn<
  Query extends query.Params,
  Data extends state.State,
> = Result<Data> & UseRetrieveObservableReturn<Query>;

export interface UseDirectRetrieveParams<
  Query extends query.Params,
  Data extends state.State,
> extends Pick<
  UseObservableBaseRetrieveParams<Query, Data>,
  "beforeRetrieve" | "addStatusOnFailure"
> {
  query: Query;
}

export type UseDirectRetrieveReturn<Data extends state.State> = Result<Data>;

export interface UseRetrieveEffectParams<
  Query extends query.Params,
  Data extends state.State,
> extends Pick<
  UseObservableBaseRetrieveParams<Query, Data>,
  "beforeRetrieve" | "addStatusOnFailure"
> {
  onChange?: (result: Result<Data>, query: Query) => void;
  query?: Query;
}

export interface UseRetrieve<Query extends query.Params, Data extends state.State> {
  (
    params: Query,
    opts?: Omit<UseDirectRetrieveParams<Query, Data>, "query">,
  ): UseDirectRetrieveReturn<Data>;
}

export interface UseRetrieveEffect<
  Query extends query.Params,
  Data extends state.State,
> {
  (params: UseRetrieveEffectParams<Query, Data>): void;
}

export interface UseRetrieveStateful<
  Query extends query.Params,
  Data extends state.State,
> {
  (): UseRetrieveStatefulReturn<Query, Data>;
}

export interface UseRetrieveObservable<
  Query extends query.Params,
  Data extends state.State,
> {
  (
    params: UseRetrieveObservableParams<Query, Data>,
  ): UseRetrieveObservableReturn<Query>;
}

/// A Suspense-shaped retrieve hook that returns the data directly. The hook
/// either returns the resolved value or suspends on the in-flight promise.
/// Concurrent reads of the same query share one fetch via the domain client's
/// query cache. Client-side listeners push new values into the cache, which
/// notifies subscribed consumers without re-suspending.
export interface UseSuspendedRetrieve<
  Query extends query.Params,
  Data extends state.State,
> {
  (query: Query): Data;
}

/// A Suspense-shaped hook that ensures a query has been retrieved into the
/// cache, then forgets about it: no cache subscription, no listener mounting.
/// Returns nothing; the data lives in the cache for children to read.
///
/// This is a narrow tool. Prefer letting children suspend independently inside
/// their own Suspense boundaries (the Twitter-shell pattern: parent renders
/// immediately, each child shows its own loading state). Reach for this hook
/// only when one of:
///
///   1. The parent must branch on the data before deciding which subtree to
///      render (e.g. resource-type dispatch).
///   2. Multiple children would otherwise issue the same query and each
///      suspend independently; pre-warming the cache here collapses them into
///      a single fetch.
///
/// Reach for `useRetrieveSuspended` instead if the caller itself needs the
/// data or needs to re-render when it changes.
export interface UseEnsureRetrieved<Query extends query.Params> {
  (query: Query): void;
}

/**
 * Returns a callback discarding a query's settled answer so the next suspending
 * read fetches again. A settled failure re-throws on every render, so resetting
 * an error boundary alone lands straight back on it.
 */
export interface UseInvalidate<Query extends query.Params> {
  (): (query: Query) => void;
}

export interface CreateRetrieveReturn<
  Query extends query.Params,
  Data extends state.State,
> {
  useRetrieve: UseRetrieve<Query, Data>;
  useRetrieveEffect: UseRetrieveEffect<Query, Data>;
  useRetrieveStateful: UseRetrieveStateful<Query, Data>;
  useRetrieveObservable: UseRetrieveObservable<Query, Data>;
  useRetrieveSuspended: UseSuspendedRetrieve<Query, Data>;
  useEnsureRetrieved: UseEnsureRetrieved<Query>;
  useInvalidate: UseInvalidate<Query>;
}

const initialResult = <Data extends state.State>(name: string): Result<Data> =>
  loadingResult<Data>(`Retrieving ${name}`, undefined);

const useStateful = <Query extends query.Params, Data extends query.Data>(
  createParams: CreateRetrieveParams<Query, Data> &
    Pick<
      UseObservableBaseRetrieveParams<Query, Data>,
      "beforeRetrieve" | "addStatusOnFailure"
    >,
): UseRetrieveStatefulReturn<Query, Data> => {
  const [state, setState] = useState<Result<Data>>(
    initialResult<Data>(createParams.name),
  );
  return {
    ...state,
    ...useObservableBase({ ...createParams, onChange: setState }),
  };
};

const useObservableBase = <Query extends query.Params, Data extends query.Data>({
  retrieve,
  subscribe,
  name,
  onChange,
  beforeRetrieve,
  addStatusOnFailure = true,
}: UseObservableBaseRetrieveParams<Query, Data> &
  CreateRetrieveParams<Query, Data>): UseRetrieveObservableReturn<Query> => {
  const client = Synnax.use();
  const queryRef = useRef<Query | null>(null);
  const listeners = useDestructors();
  const addStatus = useAdder();
  const handleCacheChange = useCallback(
    (result: query.Cached<Data> | undefined, query: Query) => {
      if (result === undefined) return;
      if (Deleted.matches<Data>(result))
        onChange(
          errorResult(
            `retrieve ${name}`,
            new DeletedError(`${name} was deleted`, result.corpse),
          ),
          query,
        );
      else onChange(successResult(`retrieved ${name}`, result), query);
    },
    [onChange, name],
  );
  const retrieveAsync = useCallback(
    async (
      querySetter: state.SetArg<Query, Partial<Query>>,
      options: query.FetchOptions = {},
    ) => {
      const { signal } = options;
      const query = state.executeSetter<Query, Partial<Query>>(
        querySetter,
        queryRef.current ?? {},
      );
      queryRef.current = query;
      try {
        if (beforeRetrieve != null) {
          const result = beforeRetrieve({ query });
          if (result == false) return;
          if (result !== true) {
            onChange(successResult(`retrieved ${name}`, result), query);
            return;
          }
        }
        if (client == null)
          return onChange(nullClientResult<Data>(`retrieve ${name}`), query);
        onChange((p) => loadingResult(`retrieving ${name}`, p.data), query);
        if (signal?.aborted) return;
        const params = { client, query };
        listeners.cleanup();
        const value = await retrieve(params);
        if (signal?.aborted) return;
        // Subscribing after the fetch keeps mount-time reads fresh: an
        // unsubscribed retrieve always refetches, a subscribed one is served
        // from the cache. A newer retrieve started while this one was in flight
        // is the one whose subscription stays mounted.
        if (subscribe != null && queryRef.current === query)
          listeners.set(
            subscribe(params, (result) => handleCacheChange(result, query)),
          );
        onChange(successResult<Data>(`retrieved ${name}`, value), query);
      } catch (error) {
        if (signal?.aborted) return;
        const res = errorResult(`retrieve ${name}`, error);
        // Nobody asked for this read, and the connection status already reports
        // an unreachable Core. The result still carries the failure.
        if (addStatusOnFailure && !isConnectionError(error)) addStatus(res.status);
        onChange(res, query);
      }
    },
    [client, name, beforeRetrieve, addStatusOnFailure, onChange],
  );
  const retrieveSync = useCallback(
    (query: state.SetArg<Query, Partial<Query>>, options?: query.FetchOptions) =>
      void retrieveAsync(query, options),
    [retrieveAsync],
  );
  return {
    retrieve: retrieveSync,
    retrieveAsync,
  };
};

const useDirect = <Query extends query.Params, Data extends query.Data>({
  query,
  ...restParams
}: UseDirectRetrieveParams<Query, Data> &
  CreateRetrieveParams<Query, Data>): UseDirectRetrieveReturn<Data> => {
  const { retrieveAsync, retrieve: _, ...rest } = useStateful(restParams);
  const memoquery = useMemoDeepEqual(query);
  useAsyncEffect(
    async (signal) => await retrieveAsync(memoquery, { signal }),
    [retrieveAsync, memoquery],
  );
  return rest;
};

const useEffect = <Query extends query.Params, Data extends query.Data>({
  query,
  onChange,
  ...restParams
}: UseRetrieveEffectParams<Query, Data> & CreateRetrieveParams<Query, Data>): void => {
  const resultRef = useRef<Result<Data>>(initialResult<Data>(restParams.name));
  const { retrieveAsync } = useObservableBase<Query, Data>({
    ...restParams,
    onChange: useCallback(
      (setter, query: Query) => {
        resultRef.current = state.executeSetter(setter, resultRef.current);
        onChange?.(resultRef.current, query);
      },
      [onChange],
    ),
  });
  const memoQuery = useMemoDeepEqual(query);
  useAsyncEffect(
    async (signal) => {
      if (memoQuery != null) await retrieveAsync(memoQuery, { signal });
    },
    [retrieveAsync, memoQuery],
  );
};

export const useObservableRetrieve = <
  Query extends query.Params,
  Data extends query.Data,
>({
  onChange,
  ...restParams
}: UseRetrieveObservableParams<Query, Data> &
  CreateRetrieveParams<Query, Data>): UseRetrieveObservableReturn<Query> => {
  const resultRef = useRef<Result<Data>>(initialResult<Data>(restParams.name));
  const handleChange = useCallback(
    (setter: state.SetArg<Result<Data>>, query: Query) => {
      resultRef.current = state.executeSetter(setter, resultRef.current);
      onChange?.(resultRef.current, query);
    },
    [onChange],
  );
  return useObservableBase<Query, Data>({
    ...restParams,
    onChange: handleChange,
  });
};

/**
 * Fetch dedup and settled answers for queries the domain client does not
 * cache. Entries persist until the next fetch of the same query replaces
 * them; domain-cached queries never populate `settled`. Scoped per client so
 * a settled error never outlives the client whose fetch produced it, and
 * settled errors are dropped when the connection epoch advances.
 */
interface LocalCache<Data> {
  epoch: number;
  inFlight: Map<string, Promise<Data>>;
  settled: Map<string, { data: Data } | { error: Error }>;
}

interface UseSuspendedParams<Query extends query.Params, Data extends query.Data> {
  query: Query;
  locals: WeakMap<Client, LocalCache<Data>>;
}

const localFor = <Data extends query.Data>(
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

const NOOP_SUBSCRIBE = () => () => {};

/**
 * How long a not-found suspending read stays pending before the not-found
 * becomes final. Covers a reference arriving ahead of its document's create
 * broadcast (e.g. a panel tab minted in another window).
 */
const NOT_FOUND_WAIT = TimeSpan.seconds(5);

const awaitCreation = <Query extends query.Params, Data extends query.Data>(
  params: RetrieveParams<Query>,
  {
    name,
    error,
    hash,
    subscribe,
    getCached,
    local,
  }: Required<Pick<CreateRetrieveParams<Query, Data>, "subscribe" | "getCached">> & {
    name: string;
    error: Error;
    hash: string;
    local: LocalCache<Data>;
  },
): Promise<Data> =>
  new Promise<Data>((resolve, reject) => {
    const epoch = local.epoch;
    let settled = false;
    let disconnect: destructor.Destructor = () => {};
    const finish = () => {
      settled = true;
      clearTimeout(timer);
      disconnect();
      local.inFlight.delete(hash);
    };
    const timer = setTimeout(() => {
      finish();
      // A reconnect landed during the wait, so the not-found may no longer hold.
      if (local.epoch === epoch) local.settled.set(hash, { error });
      reject(error);
    }, NOT_FOUND_WAIT.milliseconds);
    disconnect = subscribe(params, (result) => {
      if (result === undefined) return;
      finish();
      if (Deleted.matches<Data>(result))
        reject(new DeletedError(`${name} was deleted`, result.corpse));
      else resolve(result);
    });
    // An already-answered query delivers during subscribe itself, before the
    // destructor exists to be called; tear it down now.
    if (settled) disconnect();
    // The document may have landed, or been deleted, between the failed
    // fetch and the subscription mounting.
    const cached = getCached(params);
    if (cached !== undefined) {
      finish();
      if (Deleted.matches<Data>(cached))
        reject(new DeletedError(`${name} was deleted`, cached.corpse));
      else resolve(cached);
    }
  });

const suspendOnFetch = <Query extends query.Params, Data extends query.Data>(
  params: RetrieveParams<Query>,
  {
    name,
    retrieve,
    subscribe,
    getCached,
    local,
  }: Pick<
    CreateRetrieveParams<Query, Data>,
    "name" | "retrieve" | "subscribe" | "getCached"
  > & {
    local: LocalCache<Data>;
  },
): Data => {
  const hash = query.hash(params.query);
  const settled = local.settled.get(hash);
  if (settled != null) {
    if ("error" in settled) throw settled.error;
    return settled.data;
  }
  let promise = local.inFlight.get(hash);
  if (promise == null) {
    const epoch = local.epoch;
    promise = retrieve(params).then(
      (data) => {
        // Domain-cached queries are served by getCached on the next render;
        // everything else keeps its settled answer locally.
        if (getCached == null) local.settled.set(hash, { data });
        local.inFlight.delete(hash);
        return data;
      },
      (cause: unknown) => {
        const error = new Error(`Failed to retrieve ${name}`, { cause });
        // A domain-cached not-found stays pending: the reference may have
        // outrun its document's create broadcast, which the subscription will
        // deliver. Everything else settles.
        if (subscribe != null && getCached != null && NotFoundError.matches(cause))
          return awaitCreation(params, {
            name,
            error,
            hash,
            subscribe,
            getCached,
            local,
          });
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
  return use(promise);
};

const useSuspended = <Query extends query.Params, Data extends query.Data>({
  query,
  locals,
  name,
  retrieve,
  subscribe,
  getCached,
  deriveCached,
  equal,
}: UseSuspendedParams<Query, Data> & CreateRetrieveParams<Query, Data>): Data => {
  const memoQuery = useMemoDeepEqual(query);
  const client = Synnax.use();
  const held = useRef<{ query: Query; value: Data } | null>(null);

  // Every hook runs before the disconnected throw. Gating them on the client
  // would change this hook's count as the connection comes and goes, which
  // corrupts the caller's hook order.
  const cached = useSyncExternalStore(
    useCallback(
      (notify) => {
        if (subscribe == null || client == null) return NOOP_SUBSCRIBE();
        return subscribe({ client, query: memoQuery }, () => notify());
      },
      [client, memoQuery],
    ),
    useCallback(() => {
      if (client == null) return undefined;
      const next = getCached?.({ client, query: memoQuery });
      if (equal == null || !isLive<Data>(next)) return next;
      const prev = held.current;
      if (prev != null && prev.query === memoQuery && equal(prev.value, next))
        return prev.value;
      held.current = { query: memoQuery, value: next };
      return next;
    }, [client, memoQuery]),
  );

  if (client == null)
    throw new DisconnectedError(`Cannot retrieve ${name}: no Core connected.`);

  const local = localFor(locals, client);
  const params = { client, query: memoQuery };

  if (cached !== undefined) {
    if (Deleted.matches<Data>(cached))
      throw new DeletedError(`${name} was deleted`, cached.corpse);
    return cached;
  }
  const derived = deriveCached?.(params);
  if (derived != null) return derived;
  return suspendOnFetch(params, { name, retrieve, subscribe, getCached, local });
};

const useEnsure = <Query extends query.Params, Data extends query.Data>({
  query,
  locals,
  name,
  retrieve,
  subscribe,
  getCached,
  deriveCached,
}: UseSuspendedParams<Query, Data> & CreateRetrieveParams<Query, Data>): void => {
  const memoQuery = useMemoDeepEqual(query);
  const client = Synnax.use();

  if (client == null)
    throw new DisconnectedError(`Cannot retrieve ${name}: no Core connected.`);

  const local = localFor(locals, client);
  const params = { client, query: memoQuery };

  const cached = getCached?.(params);
  if (cached !== undefined) {
    if (Deleted.matches<Data>(cached))
      throw new DeletedError(`${name} was deleted`, cached.corpse);
    return;
  }
  if (deriveCached?.(params) != null) return;
  suspendOnFetch(params, { name, retrieve, subscribe, getCached, local });
};

const useInvalidate = <Query extends query.Params, Data extends query.Data>(
  locals: WeakMap<Client, LocalCache<Data>>,
): ((q: Query) => void) => {
  const client = Synnax.use();
  return useCallback(
    (q: Query) => {
      if (client == null) return;
      localFor(locals, client).settled.delete(query.hash(q));
    },
    [client, locals],
  );
};

export const createRetrieve = <Query extends query.Params, Data extends query.Data>(
  createParams: CreateRetrieveParams<Query, Data>,
): CreateRetrieveReturn<Query, Data> => {
  const locals = new WeakMap<Client, LocalCache<Data>>();
  return {
    useRetrieve: (
      query: Query,
      opts?: Omit<UseDirectRetrieveParams<Query, Data>, "query">,
    ) => useDirect({ ...createParams, query, ...opts }),
    useRetrieveStateful: () => useStateful(createParams),
    useRetrieveEffect: (params: UseRetrieveEffectParams<Query, Data>) =>
      useEffect({ ...createParams, ...params }),
    useRetrieveObservable: (params: UseRetrieveObservableParams<Query, Data>) =>
      useObservableRetrieve({ ...params, ...createParams }),
    useRetrieveSuspended: (query: Query) =>
      useSuspended({ ...createParams, query, locals }),
    useEnsureRetrieved: (query: Query) => useEnsure({ ...createParams, query, locals }),
    useInvalidate: () => useInvalidate<Query, Data>(locals),
  };
};
