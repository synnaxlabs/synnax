// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type cache, type Synnax as Client } from "@synnaxlabs/client";
import { type destructor } from "@synnaxlabs/x";
import { use, useCallback, useRef, useState, useSyncExternalStore } from "react";

import { base } from "@/flux/base";
import { DeletedError, DisconnectedError } from "@/flux/errors";
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
import { state } from "@/state";
import { useAdder } from "@/status/base/Aggregator";
import { Synnax } from "@/synnax";

export interface RetrieveParams<
  Query extends base.Query,
  AllowDisconnected extends boolean = false,
> {
  client: AllowDisconnected extends true ? Client | null : Client;
  query: Query;
}

/**
 * Binds a query definition onto its domain client's read surface: fetch =
 * `retrieve`, live updates = `subscribe` (the client's onChange), snapshot =
 * `getCached`. Flux holds no cache of its own; queries without `subscribe`
 * and `getCached` fetch on every mount and never receive live updates.
 */
export interface CreateRetrieveParams<
  Query extends base.Query,
  Data extends base.Data,
  AllowDisconnected extends boolean = false,
> {
  name: string;
  retrieve: (params: RetrieveParams<Query, AllowDisconnected>) => Promise<Data>;
  subscribe?: (
    params: RetrieveParams<Query, AllowDisconnected>,
    handler: cache.ChangeHandler<Data>,
  ) => destructor.Destructor;
  getCached?: (
    params: RetrieveParams<Query, AllowDisconnected>,
  ) => cache.Cached<Data> | undefined;
  allowDisconnected?: AllowDisconnected;
}

export interface BeforeRetrieveParams<Query extends base.Query> {
  query: Query;
}

export interface UseObservableBaseRetrieveParams<
  Query extends base.Query,
  Data extends state.State,
> {
  addStatusOnFailure?: boolean;
  beforeRetrieve?: (Params: BeforeRetrieveParams<Query>) => Data | boolean;
  onChange: (result: state.SetArg<Result<Data>>, query: Query) => void;
}

export interface UseRetrieveObservableParams<
  Query extends base.Query,
  Data extends state.State,
> extends Omit<UseObservableBaseRetrieveParams<Query, Data>, "onChange"> {
  onChange: (result: Result<Data>, query: Query) => void;
}

export interface UseRetrieveObservableReturn<Query extends base.Query> {
  retrieve: (
    query: state.SetArg<Query, Partial<Query>>,
    options?: base.FetchOptions,
  ) => void;
  retrieveAsync: (
    query: state.SetArg<Query, Partial<Query>>,
    options?: base.FetchOptions,
  ) => Promise<void>;
}

export type UseRetrieveStatefulReturn<
  Query extends base.Query,
  Data extends state.State,
> = Result<Data> & UseRetrieveObservableReturn<Query>;

export interface UseDirectRetrieveParams<
  Query extends base.Query,
  Data extends state.State,
> extends Pick<
  UseObservableBaseRetrieveParams<Query, Data>,
  "beforeRetrieve" | "addStatusOnFailure"
> {
  query: Query;
}

export type UseDirectRetrieveReturn<Data extends state.State> = Result<Data>;

export interface UseRetrieveEffectParams<
  Query extends base.Query,
  Data extends state.State,
> extends Pick<
  UseObservableBaseRetrieveParams<Query, Data>,
  "beforeRetrieve" | "addStatusOnFailure"
> {
  onChange?: (result: Result<Data>, query: Query) => void;
  query?: Query;
}

export interface UseRetrieve<Query extends base.Query, Data extends state.State> {
  (
    params: Query,
    opts?: Omit<UseDirectRetrieveParams<Query, Data>, "query">,
  ): UseDirectRetrieveReturn<Data>;
}

export interface UseRetrieveEffect<Query extends base.Query, Data extends state.State> {
  (params: UseRetrieveEffectParams<Query, Data>): void;
}

export interface UseRetrieveStateful<
  Query extends base.Query,
  Data extends state.State,
> {
  (): UseRetrieveStatefulReturn<Query, Data>;
}

export interface UseRetrieveObservable<
  Query extends base.Query,
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
  Query extends base.Query,
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
export interface UseEnsureRetrieved<Query extends base.Query> {
  (query: Query): void;
}

export interface CreateRetrieveReturn<
  Query extends base.Query,
  Data extends state.State,
> {
  useRetrieve: UseRetrieve<Query, Data>;
  useRetrieveEffect: UseRetrieveEffect<Query, Data>;
  useRetrieveStateful: UseRetrieveStateful<Query, Data>;
  useRetrieveObservable: UseRetrieveObservable<Query, Data>;
  useRetrieveSuspended: UseSuspendedRetrieve<Query, Data>;
  useEnsureRetrieved: UseEnsureRetrieved<Query>;
}

