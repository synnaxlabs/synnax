// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  array,
  type CrudeTimeSpan,
  deep,
  destructor,
  errors,
  record,
  scheduler,
  state,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import z from "zod";

import {
  AccessDeniedError,
  DisconnectedError,
  isConnectionError,
  NotFoundError,
} from "@/errors";
import { Deleted } from "@/query/deleted";
import { type LookupIndex } from "@/query/indexes";
import { type Listener } from "@/query/streamer";

/**
 * Presence of a key in a table: "present" when a live entry exists, "tombstoned" when
 * the entry was deleted and its corpse is retained, "unknown" when the table has never
 * seen the key or the corpse was cleared.
 */
export type EntryStatus = "present" | "tombstoned" | "unknown";

/** A change to a single entry of a {@link Table}. */
export type TableEvent<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> =
  | {
      variant: "set";
      key: Key;
      value: Value;
      /**
       * Set when a fetch found a record that the table did not hold. The record is new
       * to the table, not changed on the Core.
       */
      discovered?: boolean;
    }
  | { variant: "delete"; key: Key };

/**
 * Returns whether the event can change what the Core answers to a query: every event
 * except a set that a fetch found.
 */
export const isChange = <Key extends record.Key, Value extends state.State>(
  event: TableEvent<Key, Value>,
): boolean => event.variant === "delete" || event.discovered !== true;

/** Receives {@link TableEvent}s. Must be synchronous; thrown errors are reported to the
 *  table's error sink and do not stop delivery. */
export interface TableSubscriber<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> {
  (event: TableEvent<Key, Value>): void;
}

/**
 * Receives every event of a batched write as one call, in event order. A single write
 * delivers a batch of one. Same synchrony and error contract as
 * {@link TableSubscriber}.
 */
export interface BatchSubscriber<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> {
  (events: TableEvent<Key, Value>[]): void;
}

/**
 * How fetched records hydrate a table: "set" overwrites entries, "if-absent" preserves
 * existing entries.
 */
export type HydrateMode = "set" | "if-absent";

/**
 * Something composed from the records of a table, such as an answer or a list, that
 * keeps the records it needs from {@link Table.trim}.
 */
export interface Holder<Key extends record.Key> {
  /** Returns the keys whose records the holder needs. */
  held: () => Iterable<Key>;
  /** Discards anything composed from the dropped records. */
  drop?: (keys: ReadonlySet<Key>) => void;
}

/** A point in a table's write history, handed to a fetch by {@link Table.track}. */
export type Stamp = number;

/** Options for {@link Table.ingest}. */
export interface IngestOptions {
  /** Overrides the table's declared hydrate mode. */
  mode?: HydrateMode;
  /**
   * Skips entries written or deleted after this stamp, and every entry if the table was
   * reset after it: the fetch predates them.
   */
  since?: Stamp;
  /**
   * Set when the records changed on the Core, such as records that the change stream
   * announced. Each write is then a change, even for a record new to the table.
   */
  changed?: boolean;
}

/** A table value carrying its entry key, so batch writes and fetches can derive each
 *  entry's key from the record itself. */
export type Keyed<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> = Value & record.Keyed<Key>;

/** Parses the key of a record and ignores its other fields. */
export const keyedZ = z.object({ key: record.keyZ });

/** Construction arguments for a {@link Table}. */
export interface TableParams<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> {
  /** Receives subscriber and fetch errors that have no caller to throw to. */
  onError: (error: Error) => void;
  /** Overrides the deep-equality default used to silence redundant sets. */
  equal?: (a: Value, b: Value, key: Key) => boolean;
  /**
   * Fetches the current records for the given keys from the cluster, returning only the
   * entries that still exist. Powers {@link Table.retrieve}, key-announce listeners,
   * and reconciliation. Omit for tables with no server backing (they are skipped by all
   * three).
   */
  fetch?: (keys: Key[]) => Promise<Array<Keyed<Key, Value>>>;
  /**
   * Hydration mode, "set" by default. Dispatch-backed document tables use "if-absent"
   * so fetches never clobber locally replayed edits.
   */
  hydrate?: HydrateMode;
  /**
   * Window for coalescing concurrent miss fetches into one fetch call.
   * @default TimeSpan.milliseconds(10)
   */
  fetchDebounce?: CrudeTimeSpan;
  /**
   * Secondary indexes over the table's live entries. The table keeps each current
   * across every mutation, rollbacks and resets included.
   */
  indexes?: Array<LookupIndex<Key, Value>>;
  /**
   * Lets a fetch that started after a delete restore the key. For tables whose deleted
   * keys fall back to another record, such as a range alias inheriting from its parent.
   */
  revivable?: boolean;
}

/**
 * Retrieves by key are strict: a batch containing any vanished key rejects with
 * NotFound instead of returning the survivors. Splits a rejected batch in halves until
 * each vanished key stands alone, so a few vanished keys cost a few requests, not one
 * per cached key.
 */
