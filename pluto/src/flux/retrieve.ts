// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type Synnax } from "@synnaxlabs/client";
import { type Destructor, type MultiSeries } from "@synnaxlabs/x";
import {
  useCallback,
  useEffect as reactUseEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { type Params } from "@/flux/params";
import {
  errorResult,
  nullClientResult,
  pendingResult,
  type Result,
  successResult,
} from "@/flux/result";
import { Sync } from "@/flux/sync";
import { useAsyncEffect } from "@/hooks";
import { useMemoDeepEqual } from "@/memo";
import { state } from "@/state";
import { Synnax as PSynnax } from "@/synnax";

interface RetrieveListenerExtraArgs<
  RetrieveParams extends Params,
  Data extends state.State,
> {
  params: RetrieveParams;
  client: Synnax;
  onChange: state.Setter<Data>;
}

/**
 * Configuration for a listener that is called whenever a new value is received
 * from the specified channel. The listener is called with the new value and can
 * choose to update the state of the query by calling the `onChange` function.
 *
 * The listener will not be called if the query is in a loading or an error state.
 */
export interface RetrieveListenerConfig<
  RetrieveParams extends Params,
  Data extends state.State,
> {
  /** The channel to listen to. */
  channel: channel.Name;
  /** The function to call when a new value is received. */
  onChange: Sync.ListenerHandler<
    MultiSeries,
    RetrieveListenerExtraArgs<RetrieveParams, Data>
  >;
}

/**
 * Arguments passed to the `retrieve` function on the query.
 */
export interface RetrieveArgs<RetrieveParams extends Params> {
  client: Synnax;
  params: RetrieveParams;
}

/**
 * Arguments passed to the `useBase` hook.
 * @template QParams - The type of the parameters for the query.
 * @template Data - The type of the data being retrieved.
 */
export interface CreateRetrieveArgs<
  RetrieveParams extends Params,
  Data extends state.State,
> {
  /**
   * The name of the resource being retrieve. This is used to make pretty messages for
   * the various query states.
   */
  name: string;
  /** Executed when the query is first created, or whenever the query parameters change. */
  retrieve: (args: RetrieveArgs<RetrieveParams>) => Promise<Data>;
  /**
   * Listeners to mount to the query. These listeners will be re-mounted when
   * the query parameters changed and/or the client disconnects/re-connects or clusters
   * are switched.
   *
   * These listeners will NOT be remounted when the identity of the onChange function
   * changes.
   */
  listeners?: RetrieveListenerConfig<RetrieveParams, Data>[];
}

export interface UseObservableRetrieveArgs<V extends state.State> {
  onChange: state.Setter<Result<V>>;
}

export interface UseObservableRetrieveReturn<RetrieveParams extends Params> {
  retrieve: (
    params: state.SetArg<RetrieveParams, RetrieveParams | {}>,
    options: { signal?: AbortSignal },
  ) => void;
  retrieveAsync: (
    params: state.SetArg<RetrieveParams, RetrieveParams | {}>,
    options: { signal?: AbortSignal },
  ) => Promise<void>;
}

export type UseStatefulRetrieveReturn<
  P extends Params,
  V extends state.State,
> = Result<V> & UseObservableRetrieveReturn<P>;

export interface UseDirectRetrieveArgs<RetrieveParams extends Params> {
  params: RetrieveParams;
}

export type UseDirectRetrieveReturn<V extends state.State> = Result<V>;

export interface UseEffectRetrieveArgs<
  RetrieveParams extends Params,
  V extends state.State,
> extends UseObservableRetrieveArgs<V> {
  params: RetrieveParams;
}

export interface CreateRetrieveReturn<
  RetrieveParams extends Params,
  V extends state.State,
> {
  useObservable: (
    args: UseObservableRetrieveArgs<V>,
  ) => UseObservableRetrieveReturn<RetrieveParams>;
  useStateful: () => UseStatefulRetrieveReturn<RetrieveParams, V>;
  useDirect: (
    args: UseDirectRetrieveArgs<RetrieveParams>,
  ) => UseDirectRetrieveReturn<V>;
  useEffect: (args: UseEffectRetrieveArgs<RetrieveParams, V>) => void;
}

const useObservable = <RetrieveParams extends Params, Data extends state.State>({
  retrieve,
  listeners,
  name,
  onChange,
}: UseObservableRetrieveArgs<Data> &
  CreateRetrieveArgs<
    RetrieveParams,
    Data
  >): UseObservableRetrieveReturn<RetrieveParams> => {
  const addListener = Sync.useAddListener();
  const destructorsRef = useRef<Destructor[]>([]);
  const cleanupDestructors = useCallback(() => {
    destructorsRef.current.forEach((d) => d());
    destructorsRef.current = [];
  }, []);
  reactUseEffect(() => cleanupDestructors, [cleanupDestructors]);
  const client = PSynnax.use();
  const paramsRef = useRef<RetrieveParams | {}>({});
  const retrieveAsync = useCallback(
    async (
      paramsSetter: state.SetArg<RetrieveParams, RetrieveParams | {}>,
      { signal }: { signal?: AbortSignal },
    ) => {
      const params = state.executeSetter<RetrieveParams, RetrieveParams | {}>(
        paramsSetter,
        paramsRef.current,
      );
      paramsRef.current = params;
      try {
        cleanupDestructors();
        if (client == null) return onChange(nullClientResult(name, "retrieve"));
        onChange(pendingResult(name, "retrieving"));
        const value = await retrieve({ client, params });
        if (signal?.aborted) return;
        if (listeners == null || listeners.length === 0)
          return onChange(successResult(name, "retrieved", value));
        destructorsRef.current = listeners.map(
          ({ channel, onChange: listenerOnChange }, i) =>
            addListener({
              channel,
              onOpen: () =>
                i === listeners.length - 1 &&
                onChange(successResult(name, "retrieved", value)),
              handler: (frame) => {
                void (async () => {
                  try {
                    await listenerOnChange({
                      client,
                      params,
                      changed: frame.get(channel),
                      onChange: (value) => {
                        onChange((prev) => {
                          if (prev.data == null) return prev;
                          const next = state.executeSetter(value, prev.data);
                          return successResult(name, "retrieved", next);
                        });
                      },
                    });
                  } catch (error) {
                    onChange(errorResult(name, "retrieve", error));
                  }
                })();
              },
            }),
        );
      } catch (error) {
        onChange(errorResult(name, "retrieve", error));
      }
    },
    [client, name, addListener],
  );
  const retrieveSync = useCallback(
    (
      params: state.SetArg<RetrieveParams, RetrieveParams | {}>,
      options: { signal?: AbortSignal },
    ) => void retrieveAsync(params, options),
    [retrieveAsync],
  );
  return {
    retrieve: retrieveSync,
    retrieveAsync,
  };
};

const useStateful = <RetrieveParams extends Params, V extends state.State>(
  args: CreateRetrieveArgs<RetrieveParams, V>,
): UseStatefulRetrieveReturn<RetrieveParams, V> => {
  const [state, setState] = useState<Result<V>>(pendingResult(args.name, "retrieving"));
  return {
    ...state,
    ...useObservable({ ...args, onChange: setState }),
  };
};

const useDirect = <RetrieveParams extends Params, V extends state.State>({
  params,
  ...restArgs
}: UseDirectRetrieveArgs<RetrieveParams> &
  CreateRetrieveArgs<RetrieveParams, V>): UseDirectRetrieveReturn<V> => {
  const { retrieveAsync, retrieve: _, ...rest } = useStateful(restArgs);
  const memoParams = useMemoDeepEqual(params);
  useAsyncEffect(
    async (signal) => await retrieveAsync(memoParams, { signal }),
    [retrieveAsync, memoParams],
  );
  return rest;
};

const useEffect = <RetrieveParams extends Params, V extends state.State>({
  params,
  ...restArgs
}: UseEffectRetrieveArgs<RetrieveParams, V> &
  CreateRetrieveArgs<RetrieveParams, V>): void => {
  const { retrieveAsync } = useObservable<RetrieveParams, V>(restArgs);
  const memoParams = useMemo(() => params, [params]);
  useAsyncEffect(
    async (signal) => await retrieveAsync(memoParams, { signal }),
    [retrieveAsync, memoParams],
  );
};

export const createRetrieve = <RetrieveParams extends Params, V extends state.State>(
  factoryArgs: CreateRetrieveArgs<RetrieveParams, V>,
): CreateRetrieveReturn<RetrieveParams, V> => ({
  useObservable: (args: UseObservableRetrieveArgs<V>) =>
    useObservable({ ...factoryArgs, ...args }),
  useStateful: () => useStateful(factoryArgs),
  useDirect: (args: UseDirectRetrieveArgs<RetrieveParams>) =>
    useDirect({ ...factoryArgs, ...args }),
  useEffect: (args: UseEffectRetrieveArgs<RetrieveParams, V>) =>
    useEffect({ ...factoryArgs, ...args }),
});
