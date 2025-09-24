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
  type Destructor,
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

/**
 * Function interface for getting items from a list by key(s).
 *
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
interface GetItem<K extends record.Key, E extends record.Keyed<K>> {
  /** Get a single item by key, returns undefined if not found */
  (key: K): E | undefined;
  /** Get multiple items by an array of keys */
  (keys: K[]): E[];
}

/**
 * Options for async list operations.
 */
interface AsyncListOptions extends core.FetchOptions {
  /**
   * How to modify the list when new data is retrieved. In append mode, new entries
   * will be added to the end of the list. In replace mode, the list will be replaced
   * with the new data.
   */
  mode?: "append" | "replace";
}

/**
 * Return type for the list hook, providing comprehensive list management utilities.
 *
 * @template Query The type of parameters for the retrieve operation
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
export type UseListReturn<
  Query extends core.Shape,
  K extends record.Key,
  E extends record.Keyed<K>,
> = Omit<Result<K[]>, "data"> & {
  /** Function to trigger a list retrieval operation (fire-and-forget) */
  retrieve: (
    query: state.SetArg<Query, Partial<Query>>,
    options?: AsyncListOptions,
  ) => void;
  /** Function to trigger a list retrieval operation and await the result */
  retrieveAsync: (
    query: state.SetArg<Query, Partial<Query>>,
    options?: AsyncListOptions,
  ) => Promise<void>;
  /** Array of keys for the items currently in the list */
  data: K[];
  /** Function to get items by key, with automatic lazy loading */
  getItem: GetItem<K, E>;
  /** Function to subscribe to changes for specific items */
  subscribe: (callback: () => void, key: K) => Destructor;
};

/**
 * Arguments for retrieving a single item by key.
 *
 * @template Query The type of parameters for the retrieve operation
 * @template K The type of the key (must be a record key)
 */
export interface RetrieveByKeyParams<
  Query extends core.Shape,
  K extends record.Key,
  Store extends flux.Store,
> extends Omit<RetrieveParams<Query, Store>, "query"> {
  /** Parameters for the retrieve operation */
  query: Partial<Query>;
  /** The key of the item to retrieve */
  key: K;
}

export interface RetrieveCachedParams<
  Query extends core.Shape,
  Store extends flux.Store,
> {
  query: Partial<Query>;
  store: Store;
}

/**
 * Configuration arguments for creating a list query.
 *
 * @template Query The type of parameters for the retrieve operation
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
export interface CreateListParams<
  Query extends core.Shape,
  K extends record.Key,
  E extends record.Keyed<K>,
  Store extends flux.Store,
> extends Omit<CreateRetrieveParams<Query, E[], Store>, "mountListeners"> {
  /** Function to sort the list */
  sort?: compare.Comparator<E>;
  /** Function to retrieve a single item by key for lazy loading */
  retrieveByKey: (args: RetrieveByKeyParams<Query, K, Store>) => Promise<E | undefined>;
  /** Function that allows  */
  retrieveCached?: (args: RetrieveCachedParams<Query, Store>) => E[];
  /** Function to mount listeners for the list */
  mountListeners?: (
    args: ListMountListenersParams<Query, K, E, Store>,
  ) => Destructor | Destructor[];
}

/**
 * Arguments for using a list hook.
 *
 * @template Query The type of parameters for the retrieve operation
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
export interface UseListParams<
  Query extends core.Shape,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  /** Initial parameters for the list query */
  initialQuery?: Query;
  /** Optional filter function to apply to items */
  filter?: (item: E) => boolean;
  /** Optional function to sort the list */
  sort?: compare.Comparator<E>;
  /** Debounce time for retrieve operations */
  retrieveDebounce?: CrudeTimeSpan;
  /** Whether to retreve initial list results from the cache */
  useCachedList?: boolean;
}