const fetchSurvivors = async <Key extends record.Key, Value extends state.State>(
  fetch: NonNullable<TableParams<Key, Value>["fetch"]>,
  keys: Key[],
): Promise<Array<Keyed<Key, Value>>> => {
  try {
    return await fetch(keys);
  } catch (exc) {
    if (!NotFoundError.matches(exc)) throw errors.fromUnknown(exc);
  }
  if (keys.length === 1) return [];
  const mid = Math.ceil(keys.length / 2);
  const halves = await Promise.all([
    fetchSurvivors(fetch, keys.slice(0, mid)),
    fetchSurvivors(fetch, keys.slice(mid)),
  ]);
  return halves.flat();
};

const DEFAULT_FETCH_DEBOUNCE = TimeSpan.milliseconds(10);

const closedError = (cause?: unknown): DisconnectedError => {
  const err = new DisconnectedError("The client is closed.");
  err.cause = cause;
  return err;
};

// "confirmed" tombstones a key the Core no longer holds. "local" tombstones a key this
// client deleted ahead of the change stream.
type Removal = "evict" | "confirmed" | "local";

/** What the table tracks for each live entry beside its value. */
interface Meta {
  /** The touch sequence at the last read or write, which orders what trim drops. */
  touch: number;
  /** Set while the last write was a fetch that found a record new to the table. */
  discovered: boolean;
}

/**
 * Owns the base of a key, the value that writes build on, when a table shows other
 * content on top of it, such as edits that have not reached the Core.
 */
export interface Interceptor<Key extends record.Key, Value> {
  /** Returns the base of the key, or undefined when the key shows its base. */
  base: (key: Key) => Value | undefined;
  /** Takes a written value as the base of its key and returns the value to show. */
  apply: (key: Key, value: Value) => Value;
}

/**
 * The sole owner of one resource's record content and tombstones. Everything else in
 * the cache holds keys into it: answers store key lists, dispatch bookkeeping rides
 * entry deletion, and consumers assemble content at read time. An equal-value set
 * announces nothing: writers echoing state the table already holds (server echoes of
 * optimistic writes, idempotent backfills) are silenced by the equality check rather
 * than by writer identity.
 */
export class Table<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> {
  private readonly entries = new Map<Key, Value>();
  private readonly tombstones = new Map<Key, Deleted<Value>>();
  // Locally deleted keys whose delete the change stream has not delivered. The Core
  // committed every set the stream delivers for them before that delete.
  private readonly pendingDeletes = new Set<Key>();
  private readonly broadSubscribers = new Set<TableSubscriber<Key, Value>>();
  private readonly batchSubscribers = new Set<BatchSubscriber<Key, Value>>();
  private readonly keyedSubscribers = new Map<Key, Set<TableSubscriber<Key, Value>>>();
  private batching: TableEvent<Key, Value>[] | null = null;
  private readonly onError: (error: Error) => void;
  private readonly equal: (a: Value, b: Value, key: Key) => boolean;
  private readonly fetchEntries?: (keys: Key[]) => Promise<Array<Keyed<Key, Value>>>;
  private readonly hydrateMode: HydrateMode;
  private readonly revivable: boolean;
  private closed = false;
  private readonly fetchBatcher: scheduler.Batcher<
    Key[],
    Array<Keyed<Key, Value>>
  > | null;
  private readonly indexes: Array<LookupIndex<Key, Value>>;
  private interceptor: Interceptor<Key, Value> | null = null;
  private writeSeq = 0;
  private resetSeq = 0;
  private changes = 0;
  // Holds a stamp for present and tombstoned entries, so a fetch from before a write
  // never restores the older record.
  private readonly writeStamps = new Map<Key, Stamp>();
  // The write stamps of the keys that trim dropped. A trim forgets each one once no
  // fetch in flight predates it.
  private readonly dropped = new Map<Key, Stamp>();
  // The stamps of the fetches in flight, oldest first, since the write sequence only
  // grows.
  private readonly inFlight: Stamp[] = [];
  private readonly holders = new Set<Holder<Key>>();
  private readonly meta = new Map<Key, Meta>();
  private touchSeq = 0;
  // The touch sequence when the last trim ran. A trim keeps every entry touched since,
  // since a fetch in flight may not yet hold the records it just wrote.
  private trimmedAt = 0;

