// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import {
  compare,
  type CrudeTimeSpan,
  type Destructor,
  type MultiSeries,
  primitive,
  type record,
  TimeSpan,
} from "@synnaxlabs/x";
import { type RefObject, useCallback, useRef, useSyncExternalStore } from "react";

import { type flux } from "@/flux/aether";
import { type FetchOptions, type Params } from "@/flux/core/params";
import { useStore } from "@/flux/Provider";
import {
  errorResult,
  nullClientResult,
  pendingResult,
  type Result,
  successResult,
} from "@/flux/result";
import { type CreateRetrieveArgs, type MountListenersArgs } from "@/flux/retrieve";
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
interface AsyncListOptions extends FetchOptions {
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
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
export type UseListReturn<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> = Omit<Result<K[]>, "data"> & {
  /** Function to trigger a list retrieval operation (fire-and-forget) */
  retrieve: (
    params: state.SetArg<RetrieveParams, Partial<RetrieveParams>>,
    options?: AsyncListOptions,
  ) => void;
  /** Function to trigger a list retrieval operation and await the result */
  retrieveAsync: (
    params: state.SetArg<RetrieveParams, Partial<RetrieveParams>>,
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
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template K The type of the key (must be a record key)
 */
export interface RetrieveByKeyArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  ScopedStore extends flux.Store,
> {
  /** Parameters for the retrieve operation */
  params: Partial<RetrieveParams>;
  /** The key of the item to retrieve */
  key: K;
  /** The Synnax client instance */
  client: Client;
  /** The store instance */
  store: ScopedStore;
}

export interface RetrieveCachedArgs<
  RetrieveParams extends Params,
  ScopedStore extends flux.Store,
> {
  params: Partial<RetrieveParams>;
  store: ScopedStore;
}

/**
 * Configuration arguments for creating a list query.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
export interface CreateListArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
  ScopedStore extends flux.Store,
> extends Omit<CreateRetrieveArgs<RetrieveParams, E[], ScopedStore>, "mountListeners"> {
  /** Function to sort the list */
  sort?: compare.Comparator<E>;
  /** Function to retrieve a single item by key for lazy loading */
  retrieveByKey: (
    args: RetrieveByKeyArgs<RetrieveParams, K, ScopedStore>,
  ) => Promise<E | undefined>;
  /** Function that allows  */
  retrieveCached?: (args: RetrieveCachedArgs<RetrieveParams, ScopedStore>) => E[];
  /** Function to mount listeners for the list */
  mountListeners?: (
    args: ListMountListenersArgs<RetrieveParams, K, E, ScopedStore>,
  ) => Destructor | Destructor[];
}

/**
 * Arguments for using a list hook.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
export interface UseListArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  /** Initial parameters for the list query */
  initialParams?: RetrieveParams;
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
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
export interface UseList<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  (args?: UseListArgs<RetrieveParams, K, E>): UseListReturn<RetrieveParams, K, E>;
}

type ListChangeMode = "prepend" | "append" | "replace";

interface ListenerOnChangeOptions {
  mode?: ListChangeMode;
}

/**
 * Extra arguments passed to list listener handlers.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
export interface ListListenerExtraArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  changed: MultiSeries;
  /** The current retrieve parameters */
  params: RetrieveParams;
  /** The Synnax client instance */
  client: Client;
  /** Function to update a specific item in the list */
  onChange: (key: K, e: state.SetArg<E | null>, opts?: ListenerOnChangeOptions) => void;
  /** Function to remove an item from the list */
  onDelete: (key: K) => void;
}

/**
 * Configuration for a list listener that handles real-time updates.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
export interface ListMountListenersArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
  ScopedStore extends flux.Store,
> extends Omit<
    MountListenersArgs<ScopedStore, RetrieveParams, E[]>,
    "onChange" | "params"
  > {
  params: Partial<RetrieveParams>;
  onChange: (key: K, e: state.SetArg<E | null>, opts?: ListenerOnChangeOptions) => void;
  onDelete: (key: K) => void;
}

/** Default filter function that accepts all items */
const defaultFilter = () => true;
/** Default debounce time for retrieve operations */
const DEFAULT_RETRIEVE_DEBOUNCE = TimeSpan.milliseconds(100);

