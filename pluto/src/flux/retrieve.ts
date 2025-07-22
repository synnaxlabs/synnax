// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type Synnax } from "@synnaxlabs/client";
import { type MultiSeries } from "@synnaxlabs/x";
import { useCallback, useRef, useState } from "react";

import { useMountSynchronizers } from "@/flux/listeners";
import { type AsyncOptions, type Params } from "@/flux/params";
import {
  errorResult,
  nullClientResult,
  pendingResult,
  type Result,
  successResult,
} from "@/flux/result";
import { type Sync } from "@/flux/sync";
import { useAsyncEffect } from "@/hooks";
import { useMemoDeepEqual } from "@/memo";
import { state } from "@/state";
import { Synnax as PSynnax } from "@/synnax";

/**
 * Extra arguments passed to retrieve listener handlers.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template Data The type of data being retrieved
 */
interface RetrieveListenerExtraArgs<
  RetrieveParams extends Params,
  Data extends state.State,
> {
  /** The current retrieve parameters */
  params: RetrieveParams;
  /** The Synnax client instance */
  client: Synnax;
  /** Function that updates the query data when a new value is received */
  onChange: state.Setter<Data>;
}

/**
 * Configuration for a listener that is called whenever a new value is received
 * from the specified channel. The listener is called with the new value and can
 * choose to update the state of the query by calling the `onChange` function.
 *
 * The listener will not be called if the query is in a loading or an error state.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template Data The type of data being retrieved
 */
export interface RetrieveListenerConfig<
  RetrieveParams extends Params,
  Data extends state.State,
> {
  /** The channel to listen to for real-time updates */
  channel: channel.Name;
  /** The function to call when a new value is received from the channel */
  onChange: Sync.ListenerHandler<
    MultiSeries,
    RetrieveListenerExtraArgs<RetrieveParams, Data>
  >;
}

/**
 * Arguments passed to the `retrieve` function when executing a query.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 */
export interface RetrieveArgs<RetrieveParams extends Params> {
  /** The Synnax client instance for making requests */
  client: Synnax;
  /** The parameters for the retrieve operation */
  params: RetrieveParams;
}

/**
 * Configuration arguments for creating a retrieve query.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template Data The type of data being retrieved
 */
export interface CreateRetrieveArgs<
  RetrieveParams extends Params,
  Data extends state.State,
> {
  /**
   * The name of the resource being retrieved. This is used to make pretty messages for
   * the various query states.
   */
  name: string;
  /** Function executed when the query is evaluated or the query parameters change. */
  retrieve: (args: RetrieveArgs<RetrieveParams>) => Promise<Data>;
  /**
   * Listeners to mount to the query. These listeners will be re-mounted when
   * the query parameters change and/or the client disconnects/re-connects or clusters
   * are switched.
   *
   * These listeners will NOT be remounted when the identity of the onChange function
   * changes, as the onChange function should be static.
   */
  listeners?: RetrieveListenerConfig<RetrieveParams, Data>[];
}

/**
 * Arguments for the observable retrieve hook.
 *
 * @template V The type of data being retrieved
 */
export interface UseObservableRetrieveArgs<V extends state.State> {
  /** Callback function to handle state changes */
  onChange: state.Setter<Result<V>>;
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
    options?: AsyncOptions,
  ) => void;
  /** Function to trigger a retrieve operation and await the result */
  retrieveAsync: (
    params: state.SetArg<RetrieveParams, Partial<RetrieveParams>>,
    options?: AsyncOptions,
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
export interface UseEffectRetrieveArgs<
  RetrieveParams extends Params,
  Data extends state.State,
> extends UseObservableRetrieveArgs<Data> {
  /** Parameters for the retrieve operation */
  params: RetrieveParams;
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
  /** Hook that provides retrieve functions with external state management */
  useDirect: (
    args: UseDirectRetrieveArgs<RetrieveParams>,
  ) => UseDirectRetrieveReturn<Data>;
  /** Hook that provides retrieve functions for external state management */
  useObservable: (
    args: UseObservableRetrieveArgs<Data>,
  ) => UseObservableRetrieveReturn<RetrieveParams>;
  /** Hook that provides retrieve functions with internal state management */
  useStateful: () => UseStatefulRetrieveReturn<RetrieveParams, Data>;
  /** Hook that automatically triggers retrieve operations based on parameter changes */
  useEffect: (args: UseEffectRetrieveArgs<RetrieveParams, Data>) => void;
}

const useObservable = <RetrieveParams extends Params, Data extends state.State>({
  retrieve,
  listeners = [],
  name,
  onChange,
}: UseObservableRetrieveArgs<Data> &
  CreateRetrieveArgs<
    RetrieveParams,
    Data
  >): UseObservableRetrieveReturn<RetrieveParams> => {
  const client = PSynnax.use();
  const paramsRef = useRef<RetrieveParams | null>(null);
  const mountListeners = useMountSynchronizers();
  const retrieveAsync = useCallback(
    async (
      paramsSetter: state.SetArg<RetrieveParams, Partial<RetrieveParams>>,
      options: AsyncOptions = {},
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
        const value = await retrieve({ client, params });
        if (signal?.aborted) return;
        mountListeners(
          listeners.map((l) => ({
            channel: l.channel,
            handler: (frame) =>
              void (async () => {
                if (client == null || paramsRef.current == null) return;
                try {
                  await l.onChange({
                    client,
                    params: paramsRef.current,
                    changed: frame.get(l.channel),
                    onChange: (value) => {
                      onChange((prev) => {
                        console.log("onChange", value, prev);
                        if (prev.data == null) return prev;
                        const next = state.executeSetter(value, prev.data);
                        return successResult(name, "retrieved", next);
                      });
                    },
                  });
                } catch (error) {
                  onChange(errorResult<Data>(name, "retrieve", error));
                }
              })(),
          })),
        );
        onChange(successResult<Data>(name, "retrieved", value));
      } catch (error) {
        onChange(errorResult<Data>(name, "retrieve", error));
      }
    },
    [client, name, mountListeners],
  );
  const retrieveSync = useCallback(
    (
      params: state.SetArg<RetrieveParams, Partial<RetrieveParams>>,
      options?: { signal?: AbortSignal },
    ) => void retrieveAsync(params, options),
    [retrieveAsync],
  );
  return {
    retrieve: retrieveSync,
    retrieveAsync,
  };
};