  constructor({
    onError,
    equal = deep.equal,
    fetch,
    hydrate = "set",
    fetchDebounce = DEFAULT_FETCH_DEBOUNCE,
    indexes = [],
    revivable = false,
  }: TableParams<Key, Value>) {
    this.onError = onError;
    this.equal = equal;
    this.fetchEntries = fetch;
    this.hydrateMode = hydrate;
    this.revivable = revivable;
    this.indexes = [...indexes];
    this.fetchBatcher =
      fetch == null
        ? null
        : new scheduler.Batcher({
            interval: fetchDebounce,
            exec: async (requests) => {
              const keys = new Set<Key>();
              requests.forEach(({ req }) => req.forEach((key) => keys.add(key)));
              let fetched: Array<Keyed<Key, Value>>;
              try {
                fetched = await fetch(Array.from(keys));
              } catch (exc) {
                const strict =
                  NotFoundError.matches(exc) || AccessDeniedError.matches(exc);
                if (!strict || requests.length === 1 || this.closed)
                  throw errors.fromUnknown(exc);
                // A strict fetch rejects the whole batch when any caller's key has
                // vanished or is denied. Refetch per caller so each settles exactly as
                // its own request would have, keeping the batch transparent.
                await Promise.all(
                  requests.map(async ({ req, resolve, reject }) => {
                    try {
                      const mine = new Set(req);
                      resolve((await fetch(req)).filter(({ key }) => mine.has(key)));
                    } catch (exc) {
                      reject(exc);
                    }
                  }),
                );
                return;
              }
              // The window's fetch carries other callers' keys too; each caller
              // hydrates only the entries it asked for.
              requests.forEach(({ req, resolve }) => {
                const mine = new Set(req);
                resolve(fetched.filter(({ key }) => mine.has(key)));
              });
            },
          });
  }

  private applySet(key: Key, value: Value, discovered: boolean): void {
    this.entries.set(key, value);
    const meta = this.meta.get(key);
    if (meta == null) this.meta.set(key, { touch: ++this.touchSeq, discovered });
    else {
      meta.touch = ++this.touchSeq;
      meta.discovered = discovered;
    }
    for (const index of this.indexes) index.set(key, value);
  }

  private applyDelete(key: Key): void {
    this.entries.delete(key);
    this.writeStamps.delete(key);
    this.meta.delete(key);
    for (const index of this.indexes) index.delete(key);
  }

  /** Returns the keys whose records a holder or a keyed subscriber needs. */
  held(): Set<Key> {
    const held = new Set<Key>(this.keyedSubscribers.keys());
    this.holders.forEach((holder) => {
      for (const key of holder.held()) held.add(key);
    });
    return held;
  }

  /**
   * Keeps the records the holder needs from {@link trim}.
   * @returns A destructor that releases them.
   */
  hold(holder: Holder<Key>): destructor.Destructor {
    this.holders.add(holder);
    return () => this.holders.delete(holder);
  }

  /** The number of live entries. */
  get size(): number {
    return this.entries.size;
  }

  /**
   * Drops the least recently read or written entries until at most size remain. Skips
   * every entry that a holder or a keyed subscriber needs, and every entry read or
   * written since the last trim. Tells each holder what dropped. Drops silently: no
   * event, no tombstone, and no new version, since no subscriber holds a dropped
   * entry. A read of a dropped key fetches it again.
   * @returns the dropped keys.
   */
  trim(size: number): Key[] {
    const over = this.entries.size - size;
    const since = this.trimmedAt;
    this.trimmedAt = this.touchSeq;
    const oldest = this.inFlight.at(0) ?? Infinity;
    this.dropped.forEach((stamp, key) => {
      if (stamp <= oldest) this.dropped.delete(key);
    });
    if (over <= 0) return [];
    const held = this.held();
    const touchOf = (key: Key): number => this.meta.get(key)?.touch ?? 0;
    const dropped = Array.from(this.entries.keys())
      .filter((key) => !held.has(key) && touchOf(key) <= since)
      .sort((a, b) => touchOf(a) - touchOf(b))
      .slice(0, over);
    if (dropped.length === 0) return dropped;
    dropped.forEach((key) => {
      const stamp = this.writeStamps.get(key) ?? this.dropped.get(key);
      this.applyDelete(key);
      if (stamp != null && stamp > oldest) this.dropped.set(key, stamp);
    });
    const set = new Set(dropped);
    this.holders.forEach((holder) => holder.drop?.(set));
    return dropped;
  }

  /**
   * Registers a secondary index and backfills it from the live entries, for a domain
   * that owns an index's meaning but not the table it reads.
   * @returns the index.
   */
  index<I extends LookupIndex<Key, Value>>(index: I): I {
    this.entries.forEach((value, key) => index.set(key, value));
    this.indexes.push(index);
    return index;
  }

  /**
   * Hands the base of each key to the interceptor. A setter receives the base, the
   * table shows what the interceptor returns, and a rollback restores the base that the
   * write replaced. A table takes one interceptor.
   * @throws {Error} if the table already has an interceptor.
   */
  intercept(interceptor: Interceptor<Key, Value>): void {
    if (this.interceptor != null)
      throw new Error("The table already has an interceptor");
    this.interceptor = interceptor;
  }

