// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import {
  compare,
  type CrudeTimeSpan,
  type Destructor,
  type MultiSeries,
  primitive,
  type record,
  TimeSpan,
} from "@synnaxlabs/x";
import { useCallback, useRef, useSyncExternalStore } from "react";

import { useMountSynchronizers } from "@/flux/listeners";
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
  useSyncedRef,
} from "@/hooks";
import { state } from "@/state";
import { Synnax as PSynnax } from "@/synnax";

interface GetItem<K extends record.Key, E extends record.Keyed<K>> {
  (key?: K): E | undefined;
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
  getItem: GetItem<K, E>;
  subscribe: (callback: () => void, key?: K) => Destructor;
};

export interface RetrieveByKeyArgs<
  RetrieveParams extends Params,
  K extends record.Key,
> {
  params: Partial<RetrieveParams>;
  key: K;
  client: Synnax;
}

export interface CreateListArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> extends Omit<CreateRetrieveArgs<RetrieveParams, E[]>, "listeners"> {
  retrieveByKey: (args: RetrieveByKeyArgs<RetrieveParams, K>) => Promise<E | undefined>;
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
  retrieveDebounce?: CrudeTimeSpan;
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
const DEFAULT_RETRIEVE_DEBOUNCE = TimeSpan.milliseconds(100);

export const createList =
  <P extends Params, K extends record.Key, E extends record.Keyed<K>>({
    name,
    listeners,
    retrieve,
    retrieveByKey,
  }: CreateListArgs<P, K, E>): UseList<P, K, E> =>
  (args: UseListArgs<P, K, E> = {}) => {
    const {
      filter = defaultFilter,
      initialParams,
      retrieveDebounce = DEFAULT_RETRIEVE_DEBOUNCE,
    } = args;
    const filterRef = useSyncedRef(filter);
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

    const mountSynchronizers = useMountSynchronizers();

    const notifyListeners = useCallback(
      (changed: K) =>
        listenersRef.current.listeners.forEach((key, notify) => {
          if (key === changed) notify();
        }),
      [listenersRef],
    );

    const retrieveAsync = useCallback(
      async (paramsSetter: state.SetArg<P, P | {}>, options: AsyncListOptions = {}) => {
        const { signal, mode = "replace" } = options;

        const params = state.executeSetter(paramsSetter, paramsRef.current ?? {});
        paramsRef.current = params;

        try {
          if (client == null) return setResult(nullClientResult<K[]>(name, "retrieve"));
          setResult((p) => pendingResult(name, "retrieving", p.data));

          // If we're in replace mode, we're 'resetting' the infinite scroll position
          // of the query, so we start from the top again.
          if (mode === "replace") hasMoreRef.current = true;
          else if (mode === "append" && !hasMoreRef.current) return;

          let value = await retrieve({ client, params });
          if (signal?.aborted) return;
          value = value.filter(filterRef.current);
          if (value.length === 0) hasMoreRef.current = false;
          const keys = value.map((v) => v.key);

          // If we've already retrieved the initial data, and it's the same as the
          // data we just retrieved, then don't notify listeners.
          if (
            resultRef.current.data != null &&
            compare.primitiveArrays(resultRef.current.data, keys) === compare.EQUAL
          )
            return setResult((p) => successResult(name, "retrieved", p.data ?? []));

          value.forEach((v) => dataRef.current.set(v.key, v));

          mountSynchronizers(
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
                        setResult((p) => {
                          if (p.data == null) return p;
                          return { ...p, data: p.data.filter((key) => key !== k) };
                        });
                      },
                      onChange: (k, setter) => {
                        const v = dataRef.current.get(k);
                        if (v == null || !filterRef.current(v)) return;
                        const res = state.executeSetter(setter, v);
                        dataRef.current.set(k, res);
                        notifyListeners(k);
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
      [client, name, mountSynchronizers, filterRef],
    );

    const retrieveSingle = useCallback(
      (key: K, options: AsyncOptions = {}) => {
        const { signal } = options;
        void (async () => {
          try {
            if (client == null || primitive.isZero(key)) return;
            const item = await retrieveByKey({
              client,
              key,
              params: paramsRef.current ?? {},
            });
            if (item == null) return;
            if (!filterRef.current(item)) {
              dataRef.current.set(key, null);
              return;
            }
            dataRef.current.set(key, item);
            if (signal?.aborted) return;
            notifyListeners(key);
          } catch (error) {
            dataRef.current.set(key, null);
            setResult(errorResult<K[]>(name, "retrieve", error));
          }
        })();
      },
      [retrieveByKey, client, notifyListeners],
    );

    const getItem = useCallback(
      ((key?: K | K[]) => {
        if (key == null) return undefined;
        if (Array.isArray(key))
          return key.map((k) => getItem(k)).filter((v) => v != null);
        const res = dataRef.current.get(key);
        if (res === undefined) retrieveSingle(key);
        return res;
      }) as GetItem<K, E>,
      [retrieveSingle],
    );

    const subscribe = useCallback((callback: () => void, key?: K) => {
      if (key == null) return () => {};
      listenersRef.current.listeners.set(callback, key);
      return () => listenersRef.current.listeners.delete(callback);
    }, []);

    const retrieveSync = useDebouncedCallback(
      (params: state.SetArg<P, P | {}>, options: AsyncListOptions = {}) =>
        void retrieveAsync(params, options),
      new TimeSpan(retrieveDebounce).milliseconds,
      [retrieveAsync],
    );

    return {
      retrieve: retrieveSync,
      retrieveAsync,
      subscribe,
      getItem,
      ...result,
      data: result?.data ?? [],
    };
  };

export interface UseListItemArgs<K extends record.Key, E extends record.Keyed<K>>
  extends Pick<UseListReturn<Params, K, E>, "subscribe" | "getItem"> {
  key?: K;
}

export const useListItem = <K extends record.Key, E extends record.Keyed<K>>({
  key,
  subscribe,
  getItem,
}: UseListItemArgs<K, E>) =>
  useSyncExternalStore(
    useCallback((callback) => subscribe(callback, key), [subscribe, key]),
    useCallback(() => getItem(key), [getItem, key]),
  );
