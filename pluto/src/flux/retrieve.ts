// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type Destructor } from "@synnaxlabs/x";
import { useCallback, useRef, useState } from "react";

import { type core } from "@/flux/core";
import { useStore } from "@/flux/Provider";
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
import { useAdder } from "@/status/Aggregator";
import { Synnax } from "@/synnax";

/**
 * Arguments passed to the `retrieve` function when executing a query.
 *
 * @template Query The type of query parameters for the retrieve operation.
 * @template Store The signature of the flux store for accessing cached items.
 */
export interface RetrieveParams<Query extends core.Shape, Store extends core.Store> {
  /** The Synnax client instance for making requests */
  client: Client;
  /** The parameters for the retrieve operation */
  query: Query;
  /** The store instance for storing data */
  store: Store;
}

/**
 * Arguments passed to the `mountListeners` function.
 *
 * @template Query The type of query parameters for the retrieve operation.
 * @template Data The type of data being retrieved.
 * @template Store The signature of the flux store for accessing cached items.
 */
export interface RetrieveMountListenersParams<
  Query extends core.Shape,
  Data extends core.Shape,
  Store extends core.Store,
> extends RetrieveParams<Query, Store> {
  onChange: state.Setter<Data | undefined>;
}

/**
 * Configuration arguments for creating a retrieve query.
 *
 * @template Query The type of parameters for the retrieve operation
 * @template Data The type of data being retrieved
 */
export interface CreateRetrieveParams<
  Query extends core.Shape,
  Data extends core.Shape,
  Store extends core.Store,
> {
  /**
   * The name of the resource being retrieved. This is used to make pretty messages for
   * the various query states. This name should be in a human readable format and
   * capitalized as a proper noun.
   */
  name: string;
  /** Function executed when the query is evaluated or the query parameters change. */
  retrieve: (Params: RetrieveParams<Query, Store>) => Promise<Data>;
  /**
   * Listeners to mount to the query. These listeners will be re-mounted when
   * the query parameters change and/or the client disconnects/re-connects or clusters
   * are switched.
   *
   * These listeners will NOT be remounted when the identity of the onChange function
   * changes, as the onChange function should be static.
   */
  mountListeners?: (
    Params: RetrieveMountListenersParams<Query, Data, Store>,
  ) => Destructor | Destructor[];
}

export interface BeforeRetrieveParams<Query extends core.Shape> {
  query: Query;
}

/**
 * Arguments for the observable retrieve hook.
 *
 * @template V The type of data being retrieved
 */
export interface UseObservableBaseRetrieveParams<
  Query extends core.Shape,
  Data extends state.State,
> {
  beforeRetrieve?: (Params: BeforeRetrieveParams<Query>) => Data | boolean;
  /** Callback function to handle state changes */
  onChange: (result: state.SetArg<Result<Data>>, query: Query) => void;
  /** The scope to use for the retrieve operation */
  scope?: string;
}

/**
 * Arguments for the observable retrieve hook.
 *
 * @template Query The type of parameters for the retrieve operation
 */
export interface UseRetrieveObservableParams<
  Query extends core.Shape,
  Data extends state.State,
> extends Omit<UseObservableBaseRetrieveParams<Query, Data>, "onChange"> {
  /** Callback function to handle state changes */
  onChange: (result: Result<Data>, query: Query) => void;
}

/**
 * Return type for the observable retrieve hook.
 *
 * @template Query The type of parameters for the retrieve operation
 */
export interface UseRetrieveObservableReturn<Query extends core.Shape> {
  /** Function to trigger a retrieve operation (fire-and-forget) */
  retrieve: (
    query: state.SetArg<Query, Partial<Query>>,
    options?: core.FetchOptions,
  ) => void;
  /** Function to trigger a retrieve operation and await the result */
  retrieveAsync: (
    query: state.SetArg<Query, Partial<Query>>,
    options?: core.FetchOptions,
  ) => Promise<void>;
}

/**
 * Return type for the stateful retrieve hook, combining result state with retrieve functions.
 *
 * @template Query The type of query parameters for the retrieve operation
 * @template Data The type of data being retrieved
 */
export type UseRetrieveStatefulReturn<
  Query extends core.Shape,
  Data extends state.State,
> = Result<Data> & UseRetrieveObservableReturn<Query>;

/**
 * Arguments for the direct retrieve hook.
 *
 * @template Query The type of query parameters for the retrieve operation
 * @template Data The type of data being retrieved
 */
export interface UseDirectRetrieveParams<
  Query extends core.Shape,
  Data extends state.State,
> extends Pick<
    UseObservableBaseRetrieveParams<Query, Data>,
    "scope" | "beforeRetrieve"
  > {
  /** Parameters for the retrieve operation */
  query: Query;
}

/**
 * Return type for the direct retrieve hook.
 *
 * @template Data The type of data being retrieved
 */