  private base(key: Key): Value | undefined {
    return this.interceptor?.base(key) ?? this.entries.get(key);
  }

  /**
   * Writes the value as the base of the key and shows the result if it changed or if
   * forced.
   */
  private show(key: Key, value: Value, discovered = false, forced = false): void {
    const next = this.interceptor == null ? value : this.interceptor.apply(key, value);
    const shown = this.entries.get(key);
    if (shown != null && !forced && this.equal(next, shown, key)) return;
    this.applySet(key, next, discovered);
    this.notify(
      discovered
        ? { variant: "set", key, value: next, discovered }
        : { variant: "set", key, value: next },
    );
  }

  private setOne(
    key: Key,
    value: state.SetArg<Value | undefined>,
    fetched = false,
  ): destructor.Destructor | undefined {
    const prev = this.base(key);
    const next = state.executeSetter(value, prev);
    if (next == null) return undefined;
    // A fetch reads what the Core holds: a record new to the table is no change there.
    const discovered = fetched && this.status(key) === "unknown";
    // Lists skip a discovery, so the first change after one shows even when equal: it
    // can be the stream set of a record that the Core just created.
    const forced = !fetched && this.meta.get(key)?.discovered === true;
    if (prev != null && !forced && this.equal(next, prev, key)) return undefined;
    const prevTombstone = this.tombstones.get(key);
    const prevStamp = this.writeStamps.get(key);
    this.tombstones.delete(key);
    this.show(key, next, discovered, forced);

    return () => {
      if (prev === undefined) {
        this.applyDelete(key);
        if (prevTombstone != null) this.tombstones.set(key, prevTombstone);
        if (prevStamp != null) this.writeStamps.set(key, prevStamp);
        this.notify({ variant: "delete", key });
      } else this.show(key, prev);
    };
  }

  /**
   * Sets the value for the given key, or applies a setter to the previous value. A
   * setter producing null/undefined, or a value equal to the current one, is a no-op.
   * @returns A rollback that undoes the set.
   */
  set(key: Key, value: state.SetArg<Value | undefined>): destructor.Destructor;
  /**
   * Sets every given keyed value, keying each entry by its key property.
   * @returns A rollback that undoes the inserted values in reverse order.
   */
  set(values: Keyed<Key, Value> | Array<Keyed<Key, Value>>): destructor.Destructor;
  set(
    keyOrValues: Key | Keyed<Key, Value> | Array<Keyed<Key, Value>>,
    value?: state.SetArg<Value | undefined>,
  ): destructor.Destructor {
    if (typeof keyOrValues !== "object") {
      const rollback = this.setOne(keyOrValues as Key, value) ?? destructor.NOOP;
      this.stampWrite(keyOrValues as Key);
      return rollback;
    }
    const values = array.toArray(keyOrValues);
    const rollback = this.setMany(values);
    values.forEach(({ key }) => this.stampWrite(key));
    return rollback;
  }

  private setMany(
    values: Array<Keyed<Key, Value>>,
    fetched = false,
  ): destructor.Destructor {
    const rollbacks: destructor.Destructor[] = [];
    this.batch(() =>
      values.forEach((val) => {
        const rollback = this.setOne(val.key, val, fetched);
        if (rollback != null) rollbacks.push(rollback);
      }),
    );
    return () => this.batch(() => rollbacks.reverse().forEach((r) => r()));
  }

  /**
   * Counts a write that left the key's value as it was, so a fetch that started before
   * this call cannot overwrite the key.
   */
  stampWrite(key: Key): void {
    if (this.entries.has(key)) this.writeStamps.set(key, ++this.writeSeq);
  }

  private writtenSince(key: Key, since: Stamp): boolean {
    const stamp = this.writeStamps.get(key) ?? this.dropped.get(key) ?? 0;
    return this.resetSeq > since || stamp > since;
  }

  /**
   * Runs a fetch that writes into the table. Hands it a stamp of the current point in
   * the write history to pass to {@link ingest}, so its older records undo neither a
   * write nor a reset that lands while it is in flight. Holds the stamp until the fetch
   * settles.
   * @returns what the fetch returns.
   */
  async track<T>(fetch: (since: Stamp) => Promise<T>): Promise<T> {
    const since = this.writeSeq;
    this.inFlight.push(since);
    try {
      return await fetch(since);
    } finally {
      this.inFlight.splice(this.inFlight.indexOf(since), 1);
    }
  }

  /**
   * Counts the changes the table has delivered to its subscribers, plus resets. An
   * unchanged version means no entry was set, ingested, deleted, or reset in between.
   */
  version(): number {
    return this.changes;
  }

