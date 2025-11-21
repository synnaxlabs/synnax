// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  compare,
  type CrudeTimeSpan,
  type destructor,
  primitive,
  type record,
  TimeSpan,
} from "@synnaxlabs/x";
import { type RefObject, useCallback, useRef, useSyncExternalStore } from "react";

import { type flux } from "@/flux/aether";
import { type core } from "@/flux/core";
import { useStore } from "@/flux/Provider";
import {
  errorResult,
  loadingResult,
  nullClientResult,
  type Result,
  successResult,
} from "@/flux/result";
import {
  type CreateRetrieveParams,
  type RetrieveMountListenersParams,
  type RetrieveParams,
} from "@/flux/retrieve";
import {
  useCombinedStateAndRef,
  useDebouncedCallback,
  useDestructors,
  useInitializerRef,
  useSyncedRef,
} from "@/hooks";
import { state } from "@/state";
import { Synnax } from "@/synnax";

export interface GetItem<K extends record.Key, E extends record.Keyed<K>> {
  (key: K): E | undefined;
  (keys: K[]): E[];
}

export interface AsyncListOptions extends core.FetchOptions {
  mode?: "append" | "replace";
}

export type UseListReturn<
  Query extends core.Shape,
  K extends record.Key,
  E extends record.Keyed<K>,
> = Omit<Result<K[]>, "data"> & {
  retrieve: (
    query: state.SetArg<Query, Partial<Query>>,
    options?: AsyncListOptions,
  ) => void;
  retrieveAsync: (
    query: state.SetArg<Query, Partial<Query>>,
    options?: AsyncListOptions,
  ) => Promise<void>;
  data: K[];
  getItem: GetItem<K, E>;
  subscribe: (callback: () => void, key: K) => destructor.Destructor;
};

export interface RetrieveByKeyParams<
  Query extends core.Shape,
  K extends record.Key,
  Store extends flux.Store,
> extends Omit<RetrieveParams<Query, Store>, "query"> {
  query: Partial<Query>;
  key: K;
}

export interface RetrieveCachedParams<
  Query extends core.Shape,
  Store extends flux.Store,
> {
  query: Partial<Query>;
  store: Store;
}

export interface CreateListParams<
  Query extends core.Shape,
  K extends record.Key,
  E extends record.Keyed<K>,
  Store extends flux.Store,
> extends Omit<CreateRetrieveParams<Query, E[], Store>, "mountListeners"> {
  sort?: compare.Comparator<E>;
  retrieveByKey: (args: RetrieveByKeyParams<Query, K, Store>) => Promise<E | undefined>;
  retrieveCached?: (args: RetrieveCachedParams<Query, Store>) => E[];
  mountListeners?: (
    args: ListMountListenersParams<Query, K, E, Store>,
  ) => destructor.Destructor | destructor.Destructor[];
}

export interface UseListParams<
  Query extends core.Shape,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  initialQuery?: Query;
  filter?: (item: E) => boolean;
  sort?: compare.Comparator<E>;
  retrieveDebounce?: CrudeTimeSpan;
  useCachedList?: boolean;
}

export interface UseList<
  Query extends core.Shape,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  (args?: UseListParams<Query, K, E>): UseListReturn<Query, K, E>;
}

type ListChangeMode = "prepend" | "append" | "replace";

interface ListenerOnChangeOptions {
  mode?: ListChangeMode;
}

interface OnListChange<K extends record.Key, E extends record.Keyed<K>> {
  (key: K, e: state.SetArg<E | null>, opts?: ListenerOnChangeOptions): void;
  (e: E, opts?: ListenerOnChangeOptions): void;
}

const parseOnListChangeArgs = <K extends record.Key, E extends record.Keyed<K>>(
  key: K | E,
  e?: state.SetArg<E | null> | ListenerOnChangeOptions,
  opts?: ListenerOnChangeOptions,
): [K, state.SetArg<E | null>, ListenerOnChangeOptions] => {
  if (typeof key === "object" && "key" in key)
    return [key.key, key, (e as ListenerOnChangeOptions | undefined) ?? {}];
  return [key, e as state.SetArg<E | null>, opts ?? {}];
};

export interface ListMountListenersParams<
  Query extends core.Shape,
  Key extends record.Key,
  Data extends record.Keyed<Key>,
  Store extends flux.Store,
> extends Omit<
    RetrieveMountListenersParams<Query, Data[], Store>,
    "onChange" | "query"
  > {
  query: Partial<Query>;
  onChange: OnListChange<Key, Data>;
  onDelete: (key: Key) => void;
}

const defaultFilter = () => true;
const DEFAULT_RETRIEVE_DEBOUNCE = TimeSpan.milliseconds(100);

