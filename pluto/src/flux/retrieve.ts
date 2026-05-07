// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type destructor } from "@synnaxlabs/x";
import { use, useCallback, useRef, useState, useSyncExternalStore } from "react";

import { type base } from "@/flux/base";
import { useQueryCache, useStore } from "@/flux/Provider";
import { hashQuery, type QueryCache } from "@/flux/queryCache";
import {
  errorResult,
  loadingResult,
  nullClientResult,
  pendingResult,
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
  Query extends base.Shape,
  Store extends base.Store,
  AllowDisconnected extends boolean = false,
> {
  client: AllowDisconnected extends true ? Client | null : Client;
  query: Query;
  store: Store;
  /// Per-client query cache. Suspended retrieves (`createSuspendedRetrieve`)
  /// share entries through this cache so concurrent reads of the same query
  /// dedupe to a single fetch. Optional because legacy `createRetrieve`
  /// callers don't read it; suspended retrieves always populate it.
  cache?: QueryCache;
}

export interface RetrieveMountListenersParams<
  Query extends base.Shape,
  Data extends base.Shape,
  Store extends base.Store,
  AllowDisconnected extends boolean = false,
> extends RetrieveParams<Query, Store, AllowDisconnected> {
  onChange: state.Setter<Data | undefined>;
}

export interface CreateRetrieveParams<
  Query extends base.Shape,
  Data extends base.Shape,
  Store extends base.Store,
  AllowDisconnected extends boolean = false,
> {
  name: string;
  retrieve: (Params: RetrieveParams<Query, Store, AllowDisconnected>) => Promise<Data>;
  mountListeners?: (
    Params: RetrieveMountListenersParams<Query, Data, Store, AllowDisconnected>,
  ) => destructor.Destructor | destructor.Destructor[];
  allowDisconnected?: AllowDisconnected;
}

export interface BeforeRetrieveParams<Query extends base.Shape> {
  query: Query;
}

export interface UseObservableBaseRetrieveParams<
  Query extends base.Shape,
  Data extends state.State,
> {
  addStatusOnFailure?: boolean;
  beforeRetrieve?: (Params: BeforeRetrieveParams<Query>) => Data | boolean;
  onChange: (result: state.SetArg<Result<Data>>, query: Query) => void;
  scope?: string;
}

export interface UseRetrieveObservableParams<
  Query extends base.Shape,
  Data extends state.State,
> extends Omit<UseObservableBaseRetrieveParams<Query, Data>, "onChange"> {
  onChange: (result: Result<Data>, query: Query) => void;
}

export interface UseRetrieveObservableReturn<Query extends base.Shape> {
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
  Query extends base.Shape,
  Data extends state.State,
> = Result<Data> & UseRetrieveObservableReturn<Query>;

export interface UseDirectRetrieveParams<
  Query extends base.Shape,
  Data extends state.State,
> extends Pick<
  UseObservableBaseRetrieveParams<Query, Data>,
  "scope" | "beforeRetrieve" | "addStatusOnFailure"
> {
  query: Query;
}

export type UseDirectRetrieveReturn<Data extends state.State> = Result<Data>;

export interface UseRetrieveEffectParams<
  Query extends base.Shape,
  Data extends state.State,
> {
  scope?: string;
  onChange?: (result: Result<Data>, query: Query) => void;
  query?: Query;
}

export interface UseRetrieve<Query extends base.Shape, Data extends state.State> {
  (
    params: Query,
    opts?: Omit<UseDirectRetrieveParams<Query, Data>, "query">,
  ): UseDirectRetrieveReturn<Data>;
}

export interface UseRetrieveEffect<Query extends base.Shape, Data extends state.State> {
  (params: UseRetrieveEffectParams<Query, Data>): void;
}

export interface UseRetrieveStateful<
  Query extends base.Shape,
  Data extends state.State,
> {
  (): UseRetrieveStatefulReturn<Query, Data>;
}

export interface UseRetrieveObservable<
  Query extends base.Shape,
  Data extends state.State,
> {
  (
    params: UseRetrieveObservableParams<Query, Data>,
  ): UseRetrieveObservableReturn<Query>;
}

export interface CreateRetrieveReturn<
  Query extends base.Shape,
  Data extends state.State,
> {
  useRetrieve: UseRetrieve<Query, Data>;
  useRetrieveEffect: UseRetrieveEffect<Query, Data>;
  useRetrieveStateful: UseRetrieveStateful<Query, Data>;
  useRetrieveObservable: UseRetrieveObservable<Query, Data>;
}

const initialResult = <Data extends state.State>(name: string): Result<Data> =>
  loadingResult<Data>(`Retrieving ${name}`, undefined);

const useStateful = <
  Query extends base.Shape,
  Data extends state.State,
  ScopedStore extends base.Store,
  AllowDisconnected extends boolean = false,
>(
  createParams: CreateRetrieveParams<Query, Data, ScopedStore, AllowDisconnected>,
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
  Query extends base.Shape,
  Data extends state.State,
  ScopedStore extends base.Store,
  AllowDisconnected extends boolean = false,
>({
  retrieve,
  mountListeners,
  name,
  onChange,
  scope,
  beforeRetrieve,
  addStatusOnFailure = true,
  allowDisconnected = false as AllowDisconnected,
}: UseObservableBaseRetrieveParams<Query, Data> &
  CreateRetrieveParams<
    Query,
    Data,
    ScopedStore,
    AllowDisconnected
  >): UseRetrieveObservableReturn<Query> => {
  const client = Synnax.use();
  const queryRef = useRef<Query | null>(null);
  const store = useStore<ScopedStore>(scope);
  const cache = useQueryCache();
  const listeners = useDestructors();
  const addStatus = useAdder();
  const handleListenerChange = useCallback(
    (value: state.SetArg<Data | undefined>) => {
      if (queryRef.current == null) return;
      onChange((prev) => {
        const next = state.executeSetter(value, prev.data);
        if (next == null) return prev;
        return successResult(`retrieved ${name}`, next);
      }, queryRef.current);
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
          store,
          cache,
        };
        listeners.cleanup();
        listeners.set(mountListeners?.({ ...params, onChange: handleListenerChange }));
        const value = await retrieve(params);
        if (signal?.aborted) return;
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
  Query extends base.Shape,
  Data extends state.State,
  ScopedStore extends base.Store,
  AllowDisconnected extends boolean = false,
>({
  query,
  ...restParams
}: UseDirectRetrieveParams<Query, Data> &
  CreateRetrieveParams<
    Query,
    Data,
    ScopedStore,
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
  Query extends base.Shape,
  Data extends state.State,
  ScopedStore extends base.Store,
  AllowDisconnected extends boolean = false,
>({
  query,
  onChange,
  ...restParams
}: UseRetrieveEffectParams<Query, Data> &
  CreateRetrieveParams<Query, Data, ScopedStore, AllowDisconnected>): void => {
  const resultRef = useRef<Result<Data>>(initialResult<Data>(restParams.name));
  const { retrieveAsync } = useObservableBase<
    Query,
    Data,
    ScopedStore,
    AllowDisconnected
  >({
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
  Query extends base.Shape,
  Data extends state.State,
  ScopedStore extends base.Store,
  AllowDisconnected extends boolean = false,
>({
  onChange,
  ...restParams
}: UseRetrieveObservableParams<Query, Data> &
  CreateRetrieveParams<
    Query,
    Data,
    ScopedStore,
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
  return useObservableBase<Query, Data, ScopedStore, AllowDisconnected>({
    ...restParams,
    onChange: handleChange,
  });
};

export const createRetrieve = <
  Query extends base.Shape,
  Data extends state.State,
  ScopedStore extends base.Store = {},
  AllowDisconnected extends boolean = false,
>(
  createParams: CreateRetrieveParams<Query, Data, ScopedStore, AllowDisconnected>,
): CreateRetrieveReturn<Query, Data> => ({
  useRetrieve: (
    query: Query,
    opts?: Omit<UseDirectRetrieveParams<Query, Data>, "query">,
  ) => useDirect({ ...createParams, query, ...opts }),
  useRetrieveStateful: () => useStateful(createParams),
  useRetrieveEffect: (Params: UseRetrieveEffectParams<Query, Data>) =>
    useEffect({ ...createParams, ...Params }),
  useRetrieveObservable: (params: UseRetrieveObservableParams<Query, Data>) =>
    useObservableRetrieve({ ...params, ...createParams }),
});

/// A Suspense-shaped retrieve hook that returns the data directly. The hook
/// either returns the resolved value or suspends on the in-flight promise.
/// Concurrent reads of the same query share one fetch via the per-client
/// query cache. Channel-listener-driven updates push new values into the
/// cache, which notifies subscribed consumers without re-suspending.
export interface UseSuspendedRetrieve<
  Query extends base.Shape,
  Data extends base.Shape,
> {
  (query: Query): Data;
}

/// A Suspense-shaped hook that ensures a query has been retrieved into the
/// cache (and any listener-driven side effects have been mounted) but does
/// NOT subscribe the calling component to subsequent cache updates. Use this
/// in components that only need to gate rendering on data being present, e.g.,
/// a parent that wants children to read fresh values via their own selectors
/// without itself re-rendering when the data changes.
export interface UseEnsureSuspendedRetrieve<Query extends base.Shape> {
  (query: Query): void;
}

export interface CreateSuspendedRetrieveReturn<
  Query extends base.Shape,
  Data extends base.Shape,
  Store extends base.Store = {},
  AllowDisconnected extends boolean = false,
> {
  useRetrieve: UseSuspendedRetrieve<Query, Data>;
  useEnsureRetrieved: UseEnsureSuspendedRetrieve<Query>;
  /// Spec the hooks were built from. Exposed so callers can compose against
  /// the underlying `name`, `retrieve`, and `mountListeners`.
  spec: CreateRetrieveParams<Query, Data, Store, AllowDisconnected>;
}

/// Builds Suspense-shaped retrieve hooks. Pair with `<Flux.Suspense>` above
/// the consumer to render a fallback while loading and a status panel on
/// error.
///
/// Returns two hooks:
/// - `useRetrieve(query)` — suspends until resolved, returns the data, and
///   re-renders the consumer on every cache update (listener-driven or
///   dispatch-driven).
/// - `useEnsureRetrieved(query)` — suspends until resolved like `useRetrieve` but does
///   not subscribe. Re-renders are not triggered when the cache entry
///   subsequently changes; downstream selectors still see fresh values
///   because the underlying store is updated independently.
export const createSuspendedRetrieve = <
  Query extends base.Shape,
  Data extends base.Shape,
  ScopedStore extends base.Store = {},
  AllowDisconnected extends boolean = false,
>(
  spec: CreateRetrieveParams<Query, Data, ScopedStore, AllowDisconnected>,
): CreateSuspendedRetrieveReturn<Query, Data, ScopedStore, AllowDisconnected> => {
  const { name, retrieve, mountListeners, allowDisconnected } = spec;

  const useRetrieve = (query: Query): Data => {
    const memoQuery = useMemoDeepEqual(query);
    const client = Synnax.use();
    const store = useStore<ScopedStore>();
    const cache = useQueryCache();
    const hash = hashQuery({ name, query: memoQuery });

    if (client == null && !allowDisconnected)
      throw new Error(
        `Cannot retrieve ${name}: no Synnax client connected. Pass allowDisconnected to opt out.`,
      );

    useSyncExternalStore(
      useCallback(
        (notify) => {
          const cacheSub = cache.subscribe(hash, notify);
          if (mountListeners == null || client == null) return cacheSub;
          const onChange = (value: state.SetArg<Data | undefined>) => {
            const current = cache.get<Data>(hash);
            const prev = current?.variant === "success" ? current.data : undefined;
            const next = state.executeSetter(value, prev);
            if (next == null) return;
            cache.set(hash, successResult(name, next));
          };
          const result = mountListeners({
            client: client as AllowDisconnected extends true ? Client | null : Client,
            store,
            cache,
            query: memoQuery,
            onChange,
          });
          const listeners = Array.isArray(result) ? result : [result];
          return () => {
            cacheSub();
            listeners.forEach((d) => d?.());
          };
        },
        [cache, hash, client, store, memoQuery],
      ),
      useCallback(() => cache.get(hash), [cache, hash]),
    );

    const entry = cache.get<Data>(hash);
    if (entry?.variant === "success") return entry.data;
    if (entry?.variant === "error")
      // Throwing the Status: Flux.Suspense normalizes thrown Statuses without
      // re-running fromException, preserving the resource-specific message.
      // eslint-disable-next-line @typescript-eslint/only-throw-error
      throw entry.status;
    if (entry?.variant === "loading" && entry.promise != null)
      return use(entry.promise);

    const promise = retrieve({
      client: client as AllowDisconnected extends true ? Client | null : Client,
      query: memoQuery,
      store,
      cache,
    });
    cache.set(hash, pendingResult(name, promise));
    return use(promise);
  };

  const useEnsureRetrieved = (query: Query): void => {
    const memoQuery = useMemoDeepEqual(query);
    const client = Synnax.use();
    const store = useStore<ScopedStore>();
    const cache = useQueryCache();
    const hash = hashQuery({ name, query: memoQuery });

    if (client == null && !allowDisconnected)
      throw new Error(
        `Cannot retrieve ${name}: no Synnax client connected. Pass allowDisconnected to opt out.`,
      );

    const entry = cache.get<Data>(hash);
    if (entry?.variant === "success") return;
    if (entry?.variant === "error")
      // eslint-disable-next-line @typescript-eslint/only-throw-error
      throw entry.status;
    if (entry?.variant === "loading" && entry.promise != null) {
      use(entry.promise);
      return;
    }

    const promise = retrieve({
      client: client as AllowDisconnected extends true ? Client | null : Client,
      query: memoQuery,
      store,
      cache,
    });
    cache.set(hash, pendingResult(name, promise));
    use(promise);
  };

  return { useRetrieve, useEnsureRetrieved, spec };
};
