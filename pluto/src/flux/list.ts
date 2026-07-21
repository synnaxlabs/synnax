// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cache } from "@synnaxlabs/client";
import {
  compare,
  type CrudeTimeSpan,
  type destructor,
  primitive,
  type record,
  state,
  TimeSpan,
} from "@synnaxlabs/x";
import { useCallback, useEffect, useRef, useSyncExternalStore } from "react";

import {
  errorResult,
  loadingResult,
  nullClientResult,
  type Result,
  successResult,
} from "@/flux/result";
import { type CreateRetrieveParams, type RetrieveParams } from "@/flux/retrieve";
import {
  useCombinedStateAndRef,
  useDebouncedCallback,
  useInitializerRef,
  useSyncedRef,
} from "@/hooks";
import { Synnax } from "@/synnax";

export interface GetItem<K extends record.Key, E extends record.Keyed<K>> {
  (key: K): E | undefined;
  (keys: K[]): E[];
}

export interface AsyncListOptions extends cache.FetchOptions {
  mode?: "append" | "replace";
}

export type UseListReturn<
  Query extends cache.Query,
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
  Query extends cache.Query,
  K extends record.Key,
> extends Omit<RetrieveParams<Query>, "query"> {
  query: Partial<Query>;
  key: K;
}

export interface CreateListParams<
  Query extends cache.Query,
  K extends record.Key,
  E extends record.Keyed<K>,
> extends CreateRetrieveParams<Query, E[]> {
  sort?: compare.Comparator<E>;
  retrieveByKey: (params: RetrieveByKeyParams<Query, K>) => Promise<E | undefined>;
  /**
   * Live updates for items fetched through retrieveByKey. Page members get
   * their updates from `subscribe`; this covers lookups outside any page.
   */
  subscribeByKey?: (
    params: RetrieveByKeyParams<Query, K>,
    handler: cache.ChangeHandler<E>,
  ) => destructor.Destructor;
}

export interface UseListParams<
  Query extends cache.Query,
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
  Query extends cache.Query,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  (params?: UseListParams<Query, K, E>): UseListReturn<Query, K, E>;
}

const defaultFilter = () => true;
const DEFAULT_RETRIEVE_DEBOUNCE = TimeSpan.milliseconds(100);

/** One retrieved page: its query, answer keys, and live subscription. */
interface Page<K extends record.Key> {
  hash: string;
  keys: K[];
  unsubscribe: destructor.Destructor;
}