interface GetInitialDataParams<
  Query extends core.Shape,
  K extends record.Key,
  E extends record.Keyed<K>,
  ScopedStore extends flux.Store,
> {
  retrieveCached: CreateListParams<Query, K, E, ScopedStore>["retrieveCached"];
  queryRef: RefObject<Query | null>;
  filterRef: RefObject<((item: E) => boolean) | undefined>;
  sortRef: RefObject<compare.Comparator<E> | undefined>;
  dataRef: RefObject<Map<K, E | null>>;
  store: ScopedStore;
  useCachedList: boolean;
}

const getInitialData = <
  Query extends core.Shape,
  K extends record.Key,
  E extends record.Keyed<K>,
  ScopedStore extends flux.Store,
>({
  retrieveCached,
  queryRef: paramsRef,
  filterRef,
  sortRef,
  dataRef,
  store,
  useCachedList,
}: GetInitialDataParams<Query, K, E, ScopedStore>) => {
  if (retrieveCached == null || !useCachedList) return undefined;
  let cached = retrieveCached({ query: paramsRef.current ?? {}, store });
  if (filterRef.current != null) cached = cached.filter(filterRef.current);
  if (sortRef.current != null) cached = cached.sort(sortRef.current);
  if (cached.length === 0) return undefined;
  cached.forEach((v) => dataRef.current.set(v.key, v));
  return cached.map((v) => v.key);
};

