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

import { type FetchOptions, type Params } from "@/flux/params";
import {
  errorResult,
  nullClientResult,
  pendingResult,
  type Result,
  successResult,
} from "@/flux/result";
import { type CreateRetrieveArgs } from "@/flux/retrieve";
import { type Sync } from "@/flux/sync";
import { useMountSynchronizers } from "@/flux/useMountSynchronizers";
import {
  useCombinedStateAndRef,
  useDebouncedCallback,
  useInitializerRef,
  useSyncedRef,
} from "@/hooks";
import { state } from "@/state";
import { Synnax as PSynnax } from "@/synnax";

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
> {
  /** Parameters for the retrieve operation */
  params: Partial<RetrieveParams>;
  /** The key of the item to retrieve */
  key: K;
  /** The Synnax client instance */
  client: Synnax;
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
> extends Omit<CreateRetrieveArgs<RetrieveParams, E[]>, "listeners"> {
  /** Function to retrieve a single item by key for lazy loading */
  retrieveByKey: (args: RetrieveByKeyArgs<RetrieveParams, K>) => Promise<E | undefined>;
  /** Optional listeners for real-time list updates */
  listeners?: ListListenerConfig<RetrieveParams, K, E>[];
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
  /** Debounce time for retrieve operations */
  retrieveDebounce?: CrudeTimeSpan;
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

/**
 * Extra arguments passed to list listener handlers.
 *
 * @template RetrieveParams The type of parameters for the retrieve operation
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
interface ListListenerExtraArgs<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  changed: MultiSeries;
  /** The current retrieve parameters */
  params: RetrieveParams;
  /** The Synnax client instance */
  client: Synnax;
  /** Function to update a specific item in the list */
  onChange: (key: K, e: state.SetArg<E | null>) => void;
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
export interface ListListenerConfig<
  RetrieveParams extends Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  /** The channel to listen to for real-time updates */
  channel: string;
  /** The function to call when a new value is received from the channel */
  onChange: Sync.ListenerHandler<
    MultiSeries,
    ListListenerExtraArgs<RetrieveParams, K, E>
  >;
}

/**
 * Internal reference object for managing list listeners.
 * @internal
 */
interface ListenersRef<K extends record.Key> {
  /** Whether listeners are currently mounted */
  mounted: boolean;
  /** Map of listener callbacks to their associated keys */
  listeners: Map<() => void, K>;
}

/** Default filter function that accepts all items */
const defaultFilter = () => true;
/** Default debounce time for retrieve operations */
const DEFAULT_RETRIEVE_DEBOUNCE = TimeSpan.milliseconds(100);

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
      pendingResult<K[]>(name, "retrieving", null, false),
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
          if (client == null)
            return setResult((p) =>
              nullClientResult<K[]>(name, "retrieve", p.listenersMounted),
            );
          setResult((p) => pendingResult(name, "retrieving", p.data, false));

          // If we're in replace mode, we're 'resetting' the infinite scroll position
          // of the query, so we start from the top again.
          if (mode === "replace") hasMoreRef.current = true;
          else if (mode === "append" && !hasMoreRef.current)
            return setResult((p) =>
              successResult(name, "retrieved", p.data ?? [], p.listenersMounted),
            );

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
            return setResult((p) =>
              successResult(name, "retrieved", p.data ?? [], p.listenersMounted),
            );

          value.forEach((v) => dataRef.current.set(v.key, v));

          mountSynchronizers({
            onOpen: () => {
              setResult((p) => ({ ...p, listenersMounted: true }));
            },
            listeners: listeners?.map((l) => ({
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
                        const prev = dataRef.current.get(k) ?? null;
                        if (prev != null && !filterRef.current(prev)) return;
                        const res = state.executeSetter(setter, prev);
                        if (res == null) return;
                        if (prev == null)
                          setResult((p) => {
                            if (p.data == null) return p;
                            return { ...p, data: [...p.data, k] };
                          });
                        dataRef.current.set(k, { ...res });
                        notifyListeners(k);
                      },
                    });
                  } catch (error) {
                    if (signal?.aborted) return;
                    setResult((p) =>
                      errorResult<K[]>(name, "retrieve", error, p.listenersMounted),
                    );
                  }
                })(),
            })),
          });
          return setResult((prev) => {
            if (mode === "replace" || prev.data == null)
              return successResult(name, "retrieved", keys, prev.listenersMounted);
            const keysSet = new Set(keys);
            return successResult(
              name,
              "retrieved",
              [...prev.data.filter((k) => !keysSet.has(k)), ...keys],
              prev.listenersMounted,
            );
          });
        } catch (error) {
          if (signal?.aborted) return;
          setResult((p) =>
            errorResult<K[]>(name, "retrieve", error, p.listenersMounted),
          );
        }
      },
      [client, name, mountSynchronizers, filterRef],
    );

    const retrieveSingle = useCallback(
      (key: K, options: FetchOptions = {}) => {
        const { signal } = options;
        void (async () => {
          try {
            if (client == null || primitive.isZero(key)) return;
            const item = await retrieveByKey({
              client,
              key,
              params: paramsRef.current ?? {},
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
            setResult((p) =>
              errorResult<K[]>(name, "retrieve", error, p.listenersMounted),
            );
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