export const createList =
  <Query extends cache.Query, Key extends record.Key, Data extends record.Keyed<Key>>({
    name,
    retrieve,
    retrieveByKey,
    subscribe: subscribeToQuery,
    subscribeByKey,
    getCached,
    sort: defaultSort,
  }: CreateListParams<Query, Key, Data>): UseList<Query, Key, Data> =>
  (params: UseListParams<Query, Key, Data> = {}) => {
    const {
      filter = defaultFilter,
      sort,
      initialQuery,
      retrieveDebounce = DEFAULT_RETRIEVE_DEBOUNCE,
      useCachedList = true,
    } = params;
    const filterRef = useSyncedRef(filter);
    const sortRef = useSyncedRef(sort ?? defaultSort);
    const client = Synnax.use();
    const dataRef = useRef<Map<Key, Data | null>>(new Map());
    const listItemListeners = useInitializerRef<Map<() => void, Key>>(() => new Map());
    const queryRef = useRef<Query | null>(initialQuery ?? null);
    const pagesRef = useRef<Page<Key>[]>([]);
    const itemSubsRef = useInitializerRef<Map<Key, destructor.Destructor>>(
      () => new Map(),
    );

    const notifyListeners = useCallback(
      (changed: Key) =>
        listItemListeners.current.forEach((key, notify) => {
          if (key === changed) notify();
        }),
      [listItemListeners.current],
    );

    const getInitialData = (): Key[] | undefined => {
      if (!useCachedList || getCached == null) return undefined;
      const query = queryRef.current ?? ({} as Query);
      if (client == null) return undefined;
      const cached = getCached({
        client,
        query,
      });
      if (cached?.variant !== "changed") return undefined;
      let items = cached.data.filter(filterRef.current);
      if (sortRef.current != null) items = [...items].sort(sortRef.current);
      if (items.length === 0) return undefined;
      items.forEach((v) => dataRef.current.set(v.key, v));
      return items.map((v) => v.key);
    };

    const [result, setResult, resultRef] = useCombinedStateAndRef<Result<Key[]>>(() =>
      loadingResult<Key[]>(`retrieving ${name}`, getInitialData()),
    );
    const hasMoreRef = useRef(true);

    const recomputeKeys = useCallback(() => {
      const seen = new Set<Key>();
      const keys: Key[] = [];
      pagesRef.current.forEach((page) =>
        page.keys.forEach((k) => {
          if (seen.has(k)) return;
          seen.add(k);
          keys.push(k);
        }),
      );
      if (sortRef.current != null) {
        const items = keys
          .map((k) => dataRef.current.get(k))
          .filter((item): item is Data => item != null);
        items.sort(sortRef.current);
        return items.map((item) => item.key);
      }
      return keys;
    }, []);

    /** Applies a page answer: updates items, notifies, recomputes keys. */
    const applyPageAnswer = useCallback(
      (page: Page<Key>, items: Data[]) => {
        const filtered = items.filter(filterRef.current);
        filtered.forEach((item) => {
          const prev = dataRef.current.get(item.key);
          if (prev === item) return;
          dataRef.current.set(item.key, item);
          notifyListeners(item.key);
        });
        const nextKeys = new Set(filtered.map((item) => item.key));
        const dropped = page.keys.filter((k) => !nextKeys.has(k));
        page.keys = filtered.map((item) => item.key);
        dropped.forEach((k) => {
          if (pagesRef.current.some((p) => p.keys.includes(k))) return;
          dataRef.current.set(k, null);
          notifyListeners(k);
        });
        setResult((p) => {
          if (p.data == null) return p;
          const next = recomputeKeys();
          if (compare.primitiveArrays(p.data, next) === compare.EQUAL) return p;
          return { ...p, data: next };
        });
      },
      [notifyListeners, recomputeKeys, setResult],
    );

    const clearPages = useCallback(() => {
      pagesRef.current.forEach((page) => page.unsubscribe());
      pagesRef.current = [];
    }, []);
    useEffect(() => clearPages, [clearPages]);

    const handleItemChange = useCallback(
      (key: Key, cached: cache.Cached<Data> | undefined) => {
        if (cached == null) return;
        if (cached.variant === "changed")
          dataRef.current.set(key, filterRef.current(cached.data) ? cached.data : null);
        else dataRef.current.set(key, null);
        notifyListeners(key);
      },
      [notifyListeners],
    );

    const openItemSub = useCallback(
      (key: Key): void => {
        if (subscribeByKey == null || client == null) return;
        if (itemSubsRef.current.has(key)) return;
        itemSubsRef.current.set(
          key,
          subscribeByKey(
            {
              client,
              key,
              query: queryRef.current ?? {},
            },
            (cached) => handleItemChange(key, cached),
          ),
        );
      },
      [client, handleItemChange],
    );

    const clearItemSubs = useCallback(() => {
      itemSubsRef.current.forEach((unsub) => unsub());
      itemSubsRef.current.clear();
    }, []);
    useEffect(() => clearItemSubs, [clearItemSubs]);

    const openPage = useCallback(
      (query: Query, keys: Key[]): void => {
        const hash = cache.hashQuery(query);
        const existing = pagesRef.current.find((p) => p.hash === hash);
        if (existing != null) {
          existing.keys = keys;
          return;
        }
        const page: Page<Key> = { hash, keys, unsubscribe: () => {} };
        if (subscribeToQuery != null && client != null)
          page.unsubscribe = subscribeToQuery(
            {
              client,
              query,
            },
            (cached: cache.Cached<Data[]> | undefined) => {
              if (cached?.variant !== "changed") return;
              applyPageAnswer(page, cached.data);
            },
          );
        pagesRef.current.push(page);
      },
      [client, applyPageAnswer],
    );

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

          let value = await retrieve({
            client,
            query,
          });
          if (signal?.aborted) return;
          value = value.filter(filterRef.current);
          if (sortRef.current != null) value = [...value].sort(sortRef.current);

          if (value.length === 0) hasMoreRef.current = false;
          const keys = value.map((v) => v.key);

          if (mode === "replace") clearPages();
          value.forEach((v) => dataRef.current.set(v.key, v));
          openPage(query, keys);
          keys.forEach(notifyListeners);

          // If we've already retrieved the initial data, and it's the same as the
          // data we just retrieved, then don't notify listeners.
          if (
            resultRef.current.data != null &&
            compare.primitiveArrays(resultRef.current.data, keys) === compare.EQUAL
          )
            return setResult((p) => successResult(`retrieved ${name}`, p.data ?? []));

          return setResult(() => successResult(`retrieved ${name}`, recomputeKeys()));
        } catch (error) {
          if (signal?.aborted) return;
          setResult(errorResult(`retrieve ${name}`, error));
        }
      },
      [client, name, filterRef, clearPages, openPage, recomputeKeys],
    );

    // When the client is swapped, an active hook must re-point itself at the
    // new client's cache. A non-null query means a list retrieve ran: refetch
    // (retrieveAsync re-subscribes as it goes). Item subscriptions re-open
    // against the new client without refetching.
    const retrieveAsyncRef = useSyncedRef(retrieveAsync);
    const firstMountRef = useRef(true);
    useEffect(() => {
      if (firstMountRef.current) {
        firstMountRef.current = false;
        return;
      }
      clearPages();
      const itemKeys = [...itemSubsRef.current.keys()];
      clearItemSubs();
      itemKeys.forEach(openItemSub);
      const query = queryRef.current;
      if (query != null) void retrieveAsyncRef.current(query);
    }, [client]);

    const retrieveSingle = useCallback(
      (key: Key, options: cache.FetchOptions = {}) => {
        const { signal } = options;
        void (async () => {
          try {
            if (client == null || primitive.isZero(key)) return;
            const item = await retrieveByKey({
              client,
              key,
              query: queryRef.current ?? {},
            });
            if (signal?.aborted || item == null) return;
            openItemSub(key);
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
      [retrieveByKey, client, notifyListeners, openItemSub],
    );

    const getItem = useCallback(
      ((key?: Key | Key[]) => {
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
      [retrieveSingle],
    );

    const subscribe = useCallback((callback: () => void, key?: Key) => {
      if (key == null) return () => {};
      listItemListeners.current.set(callback, key);
      return () => listItemListeners.current.delete(callback);
    }, []);

    const retrieveSync = useDebouncedCallback(
      (query: state.SetArg<Query, Query | {}>, options: AsyncListOptions = {}) =>
        void retrieveAsync(query, options),
      retrieveDebounce,
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

export interface UseListItemParams<
  K extends record.Key,
  E extends record.Keyed<K>,
> extends Pick<UseListReturn<cache.Query, K, E>, "subscribe" | "getItem"> {
  key: K;
}

export const useListItem = <K extends record.Key, E extends record.Keyed<K>>({
  key,
  subscribe,
  getItem,
}: UseListItemParams<K, E>) =>
  useSyncExternalStore(
    useCallback((callback) => subscribe(callback, key), [subscribe, key]),
    useCallback(() => getItem(key), [getItem, key]),
    () => undefined,
  );