interface GetInitialDataArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
  ScopedStore extends flux.Store,
> {
  retrieveCached: CreateListArgs<RetrieveParams, K, E, ScopedStore>["retrieveCached"];
  paramsRef: RefObject<RetrieveParams | null>;
  filterRef: RefObject<((item: E) => boolean) | undefined>;
  sortRef: RefObject<compare.Comparator<E> | undefined>;
  dataRef: RefObject<Map<K, E | null>>;
  store: ScopedStore;
  useCachedList: boolean;
}

const getInitialData = <
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
  ScopedStore extends flux.Store,
>({
  retrieveCached,
  paramsRef,
  filterRef,
  sortRef,
  dataRef,
  store,
  useCachedList,
}: GetInitialDataArgs<RetrieveParams, K, E, ScopedStore>) => {
  if (retrieveCached == null || !useCachedList) return undefined;
  let cached = retrieveCached({ params: paramsRef.current ?? {}, store });
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
 *   retrieve: async ({ params, client }) => {
 *     return await client.users.list(params);
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
    P extends Params,
    K extends record.Key,
    E extends record.Keyed<K>,
    ScopedStore extends flux.Store = {},
  >({
    name,
    mountListeners,
    retrieve,
    retrieveByKey,
    retrieveCached,
    sort: defaultSort,
  }: CreateListArgs<P, K, E, ScopedStore>): UseList<P, K, E> =>
  (args: UseListArgs<P, K, E> = {}) => {
    const {
      filter = defaultFilter,
      sort,
      initialParams,
      retrieveDebounce = DEFAULT_RETRIEVE_DEBOUNCE,
      useCachedList = true,
    } = args;
    const filterRef = useSyncedRef(filter);
    const sortRef = useSyncedRef(sort ?? defaultSort);
    const client = Synnax.use();
    const dataRef = useRef<Map<K, E | null>>(new Map());
    const listItemListeners = useInitializerRef<Map<() => void, K>>(() => new Map());
    const store = useStore<ScopedStore>();
    const paramsRef = useRef<P | null>(initialParams ?? null);
    const [result, setResult, resultRef] = useCombinedStateAndRef<Result<K[]>>(() =>
      pendingResult<K[]>(
        name,
        "retrieving",
        getInitialData({
          retrieveCached,
          paramsRef,
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
      (changed: K) =>
        listItemListeners.current.forEach((key, notify) => {
          if (key === changed) notify();
        }),
      [listItemListeners.current],
    );

    const updateSortedData = useCallback(
      (keys: K[]) => {
        if (sortRef.current == null) return keys;

        const allItems = keys
          .map((key) => dataRef.current.get(key))
          .filter((item): item is E => item != null);

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
          params: paramsRef.current ?? {},
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
              let newData: K[];
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
          else if (mode === "append" && !hasMoreRef.current)
            return setResult((p) => successResult(name, "retrieved", p.data ?? []));

          let value = await retrieve({ client, params, store });
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
            return setResult((p) => successResult(name, "retrieved", p.data ?? []));

          value.forEach((v) => dataRef.current.set(v.key, v));

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
          if (signal?.aborted) return;
          setResult(errorResult<K[]>(name, "retrieve", error));
        }
      },
      [client, name, store, filterRef, syncListeners],
    );

    const retrieveSingle = useCallback(
      (key: K, options: FetchOptions = {}) => {
        const { signal } = options;
        void (async () => {
          if (!storeListenersMountedRef.current) syncListeners();
          try {
            if (client == null || primitive.isZero(key)) return;
            const item = await retrieveByKey({
              client,
              key,
              params: paramsRef.current ?? {},
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
            setResult(errorResult<K[]>(name, "retrieve", error));
          }
        })();
      },
      [retrieveByKey, client, notifyListeners, syncListeners],
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
      listItemListeners.current.set(callback, key);
      return () => listItemListeners.current.delete(callback);
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

/**
 * Arguments for the useListItem hook.
 *
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
export interface UseListItemArgs<K extends record.Key, E extends record.Keyed<K>>
  extends Pick<UseListReturn<Params, K, E>, "subscribe" | "getItem"> {
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
