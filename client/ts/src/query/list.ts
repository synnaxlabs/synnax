// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  compare,
  type destructor,
  type record,
  type TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { ValidationError } from "@/errors";
import {
  DEFAULT_DEBOUNCE,
  DEFAULT_TEARDOWN_GRACE,
  hash,
  type SpaceHooks,
  startStreaming,
  type Trigger,
} from "@/query/query";
import { isChange, type Keyed, type Table, type TableEvent } from "@/query/table";
import { type Data, type FetchOptions, type Params } from "@/query/types";

/** The most records a search returns. */
export const MAX_SEARCH_LIMIT = 1000;

/** A window of a list: its records in list order, and where the window ends. */
export interface Window<D extends Data, C> {
  /** The records in the window, in list order. */
  records: D[];
  /**
   * The cursor of the last record fetched, or null when the window holds every match.
   * A match that sorts past it waits for {@link List.more}.
   */
  end: C | null;
}

/**
 * Receives the new window of a list. `undefined` means the window was discarded; the
 * next read fetches it again.
 */
export interface WindowHandler<D extends Data, C> {
  (window: Window<D, C> | undefined): void;
}

/**
 * A live window over the records that match a query, in the query's order. Every list
 * of one query, apart from its limit, shares one window: the largest that any reader
 * asked for. A search is a list too, in relevance order.
 */
export interface List<D extends Data, C> {
  /**
   * Returns the window, grown to at least the query's limit, or to every match when the
   * query sets none. Serves the cache while the window is maintained, and fetches
   * otherwise.
   */
  retrieve: (options?: FetchOptions) => Promise<Window<D, C>>;
  /**
   * Fetches up to n matches past the window's end and adds them. A search instead
   * fetches its window again, at least doubled, up to {@link MAX_SEARCH_LIMIT}
   * records. Fetches nothing when the window holds every match.
   */
  more: (n: number, options?: FetchOptions) => Promise<Window<D, C>>;
  /** Returns the window without touching the network, or undefined before a fetch. */
  getCached: () => Window<D, C> | undefined;
  /**
   * Subscribes to changes in the window. The window is maintained while a subscriber
   * exists, and for a grace window after the last one leaves. Returns a destructor that
   * unsubscribes.
   */
  onChange: (handler: WindowHandler<D, C>) => destructor.Destructor;
}

/** The fields of a query that a list reads. */
export interface ListQuery<O, C> {
  limit?: number;
  order?: O;
  after?: C;
  searchTerm?: string;
}

/**
 * Declaration of the lists of one table. A list whose query only the Core can check is
 * a search: any change in the table fetches its window again. Every other list keeps
 * its window from changes, with no network.
 */
export interface ListsConfig<
  Q extends ListQuery<O, C> & { readonly [key: string]: Params },
  D extends Data,
  K extends record.Key,
  V extends Keyed<K>,
  C,
  O,
> {
  /** Resource name used in error messages, e.g. "range". */
  name: string;
  /** The table owning the records of the lists. */
  table: Table<K, V>;
  /**
   * Fetches the records that match the query from the Core: in its order, past its
   * cursor, and up to its limit.
   */
  fetch: (query: Q, options?: FetchOptions) => Promise<V[]>;
  /** Assembles a record of the window at read time. */
  compose: (record: V, query: Q) => D;
  /**
   * Whether a record satisfies the query. Pure; no network. Null when the client cannot
   * decide the record, which fetches the window again. Absent, every list is a search.
   */
  matches?: (record: V, query: Q) => boolean | null;
  /** Whether only the Core can check the query, such as a regex. */
  serverComputed?: (query: Q) => boolean;
  /**
   * The keys of a query that addresses keys and nothing else, or null for any other
   * query. A window of such a query composes from the table, with no fetch, when the
   * table knows every key.
   */
  keysOf?: (query: Q) => K[] | null;
  /** Returns the position of a record in the order, or in key order when absent. */
  cursor: (record: V, order?: O) => C;
  /** Orders two positions as the Core walks them. */
  compare: (a: C, b: C, order?: O) => number;
  /** Foreign tables whose changes reach the records of the windows. */
  triggers?: Array<Trigger<Q, K>>;
}

