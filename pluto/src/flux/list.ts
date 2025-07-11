// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { compare, type MultiSeries, type record } from "@synnaxlabs/x";
import { useCallback, useMemo, useRef, useSyncExternalStore } from "react";

import { useMountListeners } from "@/flux/listeners";
import { type Params } from "@/flux/params";
import {
  errorResult,
  nullClientResult,
  pendingResult,
  type Result,
  successResult,
} from "@/flux/result";
import { type AsyncOptions, type CreateRetrieveArgs } from "@/flux/retrieve";
import { type Sync } from "@/flux/sync";
import {
  useCombinedStateAndRef,
  useDebouncedCallback,
  useInitializerRef,
} from "@/hooks";
import { state } from "@/state";
import { Synnax as PSynnax } from "@/synnax";

interface GetItem<K extends record.Key, E extends record.Keyed<K>> {
  (key: K): E | undefined;
  (keys: K[]): E[];
}

interface AsyncListOptions extends AsyncOptions {
  mode?: "append" | "replace";
}

export type UseListReturn<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> = Omit<Result<K[]>, "data"> & {
  retrieve: (
    params: state.SetArg<RetrieveParams, Partial<RetrieveParams>>,
    options?: AsyncListOptions,
  ) => void;
  retrieveAsync: (
    params: state.SetArg<RetrieveParams, Partial<RetrieveParams>>,
    options?: AsyncListOptions,
  ) => Promise<void>;
  data: K[];
  useListItem: (key?: K) => E | undefined;
  getItem: GetItem<K, E>;
};

export interface RetrieveByKeyArgs<
  RetrieveParams extends Params,
  K extends record.Key,
> {
  params: RetrieveParams;
  key: K;
  client: Synnax;
}

export interface CreateListArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> extends Omit<CreateRetrieveArgs<RetrieveParams, E[]>, "listeners"> {
  retrieveByKey: (args: RetrieveByKeyArgs<RetrieveParams, K>) => Promise<E>;
  listeners?: ListListenerConfig<RetrieveParams, K, E>[];
  filter?: (item: E) => boolean;
}

export interface UseListArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  initialParams?: RetrieveParams;
  filter?: (item: E) => boolean;
}

export interface UseList<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  (args?: UseListArgs<RetrieveParams, K, E>): UseListReturn<RetrieveParams, K, E>;
}

interface ListListenerExtraArgs<
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

const defaultFilter = () => true;

