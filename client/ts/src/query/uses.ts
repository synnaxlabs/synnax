// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, type record, scheduler, sync, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

import { isConnectionError, NotFoundError } from "@/errors";
import type { Cache } from "@/query/cache";
import { type Retrieves, type Trigger, trigger } from "@/query/query";
import { fetchSurvivors, type Keyed, type Table } from "@/query/table";
import type { Params } from "@/query/types";

/** A retrieve that can ask for the uses of the records it selects. */
type Request = { includeUses?: boolean };

/** A retrieve of one record by key. */
type Single<K extends record.Key> = Request & { key: K };

/** A record whose retrieve can fill the resources that use it. */
type Used<K extends record.Key> = Keyed<K> & object & { uses?: unknown[] };

/** Joins a burst of changes on the channels of uses, such as the moves of a drag. */
const REFETCH_DEBOUNCE = TimeSpan.milliseconds(100);

/** The deletes of records that land while fetches of uses run. */
interface Deletes<K extends record.Key> {
  inFlight: number;
  seq: number;
  at: Map<K, number>;
}

/** Config for {@link Uses}. */
export interface UsesConfig<
  K extends record.Key,
  R extends Used<K>,
  Q extends Params & Request,
> {
  /** Names the records in errors, such as "secret". */
  name: string;
  cache: Cache;
  /** The table of the records. It never holds their uses. */
  table: Table<K, R>;
  /** The schema of a record key. */
  key: z.ZodType<K, K>;
  /** Sends a retrieve to the Core. */
  retrieve: (request: Q) => Promise<R[]>;
  /** Builds a retrieve, with uses, of the records whose keys match the filter. */
  request: (key: { eq: K } | { in: K[] }) => Q;
  /**
   * The channels that announce changes to the records that can use one. A change on
   * one, or a reconnect, refetches the uses that a maintained query watches.
   */
  channels: string[];
}

/**
 * Holds the uses of records, which are the resources that use each one, apart from the
 * records. A query that asks for uses gets them composed onto its records. A change on
 * a channel of the config, or a reconnect, refetches the uses that a maintained query
 * watches and drops the rest.
 */
export class Uses<K extends record.Key, R extends Used<K>, Q extends Params & Request> {
  /** Updates the queries that ask for uses when the uses of a record change. */
  readonly trigger: Trigger<Q | Single<K>, K>;
  /** The single space of a retriever, which takes `{ key, includeUses }`. */
  readonly single: {
    schema: z.ZodType<Single<K>, Single<K> | K>;
    space: Retrieves<Single<K>, R>;
  };
  private readonly cfg: UsesConfig<K, R, Q>;
  private readonly store: Table<K, Used<K>>;
  private readonly deletes: Deletes<K> = { inFlight: 0, seq: 0, at: new Map() };
  private readonly changes = { count: 0 };
  /** The members of each maintained query that asks for uses. */
  private readonly watchers = new Set<() => readonly K[]>();
  private readonly refetch: () => void;

  constructor(cfg: UsesConfig<K, R, Q>) {
    const { name, cache, table, key } = cfg;
    this.cfg = cfg;
    // No table fetch, so the reconcile after a reconnect cannot write uses.
    const store = cache.createTable<K, Used<K>>({
      name: `${name}.uses`,
    });
    store.hold({ held: () => this.watched() });
    // A record created again under a deleted key reads its uses again.
    table.subscribe((event) => {
      if (event.variant !== "delete") return;
      store.evict(event.key);
      if (this.deletes.inFlight > 0) this.deletes.at.set(event.key, ++this.deletes.seq);
    });
    this.store = store;
    // The Core announces no change to uses. Refetches run outside the change stream and
    // one at a time, so an older reply cannot overwrite a newer one. A change that
    // arrives while a refetch waits joins it.
    const mu = sync.createMutex({ waiting: false });
    this.refetch = scheduler.debounce(() => {
      if (mu.waiting) return;
      mu.waiting = true;
      mu.runExclusive(async () => {
        mu.waiting = false;
        await this.refetchWatched();
      }).catch((exc: unknown) => {
        // The refetch after the next reconnect fetches the uses again.
        if (isConnectionError(exc)) return;
        cache.onError(new Error(`failed to refetch ${name} uses`, { cause: exc }));
      });
    }, REFETCH_DEBOUNCE);
    const onChange = (): void => {
      this.changes.count++;
      this.refetch();
    };
    cfg.channels.forEach((channel) =>
      cache.listen({ channel, schema: z.unknown(), onChange }),
    );
    // A reopened stream missed the changes made while it was down.
    cache.onEpoch((epoch) => {
      if (epoch > 1) onChange();
    });
    const base = trigger(
      store,
      (event, q: Q | Single<K>) => (q.includeUses === true ? [event.key] : null),
      { contentOnly: true },
    );
    this.trigger = {
      ...base,
      attach: (params, onEvent, members) => {
        const detach = base.attach(params, onEvent, members);
        if (params.includeUses !== true) return detach;
        this.watchers.add(members);
        return () => {
          detach();
          this.watchers.delete(members);
        };
      },
      covers: (params, members) =>
        params.includeUses !== true || members.every((key) => store.get(key) != null),
    };
    const normalize = ({ key, includeUses }: Single<K>): Single<K> =>
      includeUses === true ? { key, includeUses } : { key };
    this.single = {
      // Drops includeUses when unset so equivalent queries hash identically.
      schema: z
        .union([
          z.strictObject({ key, includeUses: z.boolean().optional() }),
          key.transform((k) => ({ key: k })),
        ])
        .transform(normalize),
      space: cache.queries<Single<K>, R, K, R>({
        name,
        table,
        fetch: async (q) => [await this.fetchSingle(q)],
        compose: (records, q) => this.compose(records[0], q),
        keyOf: (q) => q.key,
        covers: (_, q) => q.includeUses !== true,
        triggers: [this.trigger],
        single: true,
      }),
    };
  }