interface Seen {
  /** The versions of the table and of each trigger's table. */
  versions: number[];
  /** Null while the stream does not maintain the tables. */
  epoch: number | null;
}

interface Entry<Q, K extends record.Key, D extends Data, V, C> {
  /** The query without its limit and cursor. */
  query: Q;
  /** Whether only the Core can order and match the window, as in a search. */
  ranked: boolean;
  /** The window's keys, in list order. */
  keys: K[];
  end: C | null;
  fetched: boolean;
  /** The largest limit a reader asked for: Infinity for every match, 0 before any. */
  limit: number;
  /** The limit of the ranked fetch behind the window. */
  searchLimit?: number;
  handlers: Set<WindowHandler<D, C>>;
  /** The list handles of the window, by the limit they ask for. */
  lists: Map<number | undefined, List<D, C>>;
  /** Present exactly while the window is maintained. */
  teardown?: destructor.Destructor;
  idleSince?: TimeStamp;
  /** The last fetch queued. Fetches of one window run one at a time. */
  queue: Promise<unknown>;
  /** Counts resets. A fetch applies only in the generation it started in. */
  generation: number;
  /** Keys that changed while a fetch was in flight, set only during one. */
  rechecks?: Set<K>;
  /** Set while the window ingests its own fetch, so its events skip the window. */
  ingesting: boolean;
  /** Set when the window may have missed a change. The next read fetches it. */
  stale: boolean;
  /** The table versions and epoch when the window stopped seeing changes. */
  seen?: Seen;
  refetchTimer?: ReturnType<typeof setTimeout>;
  composed?: {
    records: V[];
    keys: K[];
    end: C | null;
    versions: number[];
    value: Window<D, C>;
  };
  /**
   * The composed record of each member and the table record it came from. A member
   * keeps its composed record until its table record or a watched table changes it.
   * Only a maintained window holds it, since an unmaintained one misses those changes.
   */
  composedByKey?: Map<K, { source: V; value: D }>;
  /** The window last delivered to the handlers. */
  delivered?: Window<D, C>;
}

/**
 * The lists of one table: windows of records in a declared order, kept current from
 * table changes while subscribed. Records stay in the table, so a window holds only
 * keys and its content cannot diverge from the record cache.
 */
export class Lists<
  Q extends ListQuery<O, C> & { readonly [key: string]: Params },
  D extends Data,
  K extends record.Key,
  V extends Keyed<K>,
  C,
  O,