const initialResult = <Data extends state.State>(name: string): Result<Data> =>
  loadingResult<Data>(`Retrieving ${name}`, undefined);

const useStateful = <
  Query extends base.Query,
  Data extends base.Data,
  AllowDisconnected extends boolean = false,
>(
  createParams: CreateRetrieveParams<Query, Data, AllowDisconnected> &
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

const useObservableBase = <
  Query extends base.Query,
  Data extends base.Data,
  AllowDisconnected extends boolean = false,
>({
  retrieve,
  subscribe,
  name,
  onChange,
  beforeRetrieve,
  addStatusOnFailure = true,
  allowDisconnected = false as AllowDisconnected,
}: UseObservableBaseRetrieveParams<Query, Data> &
  CreateRetrieveParams<
    Query,
    Data,
    AllowDisconnected
  >): UseRetrieveObservableReturn<Query> => {
  const client = Synnax.use();
  const queryRef = useRef<Query | null>(null);
  const listeners = useDestructors();
  const addStatus = useAdder();
  const handleCacheChange = useCallback(
    (result: cache.Cached<Data> | undefined) => {
      const query = queryRef.current;
      if (query == null || result == null) return;
      if (result.variant === "changed")
        onChange(successResult(`retrieved ${name}`, result.data), query);
      else
        onChange(
          errorResult(
            `retrieve ${name}`,
            new DeletedError(`${name} was deleted`, result.corpse),
          ),
          query,
        );
    },
    [onChange, name],
  );
  const retrieveAsync = useCallback(
    async (
      querySetter: state.SetArg<Query, Partial<Query>>,
      options: base.FetchOptions = {},
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
        if (client == null && !allowDisconnected)
          return onChange(nullClientResult<Data>(`retrieve ${name}`), query);
        onChange((p) => loadingResult(`retrieving ${name}`, p.data), query);
        if (signal?.aborted) return;
        const params = {
          client: client as AllowDisconnected extends true ? Client | null : Client,
          query,
        };
        listeners.cleanup();
        const value = await retrieve(params);
        if (signal?.aborted) return;
        // Subscribing after the fetch keeps mount-time reads fresh: an
        // unsubscribed retrieve always refetches, a subscribed one is served
        // from the cache.
        if (subscribe != null && client != null)
          listeners.set(subscribe(params, handleCacheChange));
        onChange(successResult<Data>(`retrieved ${name}`, value), query);
      } catch (error) {
        if (signal?.aborted) return;
        const res = errorResult(`retrieve ${name}`, error);
        if (addStatusOnFailure) addStatus(res.status);
        onChange(res, query);
      }
    },
    [client, name, beforeRetrieve, addStatusOnFailure, onChange],
  );
  const retrieveSync = useCallback(
    (query: state.SetArg<Query, Partial<Query>>, options?: base.FetchOptions) =>
      void retrieveAsync(query, options),
    [retrieveAsync],
  );
  return {
    retrieve: retrieveSync,
    retrieveAsync,
  };
};

const useDirect = <
  Query extends base.Query,
  Data extends base.Data,
  AllowDisconnected extends boolean = false,
>({
  query,
  ...restParams
}: UseDirectRetrieveParams<Query, Data> &
  CreateRetrieveParams<
    Query,
    Data,
    AllowDisconnected
  >): UseDirectRetrieveReturn<Data> => {
  const { retrieveAsync, retrieve: _, ...rest } = useStateful(restParams);
  const memoquery = useMemoDeepEqual(query);
  useAsyncEffect(
    async (signal) => await retrieveAsync(memoquery, { signal }),
    [retrieveAsync, memoquery],
  );
  return rest;
};

const useEffect = <
  Query extends base.Query,
  Data extends base.Data,
  AllowDisconnected extends boolean = false,