  private setIfAbsent(
    values: Array<Keyed<Key, Value>>,
    fetched: boolean,
  ): destructor.Destructor {
    const rollbacks: destructor.Destructor[] = [];
    this.batch(() =>
      values.forEach((val) => {
        if (this.entries.has(val.key)) return;
        const rollback = this.setOne(val.key, val, fetched);
        if (rollback != null) rollbacks.push(rollback);
      }),
    );
    return () => this.batch(() => rollbacks.reverse().forEach((r) => r()));
  }

  /**
   * Writes fetched records into the table under its declared hydrate mode, or the given
   * one: "set" overwrites entries, "if-absent" leaves existing entries as they are. A
   * tombstoned key is skipped, as is a key written after the given stamp: the fetch may
   * predate the delete or the write, and only a {@link set} revives a deleted record.
   * On a revivable table, a key deleted before the given stamp also revives, unless its
   * local delete waits for its stream delete.
   * @returns A rollback that undoes the entries this call wrote.
   */
  ingest(
    values: Keyed<Key, Value> | Array<Keyed<Key, Value>>,
    options?: IngestOptions,
  ): destructor.Destructor;
  /**
   * Writes one fetched value under the given key, by the same rules.
   * @returns A rollback that undoes the entry, if this call wrote it.
   */
  ingest(key: Key, value: Value, options?: IngestOptions): destructor.Destructor;
  ingest(
    keyOrValues: Key | Keyed<Key, Value> | Array<Keyed<Key, Value>>,
    valueOrOptions?: Value | IngestOptions,
    options?: IngestOptions,
  ): destructor.Destructor {
    if (typeof keyOrValues !== "object") {
      const key = keyOrValues as Key;
      const { mode = this.hydrateMode, since, changed = false } = options ?? {};
      let rollback = destructor.NOOP;
      if (
        this.ingestible(key, since) &&
        (mode !== "if-absent" || !this.entries.has(key))
      )
        rollback =
          this.setOne(key, valueOrOptions as Value, !changed) ?? destructor.NOOP;
      this.touch(key);
      return rollback;
    }
    const {
      mode = this.hydrateMode,
      since,
      changed = false,
    }: IngestOptions = (valueOrOptions as IngestOptions | undefined) ?? {};
    const values = array.toArray(keyOrValues);
    const arr = values.filter(({ key }) => this.ingestible(key, since));
    const rollback =
      mode === "if-absent"
        ? this.setIfAbsent(arr, !changed)
        : this.setMany(arr, !changed);
    values.forEach(({ key }) => this.touch(key));
    return rollback;
  }

  // Orders the entries that trim drops. A fetch touches each record it ingests, so a
  // trim keeps the record until the fetch composes its answer, even when the record
  // left the entry as it was.
  private touch(key: Key): void {
    const meta = this.meta.get(key);
    if (meta != null) meta.touch = ++this.touchSeq;
  }

  private ingestible(key: Key, since?: Stamp): boolean {
    if (this.pendingDeletes.has(key)) return false;
    if (since != null && this.writtenSince(key, since)) return false;
    return !this.tombstones.has(key) || (this.revivable && since != null);
  }

  /**
   * Tombstones the entries of the given keys, which a fetch that started at the given
   * stamp found gone. Skips a key the table does not hold or wrote after the stamp.
   */
  ingestDeletes(keys: Key[], since: Stamp): void {
    const vanished = keys.filter(
      (key) => this.entries.has(key) && !this.writtenSince(key, since),
    );
    if (vanished.length > 0) this.remove(vanished, "confirmed");
  }

  /** Returns every entry in the table, or every entry the filter accepts. */
  get(filter?: (value: Value) => boolean): Value[];
  get(key: Key): Value | undefined;
  get(keys: Key[]): Value[];
  get(keys?: Key | Key[] | ((value: Value) => boolean)): Value | Value[] | undefined {
    if (keys === undefined) return Array.from(this.entries.values());
    if (typeof keys === "function")
      return Array.from(this.entries.values()).filter(keys);
    if (Array.isArray(keys)) {
      const values: Value[] = [];
      for (const key of keys) {
        const value = this.entries.get(key);
        if (value == null) continue;
        this.touch(key);
        values.push(value);
      }
      return values;
    }
    const value = this.entries.get(keys);
    if (value != null) this.touch(keys);
    return value;
  }

  has(key: Key): boolean {
    return this.entries.has(key);
  }

  /** Returns the key of every live entry. */
  keys(): Key[] {
    return Array.from(this.entries.keys());
  }

  /**
   * Returns the presence of the given key: present, tombstoned, or unknown.
   * Cache-internal surface: consumed by the query machinery, not domain code.
   */
  status(key: Key): EntryStatus {
    if (this.entries.has(key)) return "present";
    if (this.tombstones.has(key)) return "tombstoned";
    return "unknown";
  }