export const createList =
  <P extends Params, K extends record.Key, E extends record.Keyed<K>>({
    name,
    listeners,
    retrieve,
    retrieveByKey,
  }: CreateListArgs<P, K, E>): UseList<P, K, E> =>
  (args: UseListArgs<P, K, E> = {}) => {
    const { filter = defaultFilter, initialParams } = args;
    const client = PSynnax.use();
    const dataRef = useRef<Map<K, E | null>>(new Map());
    const listenersRef = useInitializerRef<ListenersRef<K>>(() => ({
      mounted: false,
      listeners: new Map(),
    }));
    const [result, setResult, resultRef] = useCombinedStateAndRef<Result<K[]>>(
      pendingResult<K[]>(name, "retrieving"),
    );
    const hasMoreRef = useRef(true);

    const paramsRef = useRef<P | null>(initialParams ?? null);

    const mountListeners = useMountListeners();
    const retrieveAsync = useCallback(
      async (paramsSetter: state.SetArg<P, P | {}>, options: AsyncListOptions = {}) => {
        const { signal, mode = "replace" } = options;
        const params = state.executeSetter(paramsSetter, paramsRef.current ?? {});
        paramsRef.current = params;
        try {
          if (client == null) return setResult(nullClientResult<K[]>(name, "retrieve"));
          setResult((p) => pendingResult(name, "retrieving", p.data ?? []));
          if (mode === "replace") hasMoreRef.current = true;
          else if (mode === "append" && !hasMoreRef.current) return;
          const value = await retrieve({ client, params });
          if (value.length === 0) hasMoreRef.current = false;
          const keys = value.map((v) => v.key);
          if (
            resultRef.current.data != null &&
            compare.primitiveArrays(resultRef.current.data, keys) === compare.EQUAL
          )
            return;

          value.forEach((v) => {
            if (filter(v)) dataRef.current.set(v.key, v);
          });
          if (signal?.aborted) return;
          mountListeners(
            listeners?.map((l) => ({
              channel: l.channel,
              handler: (frame) =>
                void (async () => {
                  if (client == null || paramsRef.current == null) return;
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
                        if (v == null || !filter(v)) return;
                        const res = state.executeSetter(setter, v);
                        dataRef.current.set(k, res);
                        listenersRef.current.listeners.forEach((key, listener) => {
                          if (key === k) listener();
                        });
                      },
                    });
                  } catch (error) {
                    setResult(errorResult<K[]>(name, "retrieve", error));
                  }
                })(),
            })),
          );
          return setResult((prev) => {
            if (mode === "replace" || prev.data == null)
              return successResult(name, "retrieved", keys);
            const keysSet = new Set(keys);
            return successResult(name, "retrieved", [
              ...prev.data.filter((k) => !keysSet.has(k)),
              ...keys,
            ]);
          });
        } catch (error) {
          setResult(errorResult<K[]>(name, "retrieve", error));
        }
      },
      [client, name, mountListeners],
    );

    const retrieveSingle = useCallback(
      (key: K, options: AsyncOptions = {}) => {
        const { signal } = options;
        void (async () => {
          try {
            if (client == null || paramsRef.current == null) return;
            const item = await retrieveByKey({
              client,
              key,
              params: paramsRef.current,
            });
            if (!filter(item)) {
              dataRef.current.set(key, null);
              return;
            }
            dataRef.current.set(key, item);
            if (signal?.aborted) return;
            listenersRef.current.listeners.forEach((k, listener) => {
              if (k === key) listener();
            });
          } catch (error) {
            dataRef.current.set(key, null);
            setResult(errorResult<K[]>(name, "retrieve", error));
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

    const useListItem = useCallback((key?: K) => {
      if (key == null) return undefined;
      const abortControllerRef = useRef<AbortController | null>(null);
      return useSyncExternalStore<E | undefined>(
        useCallback(
          (callback) => {
            abortControllerRef.current = new AbortController();
            listenersRef.current.listeners.set(callback, key);
            return () => {
              listenersRef.current.listeners.delete(callback);
              abortControllerRef.current?.abort();
            };
          },
          [key],
        ),
        useCallback(() => {
          const res = dataRef.current.get(key);
          if (res === undefined)
            retrieveSingle(key, { signal: abortControllerRef.current?.signal });
          return res ?? undefined;
        }, [key]),
      );
    }, []);

    const retrieveSync = useDebouncedCallback(
      (params: state.SetArg<P, P | {}>, options: AsyncListOptions = {}) =>
        void retrieveAsync(params, options),
      100,
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

export interface PagerParams extends Params {
  term?: string;
  offset?: number;
  limit?: number;
}

export interface UsePagerReturn {
  onFetchMore: () => void;
  onSearch: (term: string) => void;
}

export interface UsePagerArgs
  extends Pick<UseListReturn<PagerParams, any, any>, "retrieve"> {
  pageSize?: number;
}

export const usePager = ({ retrieve, pageSize = 10 }: UsePagerArgs): UsePagerReturn =>
  useMemo(
    () => ({
      onFetchMore: () =>
        retrieve(
          ({ offset = -pageSize, term }) => ({
            offset: offset + pageSize,
            limit: pageSize,
            term,
          }),
          { mode: "append" },
        ),
      onSearch: (term) => retrieve({ term, offset: 0, limit: pageSize }),
    }),
    [retrieve],
  );