>({
  query,
  onChange,
  ...restParams
}: UseRetrieveEffectParams<Query, Data> &
  CreateRetrieveParams<Query, Data, AllowDisconnected>): void => {
  const resultRef = useRef<Result<Data>>(initialResult<Data>(restParams.name));
  const { retrieveAsync } = useObservableBase<Query, Data, AllowDisconnected>({
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
  Query extends base.Query,
  Data extends base.Data,
  AllowDisconnected extends boolean = false,
>({
  onChange,
  ...restParams
}: UseRetrieveObservableParams<Query, Data> &
  CreateRetrieveParams<
    Query,
    Data,
    AllowDisconnected
  >): UseRetrieveObservableReturn<Query> => {
  const resultRef = useRef<Result<Data>>(initialResult<Data>(restParams.name));
  const handleChange = useCallback(
    (setter: state.SetArg<Result<Data>>, query: Query) => {
      resultRef.current = state.executeSetter(setter, resultRef.current);
      onChange?.(resultRef.current, query);
    },
    [onChange],
  );
  return useObservableBase<Query, Data, AllowDisconnected>({
    ...restParams,
    onChange: handleChange,
  });
};

/**
 * Fetch dedup and settled answers for queries the domain client does not
 * cache. Entries persist until the next fetch of the same query replaces
 * them; domain-cached queries never populate `settled`.
 */
interface LocalCache<Data> {
  inFlight: Map<string, Promise<Data>>;
  settled: Map<string, { data: Data } | { error: Error }>;
}

interface UseSuspendedParams<Query extends base.Query, Data extends base.Data> {
  query: Query;
  local: LocalCache<Data>;
}

const NOOP_SUBSCRIBE = () => () => {};

const suspendOnFetch = <
  Query extends base.Query,
  Data extends base.Data,
  AllowDisconnected extends boolean = false,
>(
  params: RetrieveParams<Query, AllowDisconnected>,
  {
    name,
    retrieve,
    getCached,
    local,
  }: Pick<
    CreateRetrieveParams<Query, Data, AllowDisconnected>,
    "name" | "retrieve" | "getCached"
  > & { local: LocalCache<Data> },
): Data => {
  const hash = base.hashQuery(params.query);
  const settled = local.settled.get(hash);
  if (settled != null) {
    if ("error" in settled) throw settled.error;
    return settled.data;
  }
  let promise = local.inFlight.get(hash);
  if (promise == null) {
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
        if (getCached == null) local.settled.set(hash, { error });
        local.inFlight.delete(hash);
        throw error;
      },
    );
    local.inFlight.set(hash, promise);
  }
  return use(promise);
};

const useSuspended = <
  Query extends base.Query,
  Data extends base.Data,
  AllowDisconnected extends boolean = false,
>({
  query,
  local,
  name,
  retrieve,
  subscribe,
  getCached,
  allowDisconnected = false as AllowDisconnected,
}: UseSuspendedParams<Query, Data> &
  CreateRetrieveParams<Query, Data, AllowDisconnected>): Data => {
  const memoQuery = useMemoDeepEqual(query);
  const client = Synnax.use();

  if (client == null && !allowDisconnected)
    throw new DisconnectedError(
      `Cannot retrieve ${name}: no Synnax client connected. Pass allowDisconnected to opt out.`,
    );

  const params = {
    client: client as AllowDisconnected extends true ? Client | null : Client,
    query: memoQuery,
  };

  const cached = useSyncExternalStore(
    useCallback(
      (notify) => {
        if (subscribe == null || client == null) return NOOP_SUBSCRIBE();
        return subscribe(params, () => notify());
      },
      [client, memoQuery],
    ),
    useCallback(() => getCached?.(params), [client, memoQuery]),
  );

  if (cached?.variant === "changed") return cached.data;
  if (cached?.variant === "deleted")
    throw new DeletedError(`${name} was deleted`, cached.corpse);
  return suspendOnFetch(params, { name, retrieve, getCached, local });
};

const useEnsure = <
  Query extends base.Query,
  Data extends base.Data,
  AllowDisconnected extends boolean = false,
>({
  query,
  local,
  name,
  retrieve,
  getCached,
  allowDisconnected = false as AllowDisconnected,
}: UseSuspendedParams<Query, Data> &
  CreateRetrieveParams<Query, Data, AllowDisconnected>): void => {
  const memoQuery = useMemoDeepEqual(query);
  const client = Synnax.use();

  if (client == null && !allowDisconnected)
    throw new DisconnectedError(
      `Cannot retrieve ${name}: no Synnax client connected. Pass allowDisconnected to opt out.`,
    );

  const params = {
    client: client as AllowDisconnected extends true ? Client | null : Client,
    query: memoQuery,
  };

  const cached = getCached?.(params);
  if (cached?.variant === "changed") return;
  if (cached?.variant === "deleted")
    throw new DeletedError(`${name} was deleted`, cached.corpse);
  suspendOnFetch(params, { name, retrieve, getCached, local });
};

export const createRetrieve = <
  Query extends base.Query,
  Data extends base.Data,
  AllowDisconnected extends boolean = false,
>(
  createParams: CreateRetrieveParams<Query, Data, AllowDisconnected>,
): CreateRetrieveReturn<Query, Data> => {
  const local: LocalCache<Data> = { inFlight: new Map(), settled: new Map() };
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
      useSuspended({ ...createParams, query, local }),
    useEnsureRetrieved: (query: Query) => useEnsure({ ...createParams, query, local }),
  };
};