  /**
   * Returns the tombstone for the given key, or undefined if none exists.
   * Cache-internal surface: consumed by the query machinery, not domain code.
   */
  getTombstone(key: Key): Deleted<Value> | undefined {
    return this.tombstones.get(key);
  }

  /**
   * Resolves the given keys to records: serves cached entries and fetches the misses
   * through the table's fetch, hydrating results under the declared mode. With refresh,
   * every key is fetched regardless of presence, cached entries the fetch omits are
   * tombstoned, and a record new to the table is a change. A key written or deleted
   * while the fetch was in flight keeps that write either way. Returns the table's
   * entries for the found keys in input order, deduplicated; keys the cluster no longer
   * has are omitted. Tables without a fetch serve cached entries only.
   */
  async retrieve(keys: Key[], opts: { refresh?: boolean } = {}): Promise<Value[]> {
    const { fetchBatcher } = this;
    if (fetchBatcher != null) {
      const misses =
        opts.refresh === true ? keys : keys.filter((key) => !this.entries.has(key));
      if (misses.length > 0) {
        if (this.closed) throw closedError();
        await this.track(async (since) => {
          const fetched = await fetchBatcher.enqueue(misses);
          if (opts.refresh === true) {
            // A refresh is authoritative for its keys: cached entries the fetch omitted
            // vanished from the cluster and are tombstoned.
            const present = new Set<Key>(fetched.map(({ key }) => key));
            const vanished = misses.filter(
              (key) =>
                !present.has(key) &&
                this.entries.has(key) &&
                !this.writtenSince(key, since),
            );
            this.batch(() => {
              if (vanished.length > 0) this.remove(vanished, "confirmed");
              if (fetched.length > 0)
                this.ingest(fetched, { mode: "set", since, changed: true });
            });
          } else if (fetched.length > 0) this.ingest(fetched, { since });
        });
      }
    }
    const seen = new Set<Key>();
    const results: Value[] = [];
    for (const key of keys) {
      if (seen.has(key)) continue;
      seen.add(key);
      const entry = this.get(key);
      if (entry != null) results.push(entry);
    }
    return results;
  }

  /**
   * Deletes entries and notifies subscribers. Deleted values are retained as tombstones
   * until a subsequent set for the key. {@link mirrorSet} skips a deleted key until the
   * change stream delivers its delete.
   * @param key - The key(s) to delete or a filter function
   * @returns A rollback that restores the deleted entries, then fetches them again: an
   * entry misses every change made while it was deleted.
   */
  delete(
    key: Key | Key[] | ((value: Value, key: Key) => boolean),
  ): destructor.Destructor {
    return this.remove(key, "local");
  }

  /**
   * Removes entries and notifies subscribers, retaining no corpses: the undo of a write
   * the record never survived, not a deletion. A reader finds the key unknown rather
   * than deleted.
   * @returns A rollback that restores the removed entries
   */
  evict(key: Key | Key[]): destructor.Destructor {
    return this.remove(key, "evict");
  }

  /**
   * Applies a set that the change stream delivered, as {@link set} does. Skips the key
   * while a local delete of it waits for its stream delete, since the Core committed
   * the set before that delete. Cache-internal surface: called by mirror listeners.
   */
  mirrorSet(key: Key, value: state.SetArg<Value | undefined>): void {
    if (this.pendingDeletes.has(key)) return;
    this.set(key, value);
  }

  /**
   * Applies deletes that the change stream delivered, tombstoning the entries the table
   * holds. Cache-internal surface: called by mirror listeners.
   */
  mirrorDelete(keys: Key[]): void {
    keys.forEach((key) => this.pendingDeletes.delete(key));
    this.remove(keys, "confirmed");
  }

  /**
   * Stops {@link mirrorSet} skipping keys whose local delete waits for its stream
   * delete. Called when the change stream opens or reopens, since a stream never
   * delivers what the Core sent before it opened. Cache-internal surface.
   */
  clearPendingDeletes(): void {
    this.pendingDeletes.clear();
  }

  private remove(
    key: Key | Key[] | ((value: Value, key: Key) => boolean),
    removal: Removal,
  ): destructor.Destructor {
    const toDelete: Array<{ key: Key; value?: Value }> = [];

    if (typeof key === "function")
      this.entries.forEach((value, k) => {
        if (key(value, k)) toDelete.push({ key: k, value: this.base(k) });
      });
    else
      array.toArray(key).forEach((k) => toDelete.push({ key: k, value: this.base(k) }));

    this.batch(() =>
      toDelete.forEach(({ key: k, value }) => {
        this.applyDelete(k);
        if (removal !== "evict" && value != null) {
          this.tombstones.set(k, new Deleted(value, TimeStamp.now()));
          // The Core sends no delete for a key it already lost. Its mark then holds
          // until the stream reopens, or until a reconcile finds the key gone.
          if (removal === "local") this.pendingDeletes.add(k);
        }
        // A repeat marks the delete as committed, so it also counts as a write.
        if (removal !== "evict" && this.tombstones.has(k))
          this.writeStamps.set(k, ++this.writeSeq);
        this.notify({ variant: "delete", key: k });
      }),
    );

    return () => {
      const restored: Key[] = [];
      this.batch(() =>
        toDelete.forEach(({ key: k, value }) => {
          if (value == null) return;
          this.tombstones.delete(k);
          this.pendingDeletes.delete(k);
          this.show(k, value);
          this.stampWrite(k);
          restored.push(k);
        }),
      );
      if (removal !== "evict") this.refetch(restored);
    };
  }