/**
 * List hook function signature.
 *
 * @template Query The type of parameters for the retrieve operation
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
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

/**
 * Configuration for a list listener that handles real-time updates.
 *
 * @template Query The type of parameters for the retrieve operation
 * @template Key The type of the key (must be a record key)
 * @template Data The type of the entity (must be keyed by K)
 */
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
  onChange: (
    key: Key,
    e: state.SetArg<Data | null>,
    opts?: ListenerOnChangeOptions,
  ) => void;
  onDelete: (key: Key) => void;
}

/** Default filter function that accepts all items */
const defaultFilter = () => true;
/** Default debounce time for retrieve operations */
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

/**
 * Creates a list query hook that provides comprehensive list management with real-time updates.
 *
 * This function creates a React hook that handles:
 * - List data retrieval with loading states
 * - Individual item lazy loading
 * - Real-time synchronization with server state
 * - Pagination and infinite scrolling support
 * - Item filtering and search
 * - Optimistic updates and error handling
 * - Subscribe to individual item changes
 *
 * @template P The type of parameters for the list query
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 * @param config Configuration object with list retrieval functions and settings
 * @returns A React hook for managing the list
 *
 * @example
 * ```typescript
 * interface UserListParams extends Params {
 *   department?: string;
 *   searchTerm?: string;
 *   offset?: number;
 *   limit?: number;
 * }
 *
 * interface User {
 *   id: number;
 *   name: string;
 *   email: string;
 *   department: string;
 * }
 *
 * const useUserList = createList<UserListParams, number, User>({
 *   name: "users",
 *   retrieve: async ({ query, client }) => {
 *     return await client.users.list(query);
 *   },
 *   retrieveByKey: async ({ key, client }) => {
 *     return await client.users.get(key);
 *   },
 *   listeners: [
 *     {
 *       channel: "user_updates",
 *       onChange: ({ changed, onChange, onDelete }) => {
 *         const updates = changed.get("user_updates");
 *         updates.forEach(update => {
 *           if (update.deleted) {
 *             onDelete(update.id);
 *           } else {
 *             onChange(update.id, update);
 *           }
 *         });
 *       }
 *     }
 *   ]
 * });
 *
 * // Usage in component
 * const { data, getItem, retrieve, variant } = useUserList({
 *   initialParams: { department: "engineering" },
 *   filter: (user) => user.name.includes("John")
 * });
 *
 * // Get individual user (lazy loaded if not in cache)
 * const user = getItem(123);
 *
 * // Load more users
 * retrieve({ offset: data.length, limit: 10 }, { mode: "append" });
 * ```
 */
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
          onChange: (k, setter, opts = {}) => {
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
          if (!storeListenersMountedRef.current) syncListeners();
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

/**
 * Arguments for the useListItem hook.
 *
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
export interface UseListItemArgs<K extends record.Key, E extends record.Keyed<K>>
  extends Pick<UseListReturn<core.Shape, K, E>, "subscribe" | "getItem"> {
  /** The key of the item to retrieve and subscribe to */
  key: K;
}

/**
 * Hook for subscribing to and retrieving individual items from a list.
 *
 * This hook provides a way to efficiently track individual items from a list
 * with automatic re-rendering when the item changes. It uses React's
 * useSyncExternalStore to provide optimal performance.
 *
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 * @param args Configuration object with key and list utilities
 * @returns The current item data, or undefined if not found
 *
 * @example
 * ```typescript
 * const userList = useUserList();
 * const user = useListItem({
 *   key: userId,
 *   subscribe: userList.subscribe,
 *   getItem: userList.getItem
 * });
 *
 * // Component will re-render when this specific user changes
 * if (user) {
 *   return <div>{user.name} - {user.email}</div>;
 * }
 * ```
 */
export const useListItem = <K extends record.Key, E extends record.Keyed<K>>({
  key,
  subscribe,
  getItem,
}: UseListItemArgs<K, E>) =>
  useSyncExternalStore(
    useCallback((callback) => subscribe(callback, key), [subscribe, key]),
    useCallback(() => getItem(key), [getItem, key]),
  );