const useStateful = <RetrieveParams extends Params, Data extends state.State>(
  args: CreateRetrieveArgs<RetrieveParams, Data>,
): UseStatefulRetrieveReturn<RetrieveParams, Data> => {
  const [state, setState] = useState<Result<Data>>(
    pendingResult<Data>(args.name, "retrieving"),
  );
  return {
    ...state,
    ...useObservable({ ...args, onChange: setState }),
  };
};

const useDirect = <RetrieveParams extends Params, Data extends state.State>({
  params,
  ...restArgs
}: UseDirectRetrieveArgs<RetrieveParams> &
  CreateRetrieveArgs<RetrieveParams, Data>): UseDirectRetrieveReturn<Data> => {
  const { retrieveAsync, retrieve: _, ...rest } = useStateful(restArgs);
  const memoParams = useMemoDeepEqual(params);
  useAsyncEffect(
    async (signal) => await retrieveAsync(memoParams, { signal }),
    [retrieveAsync, memoParams],
  );
  return rest;
};

const useEffect = <RetrieveParams extends Params, Data extends state.State>({
  params,
  ...restArgs
}: UseEffectRetrieveArgs<RetrieveParams, Data> &
  CreateRetrieveArgs<RetrieveParams, Data>): void => {
  const { retrieveAsync } = useObservable<RetrieveParams, Data>(restArgs);
  const memoParams = useMemoDeepEqual(params);
  useAsyncEffect(
    async (signal) => await retrieveAsync(memoParams, { signal }),
    [retrieveAsync, memoParams],
  );
};

/**
 * Creates a retrieve query system that provides hooks for fetching data.
 *
 * This function creates a set of React hooks that handle data retrieval with
 * proper loading states, error handling, caching, and real-time updates. It provides
 * multiple hook variants for different use cases:
 *
 * - `useObservable`: For external state management.
 * - `useStateful`: For internal state management with manual control of query execution.
 * - `useDirect`: For automatic retrieval based on parameters.
 * - `useEffect`: For side-effect based retrieval.
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
 *   name: "user",
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
 * // Usage with automatic retrieval
 * const { data, variant, error } = userRetrieve.useDirect({
 *   params: { userId: 123, includeProfile: true }
 * });
 *
 * // Usage with manual control
 * const { data, variant, error, retrieve } = userRetrieve.useStateful();
 * retrieve({ userId: 123 });
 * ```
 */
export const createRetrieve = <RetrieveParams extends Params, Data extends state.State>(
  factoryArgs: CreateRetrieveArgs<RetrieveParams, Data>,
): CreateRetrieveReturn<RetrieveParams, Data> => ({
  useObservable: (args: UseObservableRetrieveArgs<Data>) =>
    useObservable({ ...factoryArgs, ...args }),
  useStateful: () => useStateful(factoryArgs),
  useDirect: (args: UseDirectRetrieveArgs<RetrieveParams>) =>
    useDirect({ ...factoryArgs, ...args }),
  useEffect: (args: UseEffectRetrieveArgs<RetrieveParams, Data>) =>
    useEffect({ ...factoryArgs, ...args }),
});