  /**
   * Rejects every fetch in progress and every later fetch with a
   * {@link DisconnectedError}. A running reconcile fails with it once its fetch
   * settles. The table keeps its entries.
   */
  close(): void {
    this.closed = true;
    this.fetchBatcher?.close(closedError());
  }

  /**
   * Fetches the keys again in the background and overwrites their entries. Reports a
   * failure other than a lost connection or a vanished record to the table's error
   * sink.
   */
  refetch(keys: Key[]): void {
    if (this.fetchBatcher == null || keys.length === 0) return;
    this.retrieve(keys, { refresh: true }).catch((exc: unknown) => {
      // The reconcile after the next reconnect fetches them, and a delete tombstones a
      // vanished one.
      if (isConnectionError(exc) || NotFoundError.matches(exc)) return;
      this.onError(new Error("failed to refetch entries", { cause: exc }));
    });
  }

  /**
   * Re-checks every cached entry against the cluster through the table's fetch:
   * refreshes entries that still exist and tombstones entries that vanished. A no-op
   * for empty tables and tables without a fetch.
   */
  async reconcile(): Promise<void> {
    const { fetchEntries } = this;
    if (fetchEntries == null) return;
    const keys = this.keys();
    if (keys.length === 0) return;
    await this.track(async (since) => {
      const values = await fetchSurvivors(async (keys: Key[]) => {
        if (this.closed) throw closedError();
        try {
          const fetched = await fetchEntries(keys);
          if (this.closed) throw closedError();
          return fetched;
        } catch (exc) {
          throw this.closed ? closedError(exc) : errors.fromUnknown(exc);
        }
      }, keys);
      const present = new Set<Key>(values.map(({ key }) => key));
      const missing = keys.filter((k) => !present.has(k));
      // A missing key deleted here during the fetch gets no delete from the Core.
      missing.forEach((k) => this.pendingDeletes.delete(k));
      const vanished = missing.filter((k) => !this.writtenSince(k, since));
      this.batch(() => {
        if (vanished.length > 0) this.remove(vanished, "confirmed");
        if (values.length > 0) this.ingest(values, { mode: "set", since });
      });
    });
  }

  /**
   * Discards every entry and tombstone without notifying subscribers. Called only by
   * the cache's reset, which resets the answer spaces itself: per-entry delete events
   * would masquerade as deletions of live records. Fetches in flight when the reset
   * runs discard their results.
   */
  reset(): void {
    this.changes++;
    this.resetSeq = ++this.writeSeq;
    this.entries.clear();
    this.tombstones.clear();
    this.pendingDeletes.clear();
    this.writeStamps.clear();
    this.dropped.clear();
    this.meta.clear();
    for (const index of this.indexes) index.reset();
  }

  /**
   * Subscribes to entry changes delivered per batched write: one call carrying every
   * event of the batch, in order, after all of the batch's entries have applied.
   * Cache-internal surface: consumed by the query machinery, not domain code;
   * per-record consumers use {@link subscribe}.
   * @returns A destructor that unsubscribes.
   */
  subscribeBatch(subscriber: BatchSubscriber<Key, Value>): destructor.Destructor {
    this.batchSubscribers.add(subscriber);
    return () => this.batchSubscribers.delete(subscriber);
  }

  /**
   * Subscribes to entry changes. With a key, only changes to that entry fire.
   * @returns A destructor that unsubscribes.
   */
  subscribe(subscriber: TableSubscriber<Key, Value>, key?: Key): destructor.Destructor {
    if (key == null) {
      this.broadSubscribers.add(subscriber);
      return () => this.broadSubscribers.delete(subscriber);
    }
    let held = this.keyedSubscribers.get(key);
    if (held == null) {
      held = new Set();
      this.keyedSubscribers.set(key, held);
    }
    held.add(subscriber);
    return () => {
      held.delete(subscriber);
      // Guarded on identity so a stale destructor cannot drop a set a later
      // subscription re-created under the same key.
      if (held.size === 0 && this.keyedSubscribers.get(key) === held)
        this.keyedSubscribers.delete(key);
    };
  }

