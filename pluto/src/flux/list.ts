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
import { useCallback, useRef, useState, useSyncExternalStore } from "react";

import { useMountListeners } from "@/flux/listeners";
import { type Params } from "@/flux/params";
import {
  errorResult,
  nullClientResult,
  pendingResult,
  type Result,
  successResult,
} from "@/flux/result";
import {
  type AsyncOptions,
  type CreateRetrieveArgs,
  type UseStatefulRetrieveReturn,
} from "@/flux/retrieve";
import { type Sync } from "@/flux/sync";
import { useInitializerRef } from "@/hooks";
import { state } from "@/state";
import { Synnax as PSynnax } from "@/synnax";

interface GetItem<K extends record.Key, E extends record.Keyed<K>> {
  (key: K): E | undefined;
  (keys: K[]): E[];
}

export type UseListReturn<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> = Omit<UseStatefulRetrieveReturn<RetrieveParams, K[]>, "data"> & {
  data: K[];
  useListItem: (key?: K) => E | undefined;
  getItem: GetItem<K, E>;
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

export interface ListListenerExtraArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  params: RetrieveParams;
  client: Synnax;
  onChange: (key: K, e: state.SetArg<E>) => void;
  onDelete: (key: K) => void;
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

interface ListenersRef<K extends record.Key> {
  mounted: boolean;
  listeners: Map<() => void, K>;
}

export const createList =
  <P extends Params, K extends record.Key, E extends record.Keyed<K>>({
    name,
    listeners,
    retrieve,
    retrieveByKey,
  }: CreateListArgs<P, K, E>): UseList<P, K, E> =>
  () => {
    const client = PSynnax.use();
    const dataRef = useRef<Map<K, E>>(new Map());
    const itemListenersRef = useInitializerRef<ListenersRef<K>>(() => ({
      mounted: false,
      listeners: new Map(),
    }));
    const [result, setResult] = useState<Result<K[]>>(
      pendingResult(name, "retrieving"),
    );

    const paramsRef = useRef<P | {}>({});

    const mountListeners = useMountListeners();
    const retrieveAsync = useCallback(
      async (paramsSetter: state.SetArg<P, P | {}>, options: AsyncOptions = {}) => {
        const { signal } = options;
        const params = state.executeSetter(paramsSetter, paramsRef.current);
        paramsRef.current = params;
        try {
          if (client == null) return setResult(nullClientResult(name, "retrieve"));
          setResult(pendingResult(name, "retrieving"));
          const value = await retrieve({ client, params });
          const keys = value.map((v) => v.key);
          value.forEach((v) => dataRef.current.set(v.key, v));
          if (signal?.aborted) return;
          mountListeners(
            listeners?.map((l) => ({
              channel: l.channel,
              handler: (frame) =>
                void (async () => {
                  if (client == null) return;
                  try {
                    await l.onChange({
                      client,
                      params: paramsRef.current,
                      changed: frame.get(l.channel),
                      onDelete: (k) => {
                        dataRef.current.delete(k);
                        setResult(
                          (p) =>
                            ({
                              ...p,
                              data: p.data?.filter((key) => key !== k),
                            }) as Result<K[]>,
                        );
                      },
                      onChange: (k, setter) => {
                        const v = dataRef.current.get(k);
                        if (v == null) return;
                        const res = state.executeSetter(setter, v);
                        dataRef.current.set(k, res);
                        itemListenersRef.current.listeners.forEach((key, listener) => {
                          if (key === k) listener();
                        });
                      },
                    });
                  } catch (error) {
                    setResult(errorResult(name, "retrieve", error));
                  }
                })(),
            })),
          );
          return setResult(successResult(name, "retrieved", keys));
        } catch (error) {
          setResult(errorResult(name, "retrieve", error));
        }
      },
      [client, name, mountListeners],
    );

    const retrieveSingle = useCallback(
      (key: K, options: AsyncOptions = {}) => {
        const { signal } = options;
        void (async () => {
          try {
            if (client == null) return;
            dataRef.current.set(key, await retrieveByKey({ client, key }));
            if (signal?.aborted) return;
            itemListenersRef.current.listeners.forEach((k, listener) => {
              if (k === key) listener();
            });
          } catch (error) {
            setResult(errorResult(name, "retrieve", error));
          }
        })();
      },
      [dataRef, retrieveByKey, client],
    );

    const getItem = useCallback(
      ((key: K | K[]) => {
        if (Array.isArray(key))
          return key.map((k) => dataRef.current.get(k)).filter((v) => v != null);
        return dataRef.current.get(key);
      }) as GetItem<K, E>,
      [],
    );

    const useListItem = (key?: K) => {
      const abortControllerRef = useRef<AbortController | null>(null);
      return useSyncExternalStore<E | undefined>(
        useCallback(
          (callback) => {
            if (key == null) return () => {};
            abortControllerRef.current = new AbortController();
            itemListenersRef.current.listeners.set(callback, key);
            return () => {
              itemListenersRef.current.listeners.delete(callback);
              abortControllerRef.current?.abort();
            };
          },
          [key],
        ),
        useCallback(() => {
          if (key == null) return undefined;
          const res = dataRef.current.get(key);
          if (res == null)
            retrieveSingle(key, { signal: abortControllerRef.current?.signal });
          return res;
        }, [key]),
      );
    };

    const retrieveSync = useCallback(
      (params: state.SetArg<P, P | {}>, options: AsyncOptions = {}) =>
        void retrieveAsync(params, options),
      [retrieveAsync],
    );

    return {
      retrieve: retrieveSync,
      retrieveAsync,
      useListItem,
      getItem,
      ...result,
      data: result?.data ?? [],
    };
  };
