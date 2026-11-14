// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, query, type Synnax as Client } from "@synnaxlabs/client";
import {
  useDebouncedCallback,
  useInitializerRef,
  useSyncedRef,
} from "@synnaxlabs/lyra/hooks";
import { useMemoArray, useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import {
  type CrudeTimeSpan,
  type destructor,
  primitive,
  type record,
  TimeSpan,
} from "@synnaxlabs/x";
import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";

import {
  errorResult,
  loadingResult,
  noQueryResult,
  nullClientResult,
  type Result,
  successResult,
} from "@/flux/result";
import { Synnax } from "@/synnax";

/** Reads one record, or many at once, by key. */
export interface GetItem<K extends record.Key, E extends record.Keyed<K>> {
  (key: K): E | undefined;
  (keys: K[]): E[];
}

/**
 * The read surface of a domain client that a list reads. TypeScript infers the types
 * from the last overload of `watch`, so the list overload comes last.
 */
export interface ListSource<Q, E extends record.Keyed<record.Key>, I> {
  watch: {
    (item: NoInfer<I>): query.Live<NoInfer<E>>;
    (query: Q): query.List<E, unknown>;
  };
}

export interface UseListParams<Q, E extends record.Keyed<record.Key>, I = E["key"]> {
  /** The plural noun for the records in status text, such as "ranges". */
  name: string;
  /**
   * The query, apart from its search term and its limit. A search lists in relevance
   * order, so it drops the query's `order`. Null disables the list.
   */
  query?: Q | null;
  /** Hides the records it rejects. */
  filter?: (item: E) => boolean;
  /** How many records each call to `more` reads. */
  pageSize?: number;
  /** How many records the list reads at first. Defaults to `pageSize`. */
  initialSize?: number;
  /** Reads the first page as the list mounts, instead of on the first `more`. */
  eager?: boolean;
  /** How long a search term waits before the list reads it. */
  searchDebounce?: CrudeTimeSpan;
  /** Maps a key to the read of a record outside the list. Defaults to the key. */
  itemQuery?: (key: E["key"]) => I;
}

/** Return value for a list hook. Spread it into a `List.Frame`. */
export type UseListReturn<K extends record.Key, E extends record.Keyed<K>> = Omit<
  Result<K[]>,
  "data"
> & {
  /** The keys of the records in the list, in list order. */
  data: K[];
  /**
   * Whether the list has an answer. An unanswered list has the same empty data as a
   * genuinely empty one, so anything that speaks for the absence of records must wait
   * for this.
   */
  answered: boolean;
  /** Whether the list holds every record that matches. */
  complete: boolean;
  /** Reads a record of the list, or any other record of the domain that is cached. */
  getItem: GetItem<K, E>;
  /**
   * Calls the callback when the record may have changed, so the caller reads it again
   * through getItem. Fetches a record outside the list. Returns a destructor that
   * unsubscribes.
   */
  subscribe: (callback: () => void, key: K) => destructor.Destructor;
  /**
   * Reads the first page, then the next one on each later call. Does nothing while a
   * read runs or once the list holds every record.
   */
  more: () => void;
  /** Lists the records that match the term. An empty term ends the search. */
  search: (term: string) => void;
  /**
   * The search term that data answers. It trails the term given to search until the
   * list reads that term. Empty while the list has no answer.
   */
  searchTerm: string;
};

/** A window and the search term it answers. */
interface Answer<E extends record.Keyed<record.Key>> {
  window: query.Window<E, unknown>;
  searchTerm: string;
}

const DEFAULT_PAGE_SIZE = 50;
const DEFAULT_SEARCH_DEBOUNCE = TimeSpan.milliseconds(100);
const noop = () => {};
const keyItem = <I>(key: record.Key): I => key as I;

/**
 * Reads a live list of the records of a domain client: a filter, the client's declared
 * order, and a window that grows by pages. The client cache keeps the window current,
 * so the hook holds no records of its own. The list reads nothing until the first call
 * to `more` or `search`, so a closed select costs no fetch, unless it is eager.
 *
 * @param source - Selects the domain client, e.g. `(client) => client.ranges`.
 */
export const useList = <Q, E extends record.Keyed<record.Key>, I = E["key"]>(
  source: (client: Client) => ListSource<Q, E, I>,
  {
    name,
    query: q = {} as Q,
    filter,
    pageSize = DEFAULT_PAGE_SIZE,
    initialSize = pageSize,
    eager = false,
    searchDebounce = DEFAULT_SEARCH_DEBOUNCE,
    itemQuery = keyItem,
  }: UseListParams<NoInfer<Q>, E, I>,
): UseListReturn<E["key"], E> => {
  type K = E["key"];
  const op = `retrieve ${name}`;
  const client = Synnax.use();
  const [started, setStarted] = useState(eager);
  if (eager && !started) setStarted(true);
  const [term, setTerm] = useState("");
  // The list starts with the term, so the first search reads no unsearched page.
  const debouncedSearch = useDebouncedCallback(
    (next: string) => {
      setTerm(next);
      setStarted(true);
    },
    searchDebounce,
    [],
  );
  // Q is left open so it can match each client's overloaded request input.
  const base = q as { order?: unknown } | null;
  let next: Q | null = null;
  if (base != null && term === "") next = { ...base, limit: initialSize } as Q;
  else if (base != null) {
    const { order: _, ...unordered } = base;
    next = { ...unordered, searchTerm: term, limit: initialSize } as Q;
  }
  const listQuery = useMemoDeepEqual(next);
  const retriever = client == null ? null : source(client);
  const list =
    retriever == null || listQuery == null ? null : retriever.watch(listQuery);
  const current = useSyncExternalStore(
    (callback) => (list == null ? noop : list.onChange(callback)),
    () => list?.getCached(),
  );
  // A new query, such as a new search term, shows the last answer until its own lands.
  const [last, setLast] = useState<Answer<E> | undefined>(() =>
    current == null ? undefined : { window: current, searchTerm: term },
  );
  if (current != null && current !== last?.window)
    setLast({ window: current, searchTerm: term });
  const [status, setStatus] = useState<Result<null>>(() => loadingResult<null>(op));
  // Set only while a new query shows the last answer.
  const shown = list == null || current != null ? undefined : last;
  const window = current ?? shown?.window;
  const searchTerm = window == null ? "" : (shown?.searchTerm ?? term);
  const reading = useRef<object | null>(null);

  const read = async (step: string, run: () => Promise<unknown>): Promise<void> => {
    const token = {};
    reading.current = token;
    setStatus(loadingResult<null>(step));
    let next: Result<null> = successResult<null>(step, null);
    try {
      await run();
    } catch (error) {
      next = errorResult(step, error);
    }
    if (reading.current !== token) return;
    reading.current = null;
    // A failed read drops the last answer, so neither the error nor a retry shows
    // records from an earlier query.
    if (next.variant === "error") setLast(undefined);
    setStatus(next);
  };

  useEffect(() => {
    if (list == null || !started) return;
    void read(op, async () => await list.retrieve());
    return () => {
      reading.current = null;
    };
  }, [list, started]);

  const more = () => {
    if (!started) return setStarted(true);
    if (list == null || reading.current != null) return;
    // A failed first read leaves no window, so the first page reads again.
    if (current == null) return void read(op, async () => await list.retrieve());
    if (current.end == null) return;
    void read(`retrieve more ${name}`, async () => await list.more(pageSize));
  };

  const search = (next: string) => debouncedSearch(next);

  const records = window?.records;
  // getItem keeps one identity for the life of the hook and reads these at call time,
  // so a new window renders again only the readers whose records it changes.
  const reads = useSyncedRef({
    list,
    shown: shown?.window,
    retriever,
    filter,
    itemQuery,
  });
  // A window keeps the records that a change leaves, so a reader renders again only for
  // its own record.
  const readers = useInitializerRef(() => new Set<() => void>());
  const membersByWindow = useInitializerRef(
    () => new WeakMap<query.Window<E, unknown>, Map<K, E>>(),
  );
  const notifyReaders = () => readers.current.forEach((read) => read());
  useEffect(() => {
    if (list == null) return;
    return list.onChange(notifyReaders);
  }, [list]);
  // Readers read the values of the last commit, so a new value makes them read again.
  useLayoutEffect(notifyReaders, [list, shown, retriever, filter, itemQuery]);
  const membersOf = (answer?: query.Window<E, unknown>): Map<K, E> | undefined => {
    if (answer == null) return undefined;
    let members = membersByWindow.current.get(answer);
    if (members == null) {
      members = new Map(answer.records.map((r) => [r.key, r]));
      membersByWindow.current.set(answer, members);
    }
    return members;
  };
  // A select with no value reads its zero key.
  const getOne = (key: K): E | undefined => {
    const latest = reads.current;
    const keep = (record: E | undefined): E | undefined =>
      record == null || (latest.filter != null && !latest.filter(record))
        ? undefined
        : record;
    const member = membersOf(latest.shown ?? latest.list?.getCached())?.get(key);
    if (member != null) return keep(member);
    if (latest.retriever == null || primitive.isZero(key)) return undefined;
    const cached = latest.retriever.watch(latest.itemQuery(key)).getCached();
    return keep(query.isLive(cached) ? cached : undefined);
  };
  const getItem = ((keys: K | K[]) => {
    if (!Array.isArray(keys)) return getOne(keys);
    return keys.map(getOne).filter((r): r is E => r != null);
  }) as GetItem<K, E>;

  const subscribe = (callback: () => void, key: K): destructor.Destructor => {
    if (retriever == null || primitive.isZero(key)) return noop;
    const read = () => callback();
    readers.current.add(read);
    const live = retriever.watch(itemQuery(key));
    const release = live.onChange(read);
    if (live.getCached() === undefined)
      live.retrieve().then(callback, (error: unknown) => {
        // The subscriber reads a missing record from getItem.
        if (!NotFoundError.matches(error)) console.error(error);
      });
    return () => {
      readers.current.delete(read);
      release();
    };
  };

  const data = useMemoArray(
    (records ?? []).filter((r) => filter == null || filter(r)).map((r) => r.key),
  ) as K[];

  const rest = {
    data,
    answered: window != null,
    complete: window != null && window.end == null,
    getItem,
    subscribe,
    more,
    search,
    searchTerm,
  };
  if (client == null) return { ...nullClientResult<K[]>(op), ...rest };
  if (listQuery == null) return { ...noQueryResult<K[]>(op), ...rest };
  return { ...status, ...rest };
};