  /**
   * Runs fn with event delivery deferred: writes apply to entries immediately, and
   * every event they produce is delivered as one batch when the outermost batch exits.
   * Delivery stays synchronous, inside this call. Nested batches join the outermost
   * one. Events flush even when fn throws, since the entries they describe were already
   * applied.
   */
  batch<T>(fn: () => T): T {
    if (this.batching != null) return fn();
    const events: TableEvent<Key, Value>[] = [];
    this.batching = events;
    try {
      return fn();
    } finally {
      this.batching = null;
      if (events.length > 0) this.flush(events);
    }
  }

  private notify(event: TableEvent<Key, Value>) {
    this.changes++;
    if (this.batching != null) {
      this.batching.push(event);
      return;
    }
    this.flush([event]);
  }

  private flush(events: TableEvent<Key, Value>[]) {
    for (const event of events) {
      this.deliver(this.keyedSubscribers.get(event.key), event);
      this.deliver(this.broadSubscribers, event);
    }
    for (const subscriber of this.batchSubscribers)
      try {
        subscriber(events);
      } catch (exc) {
        this.onError(new Error("failed to notify table subscriber", { cause: exc }));
      }
  }

  private deliver(
    subscribers: Iterable<TableSubscriber<Key, Value>> | undefined,
    event: TableEvent<Key, Value>,
  ) {
    if (subscribers == null) return;
    for (const subscriber of subscribers)
      try {
        subscriber(event);
      } catch (exc) {
        this.onError(new Error("failed to notify table subscriber", { cause: exc }));
      }
  }
}

/**
 * A mirror listener declaration: keeps the owning table current from one stream
 * channel. Built with {@link createSetListener}, {@link createDeleteListener}, or
 * {@link createFetchListener}; bound to its table by the cache.
 */
export interface ListenerSpec<
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
> {
  /** Binds the declaration to the owning table as a raw channel listener. */
  bind: (table: Table<Key, Value>) => Listener;
}

/**
 * Declares that the channel broadcasts records to mirror into the table. By default the
 * parsed record keys itself (`changed.key`) and is stored as-is; `key` derives the
 * entry key, `value` transforms the record and may merge with the previous entry
 * (return null/undefined to skip the write). Without `key`, a record that fails the
 * schema is fetched again by its key.
 */
export const createSetListener = <
  Z extends z.ZodType,
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
>(
  channel: string,
  schema: Z,
  opts: {
    key?: (changed: z.output<Z>) => Key;
    value?: (changed: z.output<Z>, prev: Value | undefined) => Value | null | undefined;
  } = {},
): ListenerSpec<Key, Value> => {
  const {
    key = (changed) => (changed as record.Keyed<Key>).key,
    value = (changed) => changed as Value,
  } = opts;
  return {
    bind: (table): Listener<Z> => ({
      channel,
      schema,
      onChange: (changed) => {
        table.batch(() =>
          changed.forEach((c) =>
            table.mirrorSet(key(c), (prev) => value(c, prev) ?? undefined),
          ),
        );
      },
      // A derived key can read fields that an invalid record lacks.
      onInvalid:
        opts.key != null
          ? undefined
          : (raw) => {
              const keyed = keyedZ.safeParse(raw);
              if (keyed.success) table.refetch([keyed.data.key as Key]);
            },
    }),
  };
};

/**
 * Declares that the channel broadcasts deletions to mirror into the table. By default
 * the parsed value is the entry key; `key` derives it instead.
 */
export const createDeleteListener = <
  Z extends z.ZodType,
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
>(
  channel: string,
  schema: Z,
  opts: { key?: (changed: z.output<Z>) => Key } = {},
): ListenerSpec<Key, Value> => {
  const { key = (changed) => changed as Key } = opts;
  return {
    bind: (table): Listener<Z> => ({
      channel,
      schema,
      onChange: (changed) => {
        table.mirrorDelete(changed.map((c) => key(c)));
      },
    }),
  };
};

/**
 * Declares that the channel announces keys whose records changed. Announced keys the
 * table holds are refetched through the table's fetch and overwrite their entries. The
 * rest are ignored, so a client never fetches a record it did not read.
 */
export const createFetchListener = <
  Z extends z.ZodType<Key | Key[]>,
  Key extends record.Key = record.Key,
  Value extends state.State = state.State,
>(
  channel: string,
  schema: Z,
): ListenerSpec<Key, Value> => ({
  bind: (table): Listener<Z> => ({
    channel,
    schema,
    // A refetch in the background keeps later frames off the network round trip.
    onChange: (changed) =>
      table.refetch(
        changed.flatMap((c) => array.toArray(c)).filter((k) => table.has(k)),
      ),
  }),
});

export const partialUpdate = <Key extends record.Key, Value extends object>(
  table: Table<Key, Value>,
  key: Key,
  value: Partial<Value>,
): destructor.Destructor =>
  table.set(key, (p) => (p == null ? undefined : { ...p, ...value }));