export type UseDirectRetrieveReturn<Data extends state.State> = Result<Data>;

/**
 * Arguments for the effect retrieve hook.
 *
 * @template Query The type of query parameters for the retrieve operation
 * @template Data The type of data being retrieved
 */
export interface UseRetrieveEffectParams<
  Query extends core.Shape,
  Data extends state.State,
> {
  scope?: string;
  onChange?: (result: Result<Data>, query: Query) => void;
  /** Parameters for the retrieve operation */
  query?: Query;
}

export interface UseRetrieve<Query extends core.Shape, Data extends state.State> {
  (
    params: Query,
    opts?: Pick<UseDirectRetrieveParams<Query, Data>, "beforeRetrieve">,
  ): UseDirectRetrieveReturn<Data>;
}

export interface UseRetrieveEffect<Query extends core.Shape, Data extends state.State> {
  (params: UseRetrieveEffectParams<Query, Data>): void;
}

export interface UseRetrieveStateful<
  Query extends core.Shape,
  Data extends state.State,
> {
  (): UseRetrieveStatefulReturn<Query, Data>;
}

export interface UseRetrieveObservable<
  Query extends core.Shape,
  Data extends state.State,
> {
  (
    params: UseRetrieveObservableParams<Query, Data>,
  ): UseRetrieveObservableReturn<Query>;
}

/**
 * Return type for the createRetrieve function.
 *
 * @template Query The type of query parameters for the retrieve operation
 * @template Data The type of data being retrieved
 */
export interface CreateRetrieveReturn<
  Query extends core.Shape,
  Data extends state.State,
> {
  /**
   * Hook that automatically fetches data when parameters change and returns the result state.
   * Use this for most cases where you want React to handle the data fetching lifecycle automatically.
   * Data is fetched when the component mounts and re-fetched whenever query change.
   */
  useRetrieve: UseRetrieve<Query, Data>;

  /**
   * Hook that triggers data fetching as a side effect when parameters change but returns nothing.
   * Use this when you need to trigger data fetching but handle the result state externally
   * (e.g., through the onChange callback). Returns void - no state is managed internally.
   */
  useRetrieveEffect: UseRetrieveEffect<Query, Data>;

  /**
   * Hook that provides manual control over when data is fetched, with internal state management.
   * Use this when you need to trigger data fetching based on user actions or specific events.
   * Returns both the current state (data, variant, error) and functions to manually trigger retrieval.
   */
  useRetrieveStateful: UseRetrieveStateful<Query, Data>;

  useRetrieveObservable: UseRetrieveObservable<Query, Data>;
}

const initialResult = <Data extends state.State>(name: string): Result<Data> =>
  loadingResult<Data>(`Retrieving ${name}`, undefined);

const useStateful = <
  Query extends core.Shape,
  Data extends state.State,
  ScopedStore extends core.Store,
>(
  Params: CreateRetrieveParams<Query, Data, ScopedStore>,
): UseRetrieveStatefulReturn<Query, Data> => {
  const [state, setState] = useState<Result<Data>>(initialResult<Data>(Params.name));
  return {
    ...state,
    ...useObservableBase({ ...Params, onChange: setState }),
  };
};

const useObservableBase = <
  Query extends core.Shape,
  Data extends state.State,
  ScopedStore extends core.Store,
>({
  retrieve,
  mountListeners,
  name,
  onChange,
  scope,
  beforeRetrieve,
}: UseObservableBaseRetrieveParams<Query, Data> &
  CreateRetrieveParams<
    Query,
    Data,
    ScopedStore
  >): UseRetrieveObservableReturn<Query> => {
  const client = Synnax.use();
  const queryRef = useRef<Query | null>(null);
  const store = useStore<ScopedStore>(scope);
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
      options: core.FetchOptions = {},
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
        const Params = { client, query, store };
        const value = await retrieve(Params);
        if (signal?.aborted) return;
        listeners.cleanup();
        listeners.set(mountListeners?.({ ...Params, onChange: handleListenerChange }));
        onChange(successResult<Data>(`retrieved ${name}`, value), query);
      } catch (error) {
        if (signal?.aborted) return;
        const res = errorResult(`retrieve ${name}`, error);
        addStatus(res.status);
        onChange(res, query);
      }
    },
    [client, name, beforeRetrieve],
  );
  const retrieveSync = useCallback(
    (query: state.SetArg<Query, Partial<Query>>, options?: core.FetchOptions) =>
      void retrieveAsync(query, options),
    [retrieveAsync],
  );
  return {
    retrieve: retrieveSync,
    retrieveAsync,
  };
};

const useDirect = <
  Query extends core.Shape,
  Data extends state.State,
  ScopedStore extends core.Store,
