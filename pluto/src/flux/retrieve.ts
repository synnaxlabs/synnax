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

import { type flux } from "@/flux/aether";
import { type FetchOptions, type Params } from "@/flux/core/params";
import { useStore } from "@/flux/Provider";
import {
  errorResult,
  nullClientResult,
  pendingResult,
  type Result,
  successResult,
} from "@/flux/result";
import { useAsyncEffect } from "@/hooks";
import { useDestructors } from "@/hooks/useDestructors";
import { useMemoDeepEqual } from "@/memo";
import { state } from "@/state";
import { Synnax } from "@/synnax";

/**
 * Arguments passed to the `retrieve` function when executing a query.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 */
export interface RetrieveArgs<RetrieveParams, ScopedStore extends flux.Store> {
  /** The Synnax client instance for making requests */
  client: Client;
  /** The parameters for the retrieve operation */
  params: RetrieveParams;
  /** The store instance for storing data */
  store: ScopedStore;
}

export interface MountListenersArgs<
  ScopedStore extends flux.Store,
  RetrieveParams,
  Data extends state.State,
> {
  store: ScopedStore;
  client: Client;
  params: RetrieveParams;
  onChange: state.Setter<Data>;
}

/**
 * Configuration arguments for creating a retrieve query.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template Data The type of data being retrieved
 */
export interface CreateRetrieveArgs<
  RetrieveParams,
  Data extends state.State,
  ScopedStore extends flux.Store,
> {
  /**
   * The name of the resource being retrieved. This is used to make pretty messages for
   * the various query states. This name should be in a human readable format and
   * capitalized as a proper noun.
   */
  name: string;
  /** Function executed when the query is evaluated or the query parameters change. */
  retrieve: (args: RetrieveArgs<RetrieveParams, ScopedStore>) => Promise<Data>;
  /**
   * Listeners to mount to the query. These listeners will be re-mounted when
   * the query parameters change and/or the client disconnects/re-connects or clusters
   * are switched.
   *
   * These listeners will NOT be remounted when the identity of the onChange function
   * changes, as the onChange function should be static.
   */
  mountListeners?: (
    args: MountListenersArgs<ScopedStore, RetrieveParams, Data>,
  ) => Destructor | Destructor[];
}

/**
 * Arguments for the observable retrieve hook.
 *
 * @template V The type of data being retrieved
 */
export interface UseObservableRetrieveArgs<V extends state.State> {
  /** Callback function to handle state changes */
  onChange: state.Setter<Result<V>>;
  /** The scope to use for the retrieve operation */
  scope?: string;
}

/**
 * Return type for the observable retrieve hook.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 */
export interface UseObservableRetrieveReturn<RetrieveParams extends Params> {
  /** Function to trigger a retrieve operation (fire-and-forget) */
  retrieve: (
    params: state.SetArg<RetrieveParams, Partial<RetrieveParams>>,
    options?: FetchOptions,
  ) => void;
  /** Function to trigger a retrieve operation and await the result */
  retrieveAsync: (
    params: state.SetArg<RetrieveParams, Partial<RetrieveParams>>,
    options?: FetchOptions,
  ) => Promise<void>;
}

/**
 * Return type for the stateful retrieve hook, combining result state with retrieve functions.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template Data The type of data being retrieved
 */
export type UseStatefulRetrieveReturn<
  RetrieveParams extends Params,
  Data extends state.State,
> = Result<Data> & UseObservableRetrieveReturn<RetrieveParams>;

/**
 * Arguments for the direct retrieve hook.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 */
