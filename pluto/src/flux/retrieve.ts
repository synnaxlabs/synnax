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

import { useMountListeners } from "@/flux/listeners";
import { type Params } from "@/flux/params";
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

export interface AsyncOptions {
  signal?: AbortSignal;
}

export interface UseObservableRetrieveReturn<RetrieveParams extends Params> {
  retrieve: (
    params: state.SetArg<RetrieveParams, RetrieveParams | {}>,
    options?: AsyncOptions,
  ) => void;
  retrieveAsync: (
    params: state.SetArg<RetrieveParams, RetrieveParams | {}>,
    options?: AsyncOptions,
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
  const mountListeners = useMountListeners();
  const retrieveAsync = useCallback(
    async (
      paramsSetter: state.SetArg<RetrieveParams, RetrieveParams | {}>,
      options: AsyncOptions = {},
    ) => {
      const { signal } = options;
      const params = state.executeSetter<RetrieveParams, RetrieveParams | {}>(
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
        onChange(successResult(name, "retrieved", value));
      } catch (error) {
        onChange(errorResult<Data>(name, "retrieve", error));
      }
    },
    [client, name, mountListeners],
  );
  const retrieveSync = useCallback(
    (
      params: state.SetArg<RetrieveParams, RetrieveParams | {}>,
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
