// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { type MultiSeries, type record } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";

import { type Params } from "@/flux/params";
import {
  errorResult,
  nullClientResult,
  pendingResult,
  type Result,
  successResult,
} from "@/flux/result";
import {
  type CreateRetrieveArgs,
  type UseStatefulRetrieveReturn,
} from "@/flux/retrieve";
import { Sync } from "@/flux/sync";
import { state } from "@/state";
import { Synnax as PSynnax } from "@/synnax";

export type UseListReturn<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> = Omit<UseStatefulRetrieveReturn<RetrieveParams, K[]>, "data"> & {
  data: K[];
  useListItem: (key: K) => E | undefined;
};

export interface RetrieveByKeyArgs<K extends record.Key> {
  key: K;
  client: Synnax;
}

export interface CreateListArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> extends Omit<CreateRetrieveArgs<RetrieveParams, E[]>, "listeners"> {
  retrieveByKey: (args: RetrieveByKeyArgs<K>) => Promise<E>;
  listeners?: ListListenerConfig<RetrieveParams | {}, K, E>[];
}

export interface UseList<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  (): UseListReturn<RetrieveParams, K, E>;
}

interface ListListenerExtraArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  params: RetrieveParams;
  client: Synnax;
  onChange: (key: K, e: state.Setter<E | null>) => void;
}

export interface ListListenerConfig<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  channel: string;
  onChange: Sync.ListenerHandler<
    MultiSeries,
    ListListenerExtraArgs<RetrieveParams, K, E>
  >;
}

export const createList =
  <P extends Params, K extends record.Key, E extends record.Keyed<K>>({
    name,
    listeners,
    retrieve,
    retrieveByKey,
  }: CreateListArgs<P, K, E>): UseList<P, K, E> =>
  () => {
    const dataRef = useRef<Map<K, E>>(new Map());
    const listenersRef = useRef<Map<() => void, K>>(new Map());
    const [result, setResult] = useState<Result<K[]>>(
      pendingResult(name, "retrieving"),
    );
    const paramsRef = useRef<P | {}>({});
    const addListener = Sync.useAddListener();
    const client = PSynnax.use();
    useEffect(() => {
      if (listeners == null || listeners.length === 0 || client == null)
        return () => {};
      const destructors = listeners.map(({ channel, onChange: listenerOnChange }) =>
        addListener({
          channel,
          handler: (frame) => {
            void (async () => {
              try {
                await listenerOnChange({
                  client,
                  params: paramsRef.current,
                  changed: frame.get(channel),
                  onChange: (k, setter) => {
                    const v = dataRef.current.get(k);
                    if (v == null) return;
                    const res = setter(v);
                    if (res == null) return;
                    dataRef.current.set(k, res);
                    listenersRef.current.forEach((key, listener) => {
                      if (key === k) listener();
                    });
                  },
                });
              } catch (error) {
                setResult(errorResult(name, "retrieve", error));
              }
            })();
          },
        }),
      );
      return () => destructors.forEach((d) => d());
    }, [addListener, client]);

    const retrieveAsync = useCallback(
      async (
        paramsSetter: state.SetArg<P, P | {}>,
        { signal }: { signal?: AbortSignal },
      ) => {
        const params = state.executeSetter(paramsSetter, paramsRef.current);
        paramsRef.current = params;
        try {
          if (client == null) return setResult(nullClientResult(name, "retrieve"));
          setResult(pendingResult(name, "retrieving"));
          const value = await retrieve({ client, params });
          const keys = value.map((v) => v.key);
          if (signal?.aborted) return;
          return setResult(successResult(name, "retrieved", keys));
        } catch (error) {
          setResult(errorResult(name, "retrieve", error));
        }
      },
      [client, name, addListener],
    );

    const retrieveSingle = useCallback(
      (key: K) => {
        void (async () => {
          try {
            if (client == null) return;
            dataRef.current.set(key, await retrieveByKey({ client, key }));
            listenersRef.current.forEach((k, listener) => {
              if (k === key) listener();
            });
          } catch (error) {
            setResult(errorResult(name, "retrieve", error));
          }
        })();
      },
      [dataRef, retrieveByKey, client],
    );

    const useListItem = (key: K) =>
      useSyncExternalStore<E | undefined>(
        useCallback(
          (callback) => {
            listenersRef.current.set(callback, key);
            return () => listenersRef.current.delete(callback);
          },
          [key],
        ),
        useCallback(() => {
          const res = dataRef.current.get(key);
          if (res == null) retrieveSingle(key);
          return res;
        }, [key]),
      );

    const retrieveSync = useCallback(
      (params: state.SetArg<P, P | {}>, options: { signal?: AbortSignal }) =>
        void retrieveAsync(params, options),
      [retrieveAsync],
    );

    return {
      retrieve: retrieveSync,
      retrieveAsync,
      useListItem,
      ...result,
      data: result?.data ?? [],
    };
  };
