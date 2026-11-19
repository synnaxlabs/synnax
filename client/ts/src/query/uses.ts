// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record, sync } from "@synnaxlabs/x";
import { z } from "zod";

import { isConnectionError, NotFoundError } from "@/errors";
import { type Cache } from "@/query/cache";
import { type Retrieves, type Trigger, trigger } from "@/query/query";
import { type Keyed, type Table } from "@/query/table";
import { type Params } from "@/query/types";

/** A retrieve that can ask for the uses of the records it selects. */
type Request = { includeUses?: boolean };

/** A retrieve of one record by key. */
type Single<K extends record.Key> = Request & { key: K };

/** A record whose retrieve can fill the resources that use it. */
type Used<K extends record.Key> = Keyed<K> & object & { uses?: unknown };

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
   * one refetches the uses the client holds.
   */
  channels: string[];
}

/**
 * Holds the uses of records, which are the resources that use each one, apart from the
 * records. A query that asks for uses gets them composed onto its records, and a change
 * on a channel of the config refetches them.
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
  private readonly store: Table<K, Pick<R, "key" | "uses">>;
  private readonly deletes: Deletes<K> = { inFlight: 0, seq: 0, at: new Map() };
  private readonly changes = { count: 0 };
  private readonly refetch: () => void;

  constructor(cfg: UsesConfig<K, R, Q>) {
    const { name, cache, table, key } = cfg;
    this.cfg = cfg;
    const store = cache.createTable<K, Pick<R, "key" | "uses">>({
      name: `${name}.uses`,
      fetch: async (keys) =>
        (await this.retrieve(cfg.request({ in: keys }))).map(({ key, uses }) => ({
          key,
          uses,
        })),
    });
    // An answer that holds a record composes its uses too.
    store.hold({ held: () => table.held() });
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
    const mu = sync.newMutex({ waiting: false });
    this.refetch = () => {
      if (mu.waiting) return;
      mu.waiting = true;
      mu.runExclusive(async () => {
        mu.waiting = false;
        await store.reconcile();
      }).catch((exc: unknown) => {
        // The reconcile after the next reconnect fetches the uses again.
        if (isConnectionError(exc)) return;
        cache.onError(new Error(`failed to refetch ${name} uses`, { cause: exc }));
      });
    };
    cfg.channels.forEach((channel) =>
      cache.listen({
        channel,
        schema: z.unknown(),
        onChange: () => {
          this.changes.count++;
          this.refetch();
        },
      }),
    );
    this.trigger = trigger(
      store,
      (event, q: Q | Single<K>) => (q.includeUses === true ? [event.key] : null),
      { contentOnly: true },
    );
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
