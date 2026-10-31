// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query } from "@synnaxlabs/client";
import {
  useCombinedStateAndRef,
  useDebouncedCallback,
  useInitializerRef,
  useSyncedRef,
} from "@synnaxlabs/lyra/hooks";
import {
  compare,
  type CrudeTimeSpan,
  type destructor,
  primitive,
  type record,
  state,
  TimeSpan,
} from "@synnaxlabs/x";
import { useEffect, useRef, useSyncExternalStore } from "react";

import {
  errorResult,
  loadingResult,
  nullClientResult,
  type Result,
  successResult,
} from "@/flux/result";
import { type CreateRetrieveParams, type RetrieveParams } from "@/flux/retrieve";
import { Synnax } from "@/synnax";

// Bound at module scope: hooks bind `query` to the caller's params object.
const { Deleted, isLive } = query;

/** Reads one entry, or many at once, out of a list query's results. */
export interface GetItem<K extends record.Key, E extends record.Keyed<K>> {
  (key: K): E | undefined;
  (keys: K[]): E[];
}

/** Options for a list retrieval. Append for paging, replace for a new search. */
export interface AsyncListOptions extends query.FetchOptions {
  mode?: "append" | "replace";
}

/** Return value for a list hook. Spread it into a {@link List.Frame}. */
export type UseListReturn<
  Query extends query.Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> = Omit<Result<K[]>, "data"> & {
  /**
   * Retrieves once the debounce ends. Query changes made while it waits apply in order,
   * then one append runs after them. A second append made while one waits does nothing.
   */
  retrieve: (
    query: state.SetArg<Query, Partial<Query>>,
    options?: AsyncListOptions,
  ) => void;
  /**
   * Retrieves now. An append waits for the newest replace to answer, and does nothing
   * when that replace fails.
   */
  retrieveAsync: (
    query: state.SetArg<Query, Partial<Query>>,
    options?: AsyncListOptions,
  ) => Promise<void>;
  data: K[];
  /**
   * Whether the list has an answer. An unanswered list has the same empty data as a
   * genuinely empty one, so anything that speaks for the absence of items (empty
   * content, an empty-state action) must wait for this.
   */
  answered: boolean;
  getItem: GetItem<K, E>;
  subscribe: (callback: () => void, key: K) => destructor.Destructor;
};

export interface RetrieveByKeyParams<
  Query extends query.Params,
  K extends record.Key,
> extends Omit<RetrieveParams<Query>, "query"> {
  query: Partial<Query>;
  key: K;
}

export interface CreateListParams<
  Query extends query.Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> extends CreateRetrieveParams<Query, E[]> {
  sort?: compare.Comparator<E>;
  retrieveByKey: (params: RetrieveByKeyParams<Query, K>) => Promise<E | undefined>;
  /**
   * Live updates for an item outside any page: a lookup through retrieveByKey, or a
   * read item that a page let go. Page members get their updates from `onChange`.
   */
  onChangeByKey?: (
    params: RetrieveByKeyParams<Query, K>,
    handler: query.ChangeHandler<E>,
  ) => destructor.Destructor;
}

export interface UseListParams<
  Query extends query.Params,
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
  Query extends query.Params,
  K extends record.Key,
  E extends record.Keyed<K>,
> {
  (params?: UseListParams<Query, K, E>): UseListReturn<Query, K, E>;
}

const defaultFilter = () => true;
const DEFAULT_RETRIEVE_DEBOUNCE = TimeSpan.milliseconds(100);

/** A retrieve that waits for the debounce to end. */
interface Retrieve<Query extends query.Params> {
  query: state.SetArg<Query, Query | {}>;
  options: AsyncListOptions;
}

/** The retrieves a list runs when its debounce ends: the replace, then the append. */
interface PendingRetrieves<Query extends query.Params> {
  replace?: Retrieve<Query>;
  append?: Retrieve<Query>;
}

/** @returns a setter that applies first, then second. */
const chainSetters =
  <Query extends query.Params>(
    first: state.SetArg<Query, Query | {}>,
    second: state.SetArg<Query, Query | {}>,
  ): state.SetFunc<Query, Query | {}> =>
  (prev) =>
    state.executeSetter(second, state.executeSetter(first, prev));

/** One retrieved page: its query, answer keys, and live subscription. */
interface Page<K extends record.Key> {
  hash: string;
  keys: K[];
  unsubscribe: destructor.Destructor;
  /** Set when the client cache answered the page. The next retrieve replaces it. */
  cached?: boolean;
}

export const createList = <
  Query extends query.Params,
  Key extends record.Key,
  Data extends record.Keyed<Key>,