export interface UseDirectRetrieveArgs<RetrieveParams extends Params> {
  /** Parameters for the retrieve operation */
  params: RetrieveParams;
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
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template Data The type of data being retrieved
 */
export interface useRetrieveEffectArgs<
  RetrieveParams extends Params,
  Data extends state.State,
> {
  scope?: string;
  onChange?: (result: Result<Data>) => void;
  /** Parameters for the retrieve operation */
  params?: RetrieveParams;
}

/**
 * Return type for the createRetrieve function.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template Data The type of data being retrieved
 */
export interface CreateRetrieveReturn<
  RetrieveParams extends Params,
  Data extends state.State,
> {
  /**
   * Hook that automatically fetches data when parameters change and returns the result state.
   * Use this for most cases where you want React to handle the data fetching lifecycle automatically.
   * Data is fetched when the component mounts and re-fetched whenever params change.
   */
  useRetrieve: (args: RetrieveParams) => UseDirectRetrieveReturn<Data>;

  /**
   * Hook that triggers data fetching as a side effect when parameters change but returns nothing.
   * Use this when you need to trigger data fetching but handle the result state externally
   * (e.g., through the onChange callback). Returns void - no state is managed internally.
   */
  useRetrieveEffect: (args: useRetrieveEffectArgs<RetrieveParams, Data>) => void;

  /**
   * Hook that provides manual control over when data is fetched, with internal state management.
   * Use this when you need to trigger data fetching based on user actions or specific events.
   * Returns both the current state (data, variant, error) and functions to manually trigger retrieval.
   */
  useRetrieveStateful: () => UseStatefulRetrieveReturn<RetrieveParams, Data>;
}

const initialResult = <Data extends state.State>(name: string): Result<Data> =>
  pendingResult<Data>(name, "retrieving", undefined);

const useStateful = <
  RetrieveParams extends Params,
  Data extends state.State,
  ScopedStore extends flux.Store,
>(
  args: CreateRetrieveArgs<RetrieveParams, Data, ScopedStore>,
): UseStatefulRetrieveReturn<RetrieveParams, Data> => {
  const [state, setState] = useState<Result<Data>>(initialResult<Data>(args.name));
  return {
    ...state,
    ...useObservable({ ...args, onChange: setState }),
  };
};

const useObservable = <
  RetrieveParams extends Params,
  Data extends state.State,
  ScopedStore extends flux.Store,
>({
  retrieve,
  mountListeners,
  name,
  onChange,
  scope,
}: UseObservableRetrieveArgs<Data> &
  CreateRetrieveArgs<
    RetrieveParams,
    Data,
    ScopedStore
  >): UseObservableRetrieveReturn<RetrieveParams> => {
  const client = Synnax.use();
  const paramsRef = useRef<RetrieveParams | null>(null);
  const store = useStore<ScopedStore>(scope);
  const listeners = useDestructors();
  const handleListenerChange = useCallback(
    (value: state.SetArg<Data>) =>
      onChange((prev) => {
        if (prev.data === undefined) return prev;
        const next = state.executeSetter(value, prev.data);
        return successResult(name, "retrieved", next);
      }),
    [onChange, name],
  );
  const retrieveAsync = useCallback(
    async (
      paramsSetter: state.SetArg<RetrieveParams, Partial<RetrieveParams>>,
      options: FetchOptions = {},
    ) => {
      const { signal } = options;
      const params = state.executeSetter<RetrieveParams, Partial<RetrieveParams>>(
        paramsSetter,
        paramsRef.current ?? {},
      );
      paramsRef.current = params;
      try {
        if (client == null) return onChange(nullClientResult<Data>(name, "retrieve"));
        onChange((p) => pendingResult(name, "retrieving", p.data));
        if (signal?.aborted) return;
        const args = { client, params, store };
        const value = await retrieve(args);
        if (signal?.aborted) return;
        listeners.cleanup();
        listeners.set(mountListeners?.({ ...args, onChange: handleListenerChange }));
        onChange(successResult<Data>(name, "retrieved", value));
      } catch (error) {
        if (signal?.aborted) return;
        onChange(errorResult<Data>(name, "retrieve", error));
      }
    },
    [client, name],
  );
  const retrieveSync = useCallback(
    (
      params: state.SetArg<RetrieveParams, Partial<RetrieveParams>>,
      options?: FetchOptions,
    ) => void retrieveAsync(params, options),
    [retrieveAsync],
  );
  return {
    retrieve: retrieveSync,
    retrieveAsync,
  };
};

const useDirect = <
  RetrieveParams extends Params,
  Data extends state.State,
  ScopedStore extends flux.Store,
>({
  params,
  ...restArgs
}: UseDirectRetrieveArgs<RetrieveParams> &
  CreateRetrieveArgs<
    RetrieveParams,
    Data,
    ScopedStore
  >): UseDirectRetrieveReturn<Data> => {
  const { retrieveAsync, retrieve: _, ...rest } = useStateful(restArgs);
  const memoParams = useMemoDeepEqual(params);
  useAsyncEffect(
    async (signal) => await retrieveAsync(memoParams, { signal }),
    [retrieveAsync, memoParams],
  );
  return rest;
};

const useEffect = <
  RetrieveParams extends Params,
  Data extends state.State,
  ScopedStore extends flux.Store,
>({
  params,
  ...restArgs
}: useRetrieveEffectArgs<RetrieveParams, Data> &
  CreateRetrieveArgs<RetrieveParams, Data, ScopedStore>): void => {
  const resultRef = useRef<Result<Data>>(initialResult<Data>(restArgs.name));
  const { retrieveAsync } = useObservable<RetrieveParams, Data, ScopedStore>({
    ...restArgs,
    onChange: (setter) => {
      resultRef.current = state.executeSetter(setter, resultRef.current);
      restArgs.onChange?.(resultRef.current);
    },
  });
  const memoParams = useMemoDeepEqual(params);
  useAsyncEffect(
    async (signal) => {
      if (memoParams == null) return;
      await retrieveAsync(memoParams, { signal });
    },
    [retrieveAsync, memoParams],
  );
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
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template Data The type of data being retrieved
 * @param factoryArgs Configuration object containing the retrieve function and resource name
 * @returns Object containing hooks for different retrieve patterns
 *
 * @example
 * ```typescript
 * interface UserRetrieveParams extends Params {
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
 * const userRetrieve = createRetrieve<UserRetrieveParams, User>({
 *   name: "User",
 *   retrieve: async ({ params, client }) => {
 *     return await client.users.get(params.userId, {
 *       includeProfile: params.includeProfile
 *     });
 *   },
 *   listeners: [
 *     {
 *       channel: "user_updates",
 *       onChange: ({ changed, params, onChange }) => {
 *         const updatedUser = changed.get("user_updates");
 *         if (updatedUser.id === params.userId) {
 *           onChange(updatedUser);
 *         }
 *       }
 *     }
 *   ]
 * });
 *
 * // Automatic fetching - data loads when component mounts and when userId changes
 * const { data, variant, error } = userRetrieve.useDirect({
 *   params: { userId: 123, includeProfile: true }
 * });
 *
 * // Side effect only - trigger fetching but handle result elsewhere
 * userRetrieve.useEffect({
 *   params: { userId: 123 },
 *   onChange: (result) => analyticsService.track('user_loaded', result)
 * });
 * ```
 */
export const createRetrieve = <
  RetrieveParams extends Params,
  Data extends state.State,
  ScopedStore extends flux.Store = {},
>(
  factoryArgs: CreateRetrieveArgs<RetrieveParams, Data, ScopedStore>,
): CreateRetrieveReturn<RetrieveParams, Data> => ({
  useRetrieve: (args: RetrieveParams) => useDirect({ ...factoryArgs, params: args }),
  useRetrieveStateful: () => useStateful(factoryArgs),
  useRetrieveEffect: (args: useRetrieveEffectArgs<RetrieveParams, Data>) =>
    useEffect({ ...factoryArgs, ...args }),
});