export const createList =
  <
    Query extends core.Shape,
    Key extends record.Key,
    Data extends record.Keyed<Key>,
    ScopedStore extends flux.Store = {},
  >({
    name,
    mountListeners,
    retrieve,
    retrieveByKey,
    retrieveCached,
    sort: defaultSort,
  }: CreateListParams<Query, Key, Data, ScopedStore>): UseList<Query, Key, Data> =>
  (params: UseListParams<Query, Key, Data> = {}) => {
    const {
      filter = defaultFilter,
      sort,
      initialQuery = null,
      retrieveDebounce = DEFAULT_RETRIEVE_DEBOUNCE,
      useCachedList = true,
    } = params;
    const filterRef = useSyncedRef(filter);
    const sortRef = useSyncedRef(sort ?? defaultSort);
    const client = Synnax.use();
    const dataRef = useRef<Map<Key, Data | null>>(new Map());
    const listItemListeners = useInitializerRef<Map<() => void, Key>>(() => new Map());
    const store = useStore<ScopedStore>();
    const queryRef = useRef<Query | null>(initialQuery);
    const [result, setResult, resultRef] = useCombinedStateAndRef<Result<Key[]>>(() =>
      loadingResult<Key[]>(
        `retrieving ${name}`,
        getInitialData({
          retrieveCached,
          queryRef,
          filterRef,
          sortRef,
          dataRef,
          store,
          useCachedList,
        }),
      ),
    );
    const hasMoreRef = useRef(true);
    const storeListeners = useDestructors();
    const storeListenersMountedRef = useRef(false);

    const notifyListeners = useCallback(
      (changed: Key) =>
        listItemListeners.current.forEach((key, notify) => {
          if (key === changed) notify();
        }),
      [listItemListeners.current],
    );

    const updateSortedData = useCallback(
      (keys: Key[]) => {
        if (sortRef.current == null) return keys;

        const allItems = keys
          .map((key) => dataRef.current.get(key))
          .filter((item): item is Data => item != null);

        allItems.sort(sortRef.current);
        return allItems.map((item) => item.key);
      },
      [sortRef],
    );

    const syncListeners = useCallback(() => {
      if (client == null) return;
      storeListenersMountedRef.current = true;
      storeListeners.cleanup();
      storeListeners.set(
        mountListeners?.({
          client,
          store,
          query: queryRef.current ?? {},
          onDelete: (k) => {
            dataRef.current.delete(k);
            setResult((p) => {
              if (p.data == null) return p;
              return { ...p, data: p.data.filter((key) => key !== k) };
            });
          },
          onChange: (
            argKey: Key | Data,
            argSetter?: state.SetArg<Data | null> | ListenerOnChangeOptions,
            argOptions?: ListenerOnChangeOptions,
          ) => {
            const [k, setter, opts] = parseOnListChangeArgs<Key, Data>(
              argKey,
              argSetter,
              argOptions,
            );
            const { mode = "append" } = opts;
            const prev = dataRef.current.get(k) ?? null;
            if (prev != null && !filterRef.current(prev)) return;
            const res = state.executeSetter(setter, prev);
            if (res == null || !filterRef.current(res)) return;
            dataRef.current.set(k, res);
            setResult((p) => {
              if (p.data == null) return p;
              let newData: Key[];
              if (prev == null)
                if (sortRef.current != null) newData = updateSortedData([...p.data, k]);
                else newData = mode === "prepend" ? [k, ...p.data] : [...p.data, k];
              else if (sortRef.current != null) {
                const currentIndex = p.data.indexOf(k);
                const sortedData = updateSortedData(p.data);
                const newIndex = sortedData.indexOf(k);
                if (currentIndex !== newIndex) newData = sortedData;
                else newData = p.data;
              } else newData = p.data;
              return { ...p, data: newData };
            });

            notifyListeners(k);
          },
        }),
      );
    }, [mountListeners, storeListeners]);

    const retrieveAsync = useCallback(
      async (
        paramsSetter: state.SetArg<Query, Query | {}>,
        options: AsyncListOptions = {},
      ) => {
        const { signal, mode = "replace" } = options;

        const query = state.executeSetter(paramsSetter, queryRef.current ?? {});
        queryRef.current = query;

        try {
          if (client == null)
            return setResult(nullClientResult<Key[]>(`retrieve ${name}`));
          setResult((p) => loadingResult(`retrieving ${name}`, p.data));

          // If we're in replace mode, we're 'resetting' the infinite scroll position
          // of the query, so we start from the top again.
          if (mode === "replace") hasMoreRef.current = true;
          else if (mode === "append" && !hasMoreRef.current)
            return setResult((p) => successResult(`retrieved ${name}`, p.data ?? []));

          let value = await retrieve({ client, query, store });
          if (signal?.aborted) return;
          value = value.filter(filterRef.current);
          if (sortRef.current != null) value = value.sort(sortRef.current);

          if (value.length === 0) hasMoreRef.current = false;
          const keys = value.map((v) => v.key);

          syncListeners();

          // If we've already retrieved the initial data, and it's the same as the
          // data we just retrieved, then don't notify listeners.
          if (
            resultRef.current.data != null &&
            compare.primitiveArrays(resultRef.current.data, keys) === compare.EQUAL
          )
            return setResult((p) => successResult(`retrieved ${name}`, p.data ?? []));

          value.forEach((v) => dataRef.current.set(v.key, v));

          return setResult((prev) => {
            if (mode === "replace" || prev.data == null)
              return successResult(`retrieved ${name}`, keys);
            const keysSet = new Set(keys);
            return successResult(`retrieved ${name}`, [
              ...prev.data.filter((k) => !keysSet.has(k)),
              ...keys,
            ]);
          });
        } catch (error) {
          if (signal?.aborted) return;
          setResult(errorResult(`retrieve ${name}`, error));
        }
      },
      [client, name, store, filterRef, syncListeners],
    );

    const retrieveSingle = useCallback(
      (key: Key, options: core.FetchOptions = {}) => {
        const { signal } = options;
        void (async () => {
          try {
            if (client == null || primitive.isZero(key)) return;
            const item = await retrieveByKey({
              client,
              key,
              query: queryRef.current ?? {},
              store,
            });
            if (signal?.aborted || item == null) return;
            if (!filterRef.current(item)) {
              dataRef.current.set(key, null);
              return;
            }
            dataRef.current.set(key, item);
            notifyListeners(key);
          } catch (error) {
            if (signal?.aborted) return;
            dataRef.current.set(key, null);
            setResult(errorResult(`retrieve ${name}`, error));
          }
        })();
      },
      [retrieveByKey, client, notifyListeners, syncListeners],
    );

    const getItem = useCallback(
      ((key?: Key | Key[]) => {
        if (!storeListenersMountedRef.current) syncListeners();
        if (Array.isArray(key))
          return key.map((k) => getItem(k)).filter((v) => v != null);
        // Zero-value keys that are not null or undefined are common as
        // initialized fields in various data structures ("", 0, etc.).
        // A 'zero-value' is never valid as a key in Synnax, and a simple
        // null check would result in excessive server refetches for
        // keys we already know are invalid, so we do a full check
        // for a zero-value instead.
        if (primitive.isZero(key)) return undefined;
        const res = dataRef.current.get(key);
        if (res === undefined) retrieveSingle(key);
        return res;
      }) as GetItem<Key, Data>,
      [retrieveSingle, syncListeners],
    );

    const subscribe = useCallback((callback: () => void, key?: Key) => {
      if (key == null) return () => {};
      listItemListeners.current.set(callback, key);
      return () => listItemListeners.current.delete(callback);
    }, []);

    const retrieveSync = useDebouncedCallback(
      (query: state.SetArg<Query, Query | {}>, options: AsyncListOptions = {}) =>
        void retrieveAsync(query, options),
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
  extends Pick<UseListReturn<core.Shape, K, E>, "subscribe" | "getItem"> {
  key: K;
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