  /** @returns the record with its uses when the query asks for them. */
  compose(record: R, query: Params): R {
    if ((query as Request).includeUses !== true) return record;
    return { ...record, uses: this.store.get(record.key)?.uses };
  }

  /**
   * @returns whether the query asks for uses of the record that the client does not
   * hold, so that only a fetch can answer it.
   */
  lacks(record: R, query: Q): boolean {
    return query.includeUses === true && this.store.get(record.key) == null;
  }

  /**
   * Records that the records with the keys have no uses, since this client is creating
   * them. A query that asks for uses then lists them with no fetch.
   * @returns a destructor that restores the uses that the client held before.
   */
  created(keys: K[]): destructor.Destructor {
    return this.store.set(keys.map((key) => ({ key, uses: [] })));
  }

  /**
   * Fetches the records that the request selects. A request that asks for uses keeps
   * them apart from the records it returns.
   */
  async fetch(request: Q): Promise<R[]> {
    if (request.includeUses !== true) return await this.cfg.retrieve(request);
    const changes = this.changes.count;
    const records = await this.retrieve(request);
    this.store.set(records.map(({ key, uses }) => ({ key, uses })));
    // A refetch that a change started can miss these keys or land first.
    if (this.changes.count !== changes) this.refetch();
    return records.map((r) => {
      const stripped = { ...r };
      delete stripped.uses;
      return stripped;
    });
  }

  private watched(): Set<K> {
    const watched = new Set<K>();
    this.watchers.forEach((members) => members().forEach((key) => watched.add(key)));
    return watched;
  }

  // Only the refetch mutex runs this, so no reply can put back the uses it drops.
  private async refetchWatched(): Promise<void> {
    const { store, cfg } = this;
    const watched = this.watched();
    const unwatched = store.keys().filter((key) => !watched.has(key));
    if (unwatched.length > 0) store.evict(unwatched);
    const keys = store.keys();
    if (keys.length === 0) return;
    await store.track(async (since) => {
      const records = await fetchSurvivors(
        async (keys: K[]) =>
          (await this.retrieve(cfg.request({ in: keys }))).map(({ key, uses }) => ({
            key,
            uses,
          })),
        keys,
      );
      store.ingest(records, { mode: "set", since });
    });
  }

  // A delete of a record that a retrieve returns makes its uses stale, so the retrieve
  // runs again.
  private async retrieve(request: Q): Promise<R[]> {
    const { deletes } = this;
    const since = deletes.seq;
    deletes.inFlight++;
    let records: R[];
    try {
      records = await this.cfg.retrieve(request);
    } finally {
      deletes.inFlight--;
    }
    const overtaken = records.some(({ key }) => (deletes.at.get(key) ?? 0) > since);
    if (deletes.inFlight === 0) deletes.at.clear();
    return overtaken ? await this.retrieve(request) : records;
  }

  private async fetchSingle({ key, includeUses }: Single<K>): Promise<K> {
    const { name, table, request } = this.cfg;
    const notFound = () => new NotFoundError(`${name} with key ${key} not found`);
    if (includeUses === true)
      return await table.track(async (since) => {
        const records = await this.fetch(request({ eq: key }));
        if (records.length === 0) {
          table.ingestDeletes([key], since);
          throw notFound();
        }
        table.ingest(records, { since });
        return key;
      });
    if ((await table.retrieve([key])).length === 0) throw notFound();
    return key;
  }
}