>({
  query,
  ...restParams
}: UseDirectRetrieveParams<Query, Data> &
  CreateRetrieveParams<Query, Data, ScopedStore>): UseDirectRetrieveReturn<Data> => {
  const { retrieveAsync, retrieve: _, ...rest } = useStateful(restParams);
  const memoquery = useMemoDeepEqual(query);
  useAsyncEffect(
    async (signal) => await retrieveAsync(memoquery, { signal }),
    [retrieveAsync, memoquery],
  );
  return rest;
};

const useEffect = <
  Query extends core.Shape,
  Data extends state.State,
  ScopedStore extends core.Store,
>({
  query,
  onChange,
  ...restParams
}: UseRetrieveEffectParams<Query, Data> &
  CreateRetrieveParams<Query, Data, ScopedStore>): void => {
  const resultRef = useRef<Result<Data>>(initialResult<Data>(restParams.name));
  const { retrieveAsync } = useObservableBase<Query, Data, ScopedStore>({
    ...restParams,
    onChange: useCallback(
      (setter, query: Query) => {
        resultRef.current = state.executeSetter(setter, resultRef.current);
        onChange?.(resultRef.current, query);
      },
      [onChange],
    ),
  });
  const memoquery = useMemoDeepEqual(query);
  useAsyncEffect(
    async (signal) => {
      if (memoquery == null) return;
      await retrieveAsync(memoquery, { signal });
    },
    [retrieveAsync, memoquery],
  );
};

export const useObservableRetrieve = <
  Query extends core.Shape,
  Data extends state.State,
  ScopedStore extends core.Store,
>({
  onChange,
  ...restParams
}: UseRetrieveObservableParams<Query, Data> &
  CreateRetrieveParams<
    Query,
    Data,
    ScopedStore
  >): UseRetrieveObservableReturn<Query> => {
  const resultRef = useRef<Result<Data>>(initialResult<Data>(restParams.name));
  const handleChange = useCallback(
    (setter: state.SetArg<Result<Data>>, query: Query) => {
      resultRef.current = state.executeSetter(setter, resultRef.current);
      onChange?.(resultRef.current, query);
    },
    [onChange],
  );
  return useObservableBase<Query, Data, ScopedStore>({
    ...restParams,
    onChange: handleChange,
  });
};

/**
 * Creates a retrieve query system that provides hooks for fetching data with different control patterns.
 *
 * This function creates a set of React hooks that handle data retrieval with
 * proper loading states, error handling, caching, and real-time updates. It provides
 * two hook variants for different use cases:
 *
 * - `useDirect`: Automatically fetches data when parameters change. Best for most use cases.
 * - `useEffect`: Triggers data fetching as a side effect without returning state. Use when handling results externally.
 *
 * @template Query The type of query parameters for the retrieve operation
 * @template Data The type of data being retrieved
 * @param createParams Configuration object containing the retrieve function and resource name
 * @returns Object containing hooks for different retrieve patterns
 *
 * @example
 * ```typescript
 * interface UserQuery extends core.Shape {
 *   userId: number;
 *   includeProfile?: boolean;
 * }
 *
 * interface User {
 *   id: number;
 *   name: string;
 *   email: string;
 * }
 *
 * const userRetrieve = createRetrieve<UserQuery, User>({
 *   name: "User",
 *   retrieve: async ({ query, client }) => {
 *     return await client.users.get(query.userId, {
 *       includeProfile: query.includeProfile
 *     });
 *   },
 *   listeners: [
 *     {
 *       channel: "user_updates",
 *       onChange: ({ changed, query, onChange }) => {
 *         const updatedUser = changed.get("user_updates");
 *         if (updatedUser.id === query.userId) {
 *           onChange(updatedUser);
 *         }
 *       }
 *     }
 *   ]
 * });
 *
 * // Automatic fetching - data loads when component mounts and when userId changes
 * const { data, variant, error } = userRetrieve.useDirect({
 *   query: { userId: 123, includeProfile: true }
 * });
 *
 * // Side effect only - trigger fetching but handle result elsewhere
 * userRetrieve.useEffect({
 *   query: { userId: 123 },
 *   onChange: (result) => analyticsService.track('user_loaded', result)
 * });
 * ```
 */
export const createRetrieve = <
  Query extends core.Shape,
  Data extends state.State,
  ScopedStore extends core.Store = {},
>(
  createParams: CreateRetrieveParams<Query, Data, ScopedStore>,
): CreateRetrieveReturn<Query, Data> => ({
  useRetrieve: (
    query: Query,
    opts?: Pick<UseDirectRetrieveParams<Query, Data>, "beforeRetrieve">,
  ) => useDirect({ ...createParams, query, ...opts }),
  useRetrieveStateful: () => useStateful(createParams),
  useRetrieveEffect: (Params: UseRetrieveEffectParams<Query, Data>) =>
    useEffect({ ...createParams, ...Params }),
  useRetrieveObservable: (params: UseRetrieveObservableParams<Query, Data>) =>
    useObservableRetrieve({ ...params, ...createParams }),
});