> {
  private readonly entries = new Map<string, Entry<Q, K, D, V, C>>();
  private readonly config: ListsConfig<Q, D, K, V, C, O>;
  private readonly hooks: SpaceHooks & { onError: (error: Error) => void };
  private readonly detachEpoch?: destructor.Destructor;
  private readonly release: destructor.Destructor;
  private readonly grace: TimeSpan;
  private sweepTimer?: ReturnType<typeof setTimeout>;

  constructor(config: ListsConfig<Q, D, K, V, C, O>, hooks: SpaceHooks = {}) {
    this.config = config;
    this.hooks = { ...hooks, onError: hooks.onError ?? console.error };
    this.grace = hooks.teardownGrace ?? DEFAULT_TEARDOWN_GRACE;
    this.release = config.table.hold({
      held: () => this.held(),
      drop: (keys) => this.drop(keys),
    });
    this.detachEpoch = hooks.onEpoch?.((epoch) => {
      // The fresh stream's own epoch refetches after a reset, not the reset itself.
      if (epoch === 0) return;
      this.entries.forEach((entry) => {
        entry.stale = true;
        if (entry.teardown != null) this.scheduleRefetch(entry);
      });
    });
  }

  /**
   * Returns the list of the records that match the query. Equal queries return the same
   * list, so a reader can key a subscription on it.
   */
  list(query: Q): List<D, C> {
    const { limit, after: _after, ...rest } = query;
    if (limit != null && limit <= 0)
      throw new ValidationError(`a ${this.config.name} list needs a positive limit`);
    const base = (
      rest.searchTerm === "" ? { ...rest, searchTerm: undefined } : rest
    ) as Q;
    const entry = this.ensure(base);
    let list = entry.lists.get(limit);
    if (list == null) {
      list = {
        retrieve: async (options) => await this.retrieve(entry, limit, options),
        more: async (n, options) => await this.more(entry, n, options),
        getCached: () => this.cachedOf(entry),
        onChange: (handler) => this.subscribe(entry, limit, handler),
      };
      entry.lists.set(limit, list);
    }
    return list;
  }

  /**
   * Discards every window and notifies subscribers that their window was discarded, so
   * the next read fetches it again. Maintenance stays mounted. Called by the cache when
   * the cluster behind the connection is replaced.
   */
  reset(): void {
    this.entries.forEach((entry) => {
      this.clearRefetch(entry);
      entry.generation++;
      entry.keys = [];
      entry.end = null;
      entry.fetched = false;
      entry.searchLimit = undefined;
      entry.stale = false;
      entry.composed = undefined;
      entry.composedByKey = undefined;
      this.deliver(entry);
    });
  }

  /**
   * Fetches again, after a short wait, each window whose query the predicate accepts.
   * For changes the table cannot see, such as a record the caller may not read.
   */
  refetchMatching(predicate: (query: Q) => boolean): void {
    this.entries.forEach((entry) => {
      if (predicate(entry.query)) this.scheduleRefetch(entry);
    });
  }

  /** Detaches every subscription. Windows and handlers are dropped. */
  close(): void {
    this.detachEpoch?.();
    this.release();
    if (this.sweepTimer != null) clearTimeout(this.sweepTimer);
    this.sweepTimer = undefined;
    this.entries.forEach((entry) => {
      entry.teardown?.();
      this.clearRefetch(entry);
    });
    this.entries.clear();
  }

  /** The keys of every maintained window. */
  private *held(): Iterable<K> {
    for (const { teardown, keys } of this.entries.values())
      if (teardown != null) yield* keys;
  }

  /** Discards each unmaintained window that holds a dropped record. */
  private drop(keys: ReadonlySet<K>): void {
    this.entries.forEach((entry) => {
      if (entry.teardown != null || !entry.keys.some((key) => keys.has(key))) return;
      entry.keys = [];
      entry.end = null;
      entry.fetched = false;
      entry.searchLimit = undefined;
      entry.composed = undefined;
      entry.composedByKey = undefined;
    });
  }

  private ensure(query: Q): Entry<Q, K, D, V, C> {
    const hashed = hash(query);
    let entry = this.entries.get(hashed);
    if (entry == null) {
      const { matches, serverComputed } = this.config;
      entry = {
        query,
        ranked:
          matches == null ||
          query.searchTerm != null ||
          serverComputed?.(query) === true,
        keys: [],
        end: null,
        fetched: false,
        limit: 0,
        handlers: new Set(),
        lists: new Map(),
        queue: Promise.resolve(),
        generation: 0,
        ingesting: false,
        stale: false,
      };
      this.entries.set(hashed, entry);
    }
    return entry;
  }

  private async retrieve(
    entry: Entry<Q, K, D, V, C>,
    limit: number | undefined,
    options?: FetchOptions,
  ): Promise<Window<D, C>> {
    entry.limit = Math.max(entry.limit, limit ?? Infinity);
    void startStreaming(this.hooks, (exc) => this.report(exc));
    return await this.enqueue(entry, async () => {
      if (!this.trusted(entry) || (entry.ranked && this.searchShort(entry)))
        await this.replace(entry, options);
      else if (entry.end != null && (limit == null || entry.keys.length < limit))
        await this.extend(
          entry,
          limit == null ? undefined : limit - entry.keys.length,
          options,
        );
      return this.windowOf(entry);
    });
  }

  private async more(
    entry: Entry<Q, K, D, V, C>,
    n: number,
    options?: FetchOptions,
  ): Promise<Window<D, C>> {
    return await this.enqueue(entry, async () => {
      if (n <= 0) return this.windowOf(entry);
      if (!this.trusted(entry)) {
        entry.limit = Math.max(entry.limit, entry.keys.length + n);
        await this.replace(entry, options);
      } else if (entry.end != null)
        if (entry.ranked) {
          const searchLimit = entry.searchLimit ?? 0;
          entry.limit = Math.min(
            Math.max(searchLimit + n, 2 * searchLimit),
            MAX_SEARCH_LIMIT,
          );
          if (entry.limit > searchLimit) await this.replace(entry, options);
        } else await this.extend(entry, n, options);
      return this.windowOf(entry);
    });
  }

  /** Whether the window is current: fetched, maintained, and with no missed change. */
  private trusted(entry: Entry<Q, K, D, V, C>): boolean {
    return (
      entry.fetched &&
      entry.teardown != null &&
      !entry.stale &&
      this.hooks.maintained?.() !== false
    );
  }

  /** Whether a reader asked a search for more records than its fetch returned. */
  private searchShort(entry: Entry<Q, K, D, V, C>): boolean {
    if (entry.end == null) return false;
    return entry.limit > (entry.searchLimit ?? Infinity);
  }

  /** Runs the task after every fetch queued before it. The queue never rejects. */
  private async enqueue<T>(entry: Entry<Q, K, D, V, C>, task: () => Promise<T>) {
    const run = entry.queue.then(task);
    entry.queue = run.catch(() => undefined);
    return await run;
  }

  /** Fetches the window from its start, replacing it. */
  private async replace(
    entry: Entry<Q, K, D, V, C>,
    options?: FetchOptions,
  ): Promise<void> {
    this.clearRefetch(entry);
    const { ranked } = entry;
    // A refetch keeps the records that more added.
    const size = finite(
      ranked ? entry.limit : Math.max(entry.limit, entry.keys.length),
    );
    const page = await this.page(entry, { ...entry.query, limit: size }, options);
    // A reset discarded the window, so the read goes to the new cluster.
    if (page == null) return await this.replace(entry, options);
    entry.fetched = true;
    entry.stale = false;
    entry.end = this.endOf(entry, page.records, size);
    if (ranked) {
      entry.searchLimit = size;
      entry.keys = page.records.map(({ key }) => key);
      // Only the Core can place a change that landed during the fetch.
      if (page.rechecks.size > 0) this.scheduleRefetch(entry);
    } else {
      entry.keys = [];
      this.admitPage(entry, page);
    }
    this.deliver(entry);
  }

  /** Fetches up to n records past the window's end, or every one when n is absent. */
  private async extend(
    entry: Entry<Q, K, D, V, C>,
    n: number | undefined,
    options?: FetchOptions,
  ): Promise<void> {
    const after = entry.end as C;
    const page = await this.page(entry, { ...entry.query, after, limit: n }, options);
    if (page == null) return await this.replace(entry, options);
    entry.end = this.endOf(entry, page.records, n);
    this.admitPage(entry, page);
    entry.limit = Math.max(entry.limit, entry.keys.length);
    this.deliver(entry);
  }

  /**
   * Fetches one page and writes it into the table. Returns null when a reset discarded
   * the window while the fetch was in flight. The returned rechecks hold the keys that
   * changed during the fetch, which the page may predate.
   */
  private async page(
    entry: Entry<Q, K, D, V, C>,
    query: Q,
    options?: FetchOptions,
  ): Promise<{ records: V[]; rechecks: Set<K> } | null> {
    const { table, fetch } = this.config;
    const { generation } = entry;
    const rechecks = entry.rechecks ?? new Set<K>();
    entry.rechecks = rechecks;
    // A change that lands during the fetch reaches no unmaintained window.
    const before = this.seen();
    return await table.track(async (since) => {
      let records: V[];
      try {
        records = await fetch(query, options);
      } finally {
        if (entry.rechecks === rechecks) entry.rechecks = undefined;
      }
      if (entry.generation !== generation) return null;
      const missed = !unchanged(before, this.seen());
      entry.ingesting = true;
      try {
        table.ingest(records, { since });
      } finally {
        entry.ingesting = false;
      }
      if (entry.teardown == null) entry.seen = missed ? before : this.seen();
      return { records, rechecks };
    });
  }

  private endOf(entry: Entry<Q, K, D, V, C>, records: V[], limit?: number): C | null {
    const last = records.at(-1);
    if (limit == null || records.length < limit || last == null) return null;
    return this.config.cursor(last, entry.ranked ? undefined : entry.query.order);
  }

  /** Admits a page and the keys that changed while it was in flight. */
  private admitPage(
    entry: Entry<Q, K, D, V, C>,
    { records, rechecks }: { records: V[]; rechecks: Set<K> },
  ): void {
    const keys = records.map(({ key }) => key);
    this.admit(entry, [...keys, ...rechecks], new Set(keys));
  }

  /**
   * Places each changed key in the window: at its sorted place when its record matches
   * and sorts at or before the end, and out of the window otherwise. The Core vouches
   * for the keys in fetched, so the client's undecided match admits them. Returns
   * whether the window changed.
   */
  private admit(
    entry: Entry<Q, K, D, V, C>,
    keys: Iterable<K>,
    fetched?: Set<K>,
  ): boolean {
    const { table } = this.config;
    const changed = new Set(keys);
    const kept = entry.keys.filter((key) => !changed.has(key));
    let moved = kept.length !== entry.keys.length;
    for (const key of changed) {
      const record = table.get(key);
      if (record == null || !this.qualifies(entry, record, fetched?.has(key) === true))
        continue;
      kept.splice(this.placeOf(entry, kept, record), 0, key);
      moved = true;
    }
    entry.keys = kept;
    return moved;
  }

  private qualifies(entry: Entry<Q, K, D, V, C>, record: V, fetched: boolean): boolean {
    const { matches, cursor, compare } = this.config;
    const matched = matches?.(record, entry.query) ?? null;
    if (matched == null && !fetched) {
      this.scheduleRefetch(entry);
      return false;
    }
    if (matched === false) return false;
    const { end, query } = entry;
    return end == null || compare(cursor(record, query.order), end, query.order) <= 0;
  }

  /** Returns the index at which the record goes in the sorted keys. */
  private placeOf(entry: Entry<Q, K, D, V, C>, keys: K[], record: V): number {
    const { table, cursor, compare, name } = this.config;
    const { order } = entry.query;
    const position = cursor(record, order);
    let low = 0;
    let high = keys.length;
    while (low < high) {
      const mid = (low + high) >>> 1;
      const other = table.get(keys[mid]);
      if (other == null) throw new Error(`a ${name} list holds a key with no record`);
      if (compare(cursor(other, order), position, order) < 0) low = mid + 1;
      else high = mid;
    }
    return low;
  }

  private onEvents(entry: Entry<Q, K, D, V, C>, events: Array<TableEvent<K, V>>): void {
    if (entry.ingesting) return;
    const changes = entry.ranked ? events.filter(isChange) : events;
    if (changes.length === 0) return;
    const keys = changes.map(({ key }) => key);
    if (entry.rechecks != null) keys.forEach((key) => entry.rechecks?.add(key));
    if (!entry.fetched) return;
    if (!entry.ranked) {
      if (this.admit(entry, keys)) this.deliver(entry);
      return;
    }
    // Only the Core knows relevance. A change to a member shows at once, so a local
    // write renders before the refetch lands.
    this.scheduleRefetch(entry);
    const members = new Set(entry.keys);
    if (keys.some((key) => members.has(key))) this.deliver(entry);
  }

  private subscribe(
    entry: Entry<Q, K, D, V, C>,
    limit: number | undefined,
    handler: WindowHandler<D, C>,
  ): destructor.Destructor {
    void startStreaming(this.hooks, (exc) => this.report(exc));
    // A refetch sizes the window to the largest limit a reader asked for.
    entry.limit = Math.max(entry.limit, limit ?? Infinity);
    entry.handlers.add(handler);
    entry.idleSince = undefined;
    if (entry.teardown == null) this.maintain(entry);
    // A change in the grace window found no subscriber to refetch for.
    else if (entry.stale && entry.fetched) this.scheduleRefetch(entry);
    return () => {
      entry.handlers.delete(handler);
      if (entry.handlers.size > 0) return;
      if (this.grace.isZero) return this.teardown(entry);
      entry.idleSince = TimeStamp.now();
      this.armSweep();
    };
  }

  /**
   * Handles a change to a trigger's table. The window's records are the same objects,
   * so a change to a member composes the window again.
   */
  private onTrigger(
    entry: Entry<Q, K, D, V, C>,
    trigger: Trigger<Q, K>,
    result: K[] | "refetch",
  ): void {
    if (entry.ingesting || !entry.fetched) return;
    if (result === "refetch") {
      // The change can reach any member.
      entry.composedByKey = undefined;
      return this.scheduleRefetch(entry);
    }
    // Admitting can drop a member that a refetch puts back, so the composed records go
    // first.
    const members = new Set(entry.keys);
    result.forEach((key) => entry.composedByKey?.delete(key));
    if (result.some((key) => members.has(key))) entry.composed = undefined;
    if (trigger.contentOnly !== true)
      if (entry.ranked) this.scheduleRefetch(entry);
      else this.admit(entry, result);
    const admitted = new Set(entry.keys);
    if (!result.some((key) => admitted.has(key))) return;
    this.deliver(entry);
  }

  private maintain(entry: Entry<Q, K, D, V, C>): void {
    const { table, triggers = [] } = this.config;
    const releases = [
      table.subscribeBatch((events) => this.onEvents(entry, events)),
      ...triggers.map((t) =>
        t.attach(entry.query, (result) => this.onTrigger(entry, t, result)),
      ),
    ];
    entry.teardown = () => releases.forEach((release) => release());
    // The first fetch starts later, and its answer may predate a change made before it.
    if (!entry.fetched) entry.rechecks ??= new Set();
    if (entry.fetched && !unchanged(entry.seen, this.seen())) entry.stale = true;
    if (!entry.stale || !entry.fetched) return;
    // A record deleted while unmaintained leaves its key until the refetch lands.
    entry.keys = entry.keys.filter((key) => table.get(key) != null);
    // A trigger's change leaves the records as they were, so the window composes again.
    entry.composed = undefined;
    this.scheduleRefetch(entry);
  }

  private teardown(entry: Entry<Q, K, D, V, C>): void {
    entry.idleSince = undefined;
    entry.teardown?.();
    entry.teardown = undefined;
    entry.composedByKey = undefined;
    entry.seen = this.seen();
    this.clearRefetch(entry);
  }

  private seen(): Seen {
    const maintained = this.hooks.maintained?.() !== false;
    const epoch = this.hooks.epoch?.() ?? 1;
    return {
      versions: this.versions(),
      epoch: maintained && epoch !== 0 ? epoch : null,
    };
  }

  /** The versions of the table and of each trigger's table. */
  private versions(): number[] {
    const { table, triggers = [] } = this.config;
    return [table.version(), ...triggers.map((t) => t.version())];
  }

  /** Arms one sweep timer per table while any window sits in the grace window. */
  private armSweep(): void {
    this.sweepTimer ??= setTimeout(() => {
      this.sweepTimer = undefined;
      this.sweep();
    }, this.grace.milliseconds / 2);
  }

  private sweep(): void {
    let idle = false;
    const now = TimeStamp.now();
    this.entries.forEach((entry) => {
      if (entry.idleSince == null) return;
      if (entry.handlers.size > 0) {
        entry.idleSince = undefined;
        return;
      }
      if (now.span(entry.idleSince).lessThan(this.grace)) {
        idle = true;
        return;
      }
      this.teardown(entry);
    });
    if (idle) this.armSweep();
  }

  /** Refetches a maintained window after a debounce. An idle window refetches on its
   *  next read instead. */
  private scheduleRefetch(entry: Entry<Q, K, D, V, C>): void {
    entry.stale = true;
    if (entry.handlers.size === 0 || entry.refetchTimer != null) return;
    entry.refetchTimer = setTimeout(() => {
      entry.refetchTimer = undefined;
      // A failed refetch leaves the window in place, and the next read fetches it.
      this.enqueue(entry, async () => await this.replace(entry)).catch(() => undefined);
    }, DEFAULT_DEBOUNCE.milliseconds);
  }

  private clearRefetch(entry: Entry<Q, K, D, V, C>): void {
    if (entry.refetchTimer == null) return;
    clearTimeout(entry.refetchTimer);
    entry.refetchTimer = undefined;
  }

  private cachedOf(entry: Entry<Q, K, D, V, C>): Window<D, C> | undefined {
    if (!entry.fetched) this.fill(entry);
    return entry.fetched ? this.windowOf(entry) : undefined;
  }

  /** Fills a keys-only window from the table when the table knows every key. */
  private fill(entry: Entry<Q, K, D, V, C>): void {
    const { table, keysOf } = this.config;
    const keys = keysOf?.(entry.query);
    if (keys == null || keys.some((key) => table.status(key) === "unknown")) return;
    const present = keys.filter((key) => table.status(key) === "present");
    entry.keys = [];
    entry.end = null;
    this.admit(entry, present, new Set(present));
    entry.fetched = true;
    entry.stale = false;
    if (entry.teardown == null) entry.seen = this.seen();
  }

  /**
   * Composes the window, interned against its records so repeated reads stay
   * referentially stable. An unchanged member keeps its composed record, so a reader
   * can find the changed records by identity.
   */
  private windowOf(entry: Entry<Q, K, D, V, C>): Window<D, C> {
    const { table, compose } = this.config;
    const { composed, end, keys, query } = entry;
    const versions = this.versions();
    // Keys are replaced, never edited, so an unchanged table answers in constant time.
    if (
      composed?.keys === keys &&
      composed.end === end &&
      compare.arraysEqual(composed.versions, versions)
    )
      return composed.value;
    const records = table.get(keys);
    // An unmaintained window misses trigger changes, so equal records can hide one.
    if (
      entry.teardown != null &&
      composed != null &&
      composed.end === end &&
      compare.arraysEqual(composed.records, records)
    ) {
      entry.composed = { ...composed, keys, versions };
      return composed.value;
    }
    const composedByKey = new Map<K, { source: V; value: D }>();
    const composedRecords = records.map((source) => {
      const previous = entry.composedByKey?.get(source.key);
      const value =
        previous?.source === source ? previous.value : compose(source, query);
      composedByKey.set(source.key, { source, value });
      return value;
    });
    const value = { records: composedRecords, end };
    entry.composed = { records, keys, end, versions, value };
    if (entry.teardown != null) entry.composedByKey = composedByKey;
    return value;
  }

  /** Delivers the window to every handler, unless they already hold it. */
  private deliver(entry: Entry<Q, K, D, V, C>): void {
    const window = this.cachedOf(entry);
    if (window === entry.delivered) return;
    entry.delivered = window;
    entry.handlers.forEach((handler) => {
      try {
        handler(window);
      } catch (exc) {
        this.report(exc, `failed to notify ${this.config.name} list subscriber`);
      }
    });
  }

  private report(
    exc: unknown,
    message = `failed to maintain ${this.config.name} lists`,
  ): void {
    this.hooks.onError(new Error(message, { cause: exc }));
  }
}

/** Whether the tables took no change between the two snapshots. */
const unchanged = (before: Seen | undefined, now: Seen): boolean =>
  before?.epoch != null &&
  compare.arraysEqual(before.versions, now.versions) &&
  before.epoch === now.epoch;

/** Returns the limit to send the Core: undefined for every record. */
const finite = (limit: number): number | undefined =>
  Number.isFinite(limit) ? limit : undefined;