>({
  name,
  retrieve,
  retrieveByKey,
  onChange: subscribeToQuery,
  onChangeByKey,
  getCached,
  sort: defaultSort,
  normalizeQuery,
}: CreateListParams<Query, Key, Data>): UseList<Query, Key, Data> => {
  const normalized = (q: Query): Query => normalizeQuery?.(q) ?? q;
  const useList = (
    params: UseListParams<Query, Key, Data> = {},
  ): UseListReturn<Query, Key, Data> => {
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
    const listItemListenersRef = useInitializerRef<Map<Key, Set<() => void>>>(
      () => new Map(),
    );
    const queryRef = useRef<Query | null>(
      initialQuery != null ? normalized(initialQuery) : null,
    );
    const pagesRef = useRef<Page<Key>[]>([]);
    const itemSubsRef = useInitializerRef<Map<Key, destructor.Destructor>>(
      () => new Map(),
    );

    const notifyListeners = (changed: Key) =>
      listItemListenersRef.current.get(changed)?.forEach((notify) => notify());

    const readCache = (query: Query): Key[] | undefined => {
      if (!useCachedList || getCached == null || client == null) return undefined;
      const cached = getCached({ client, query });
      if (!isLive(cached)) return undefined;
      let items = cached.filter(filterRef.current);
      if (sortRef.current != null) items = [...items].sort(sortRef.current);
      items.forEach((v) => dataRef.current.set(v.key, v));
      return items.map((v) => v.key);
    };

    const [result, setResult, resultRef] = useCombinedStateAndRef<Result<Key[]>>(() =>
      loadingResult<Key[]>(
        `retrieving ${name}`,
        readCache(queryRef.current ?? normalized({} as Query)),
      ),
    );
    const hasMoreRef = useRef(true);
    const pendingRef = useRef<PendingRetrieves<Query> | null>(null);
    /** The newest replace. It resolves to whether its answer reached the list. */
    const latestReplaceRef = useInitializerRef(() => Promise.resolve(true));

    const recomputeKeys = () => {
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
    };

    const handleItemChange = (key: Key, cached: query.Cached<Data> | undefined) => {
      if (cached === undefined) return;
      if (Deleted.matches<Data>(cached)) dataRef.current.set(key, null);
      else dataRef.current.set(key, filterRef.current(cached) ? cached : null);
      notifyListeners(key);
    };

    // Declared above its callers. The compiler caches neither function when a hook call
    // sits between a caller and a callee declared below it.
    const openItemSub = (key: Key): void => {
      if (onChangeByKey == null || client == null) return;
      if (itemSubsRef.current.has(key)) return;
      itemSubsRef.current.set(
        key,
        onChangeByKey(
          {
            client,
            key,
            query: queryRef.current ?? {},
          },
          (cached) => handleItemChange(key, cached),
        ),
      );
    };

    /** Applies a page answer: updates items, notifies, recomputes keys. */
    const applyPageAnswer = (page: Page<Key>, items: Data[]) => {
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
        // The item may only have left the query, so an item a consumer reads keeps a
        // feed of its own, which reports a deletion.
        if (itemSubsRef.current.has(k)) return;
        if (listItemListenersRef.current.has(k)) return openItemSub(k);
        dataRef.current.set(k, null);
        notifyListeners(k);
      });
      setResult((p) => {
        if (p.data == null) return p;
        const next = recomputeKeys();
        if (compare.primitiveArrays(p.data, next) === compare.EQUAL) return p;
        return { ...p, data: next };
      });
    };

    const closePages = (closes: (page: Page<Key>) => boolean) => {
      pagesRef.current = pagesRef.current.filter((page) => {
        if (!closes(page)) return true;
        page.unsubscribe();
        return false;
      });
    };
    const clearPages = () => closePages(() => true);
    useEffect(() => clearPages, [clearPages]);

    // An answer that arrives after unmount would open a feed that nothing closes.
    const mountedRef = useRef(false);
    useEffect(() => {
      mountedRef.current = true;
      return () => {
        mountedRef.current = false;
      };
    }, []);

    const clearItemSubs = () => {
      itemSubsRef.current.forEach((unsub) => unsub());
      itemSubsRef.current.clear();
    };
    useEffect(() => clearItemSubs, [clearItemSubs]);

    const closeItemSub = (key: Key) => {
      itemSubsRef.current.get(key)?.();
      itemSubsRef.current.delete(key);
    };

    /**
     * Gives each record that no page holds its own feed while a consumer reads it, such
     * as a selection, and drops the rest. Without a feed the copy goes stale.
     */
    const coverRecords = () => {
      const paged = new Set(pagesRef.current.flatMap((page) => page.keys));
      dataRef.current.forEach((item, key) => {
        if (item == null || paged.has(key) || itemSubsRef.current.has(key)) return;
        if (listItemListenersRef.current.has(key)) openItemSub(key);
        else dataRef.current.delete(key);
      });
    };

    const openPage = (q: Query, keys: Key[], cached = false): void => {
      const hash = query.hash(q);
      const existing = pagesRef.current.find((p) => p.hash === hash);
      if (existing != null) {
        existing.keys = keys;
        return;
      }
      const page: Page<Key> = { hash, keys, unsubscribe: () => {}, cached };
      if (subscribeToQuery != null && client != null)
        page.unsubscribe = subscribeToQuery(
          {
            client,
            query: q,
          },
          (answer: query.Cached<Data[]> | undefined) => {
            if (!isLive(answer)) return;
            applyPageAnswer(page, answer);
          },
        );
      pagesRef.current.push(page);
    };

    /** @returns whether the answer reached the list. */
    const retrievePage = async (
      paramsSetter: state.SetArg<Query, Query | {}>,
      options: AsyncListOptions,
    ): Promise<boolean> => {
      const { signal, mode = "replace" } = options;

      const query = normalized(
        state.executeSetter(paramsSetter, queryRef.current ?? {}),
      );
      queryRef.current = query;

      // The compiler rejects `?.`, `??`, and `?:` in a try, so the body is a closure.
      const apply = async (): Promise<boolean> => {
        if (client == null) {
          setResult(nullClientResult<Key[]>(`retrieve ${name}`));
          return false;
        }
        setResult((p) => loadingResult(`retrieving ${name}`, p.data));

        // If we're in replace mode, we're 'resetting' the infinite scroll position
        // of the query, so we start from the top again.
        if (mode === "replace") hasMoreRef.current = true;
        else if (mode === "append" && !hasMoreRef.current) {
          setResult((p) => successResult(`retrieved ${name}`, p.data ?? []));
          return true;
        }

        let value = await retrieve({
          client,
          query,
        });
        if (signal?.aborted || queryRef.current !== query || !mountedRef.current)
          return false;
        value = value.filter(filterRef.current);
        if (sortRef.current != null) value = [...value].sort(sortRef.current);

        if (value.length === 0) hasMoreRef.current = false;
        const keys = value.map((v) => v.key);

        // An answer replaces a page the cache answered in either mode.
        closePages((page) => mode === "replace" || page.cached === true);
        value.forEach((v) => dataRef.current.set(v.key, v));
        openPage(query, keys);
        keys.forEach(notifyListeners);
        coverRecords();

        // If we've already retrieved the initial data, and it's the same as the
        // data we just retrieved, then don't notify listeners.
        if (
          resultRef.current.data != null &&
          compare.primitiveArrays(resultRef.current.data, keys) === compare.EQUAL
        )
          setResult((p) => successResult(`retrieved ${name}`, p.data ?? []));
        else setResult(() => successResult(`retrieved ${name}`, recomputeKeys()));
        return true;
      };
      try {
        return await apply();
      } catch (error) {
        if (signal?.aborted || queryRef.current !== query || !mountedRef.current)
          return false;
        setResult(errorResult(`retrieve ${name}`, error));
        return false;
      }
    };
    const retrievePageRef = useSyncedRef(retrievePage);

    const retrieveAsync = async (
      paramsSetter: state.SetArg<Query, Query | {}>,
      options: AsyncListOptions = {},
    ): Promise<void> => {
      if (options.mode !== "append") {
        latestReplaceRef.current = retrievePage(paramsSetter, options);
        await latestReplaceRef.current;
        return;
      }
      // An append pages from the answer to the newest replace. Started earlier, it
      // would become the newest retrieve and discard that answer.
      let replace: Promise<boolean>;
      let answered: boolean;
      do {
        replace = latestReplaceRef.current;
        answered = await replace;
      } while (replace !== latestReplaceRef.current);
      // The wait can span a client change, so the append runs in the newest render.
      if (answered) await retrievePageRef.current(paramsSetter, options);
    };

    /**
     * Fills the list from the cached answer to the query, behind a page that keeps the
     * answer live until a retrieve answers. Does nothing when the cache has no answer.
     */
    const createCachedPage = (query: Query) => {
      const keys = readCache(query);
      if (keys == null) return;
      openPage(query, keys, true);
      keys.forEach(notifyListeners);
      setResult((p) =>
        p.data != null && compare.primitiveArrays(p.data, keys) === compare.EQUAL
          ? p
          : loadingResult(`retrieving ${name}`, keys),
      );
    };

    // The initial state fills the list from the cache in render, where no feed can
    // open, so the mount fills it again with one. When the client is swapped, an active
    // hook must re-point itself at the new client's cache. A non-null query means a
    // list retrieve ran: refetch (retrieveAsync re-subscribes as it goes). Item
    // subscriptions re-open against the new client without refetching.
    const retrieveAsyncRef = useSyncedRef(retrieveAsync);
    const firstMountRef = useRef(true);
    useEffect(() => {
      if (firstMountRef.current) {
        firstMountRef.current = false;
        if (resultRef.current.data != null && pagesRef.current.length === 0)
          createCachedPage(queryRef.current ?? normalized({} as Query));
        return;
      }
      clearPages();
      const itemKeys = [...itemSubsRef.current.keys()];
      clearItemSubs();
      itemKeys.forEach(openItemSub);
      coverRecords();
      const query = queryRef.current;
      if (query != null) void retrieveAsyncRef.current(query);
    }, [client]);

    const retrieveSingle = async (
      key: Key,
      options: query.FetchOptions = {},
    ): Promise<void> => {
      const { signal } = options;
      // The compiler rejects `?.`, `??`, and `?:` in a try, so the body is a closure.
      const apply = async (): Promise<void> => {
        if (client == null || primitive.isZero(key)) return;
        const item = await retrieveByKey({
          client,
          key,
          query: queryRef.current ?? {},
        });
        if (signal?.aborted || item == null || !mountedRef.current) return;
        openItemSub(key);
        if (!filterRef.current(item)) {
          dataRef.current.set(key, null);
          return;
        }
        dataRef.current.set(key, item);
        notifyListeners(key);
      };
      try {
        await apply();
      } catch (error) {
        if (signal?.aborted || !mountedRef.current) return;
        dataRef.current.set(key, null);
        setResult(errorResult(`retrieve ${name}`, error));
      }
    };

    const getOne = (key?: Key) => {
      // Zero-value keys that are not null or undefined are common as initialized
      // fields in various data structures ("", 0, etc.). A 'zero-value' is never
      // valid as a key in Synnax, and a simple null check would result in excessive
      // server refetches for keys we already know are invalid, so we do a full check
      // for a zero-value instead.
      if (primitive.isZero(key)) return undefined;
      const res = dataRef.current.get(key);
      if (res === undefined) void retrieveSingle(key);
      return res;
    };
    const getItem = ((key?: Key | Key[]) =>
      Array.isArray(key)
        ? key.map((k) => getOne(k)).filter((v) => v != null)
        : getOne(key)) as GetItem<Key, Data>;

    const subscribe = (callback: () => void, key?: Key) => {
      if (key == null) return () => {};
      const listeners = listItemListenersRef.current;
      let callbacks = listeners.get(key);
      if (callbacks == null) {
        callbacks = new Set();
        listeners.set(key, callbacks);
      }
      callbacks.add(callback);
      return () => {
        callbacks.delete(callback);
        if (callbacks.size > 0) return;
        listeners.delete(key);
        if (!itemSubsRef.current.has(key)) return;
        closeItemSub(key);
        // Without a feed the copy goes stale, so the next read fetches it again.
        if (!pagesRef.current.some((p) => p.keys.includes(key)))
          dataRef.current.delete(key);
      };
    };

    const debouncedRetrieve = useDebouncedCallback(
      () => {
        const pending = pendingRef.current;
        if (pending == null) return;
        pendingRef.current = null;
        const { replace, append } = pending;
        if (replace != null)
          void retrieveAsyncRef.current(replace.query, replace.options);
        if (append != null) void retrieveAsyncRef.current(append.query, append.options);
      },
      retrieveDebounce,
      [],
    );

    const retrieveSync = (
      query: state.SetArg<Query, Query | {}>,
      options: AsyncListOptions = {},
    ) => {
      const { replace, append } = pendingRef.current ?? {};
      // A frame waits for the answer to each fetch for more, so an append made while a
      // replace waits must still run.
      if (options.mode === "append") {
        if (append != null) return;
        pendingRef.current = { replace, append: { query, options } };
      } else {
        const next = replace == null ? query : chainSetters(replace.query, query);
        pendingRef.current = { replace: { query: next, options }, append };
      }
      // A cold list fills from the cached answer to the exact query the debounced
      // retrieve will fetch first, so a remount paints before the network round trip.
      // Setters must be pure: the debounced retrieve executes the setter again.
      const first = pendingRef.current.replace?.query ?? query;
      if (resultRef.current.data == null)
        createCachedPage(
          normalized(state.executeSetter(first, queryRef.current ?? {})),
        );
      debouncedRetrieve();
    };

    return {
      retrieve: retrieveSync,
      retrieveAsync,
      subscribe,
      getItem,
      ...result,
      data: result.data ?? [],
      answered: result.data != null,
    };
  };
  return useList;
};

export interface UseListItemParams<
  K extends record.Key,
  E extends record.Keyed<K>,
> extends Pick<UseListReturn<query.Params, K, E>, "subscribe" | "getItem"> {
  key: K;
}

export const useListItem = <K extends record.Key, E extends record.Keyed<K>>({
  key,
  subscribe,
  getItem,
}: UseListItemParams<K, E>) =>
  useSyncExternalStore(
    (callback) => subscribe(callback, key),
    () => getItem(key),
    () => undefined,
  );
