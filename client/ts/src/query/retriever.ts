// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, type record, type state } from "@synnaxlabs/x";
import type z from "zod";

import { NotFoundError, ValidationError } from "@/errors";
import { type Cache } from "@/query/cache";
import {
  type Cached,
  type ChangeHandler,
  type Retrieves,
  type WatchEntry,
} from "@/query/query";
import { type Table } from "@/query/table";
import { type Data, type FetchOptions, type Params } from "@/query/types";

/** Query fields only the server can evaluate, shared by most request shapes. */
const DEFAULT_SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

/**
 * True for requests that address keys and nothing else: a non-empty `keys`
 * field with every other field nullish. Such requests resolve through the
 * table's fetch primitive instead of the request fetch.
 */
const isKeysOnly = (query: unknown): query is { keys: unknown[] } => {
  if (typeof query !== "object" || query === null || Array.isArray(query)) return false;
  const { keys } = query as { keys?: unknown };
  if (!Array.isArray(keys) || keys.length === 0) return false;
  return Object.entries(query).every(([k, v]) => k === "keys" || v == null);
};

/**
 * Declaration of a domain client's read surface. The single space is derived
 * from the table and its fetch: `{ key }` params resolve one record, with
 * deletion flipping the answer to deleted. Declare `single` only when the
 * domain's single query is richer than a key.
 */
export interface RetrieverParams<
  Z extends z.ZodType<Params>,
  K extends record.Key,
  V extends state.State & record.Keyed<K>,
  D extends Data = V,
  SP = { key: K },
  SN extends Params = K,
> {
  /** Resource name used in error messages, e.g. "range". */
  name: string;
  /** The table owning this domain's record content. */
  table: Table<K, V>;
  /** The space answering every non-single query shape. */
  request: {
    /**
     * Parses and canonicalizes request params so equivalent queries hash
     * identically; its output type is the space's query type.
     */
    schema: Z;
    /**
     * Fetches matching records from the cluster. Keys-only requests never
     * reach it; they resolve through the table's fetch. Results hydrate the
     * table under its declared mode.
     */
    fetch: (
      query: z.output<Z>,
      options?: FetchOptions,
    ) => Promise<Array<V & record.Keyed<K>>>;
    /** Rule 2: whether a record satisfies the query. Pure; no network. */
    matches?: (record: V, query: z.output<Z>) => boolean;
    /** Overrides {@link DEFAULT_SERVER_FIELDS} for this request shape. */
    serverFields?: readonly string[];
    /** Foreign tables whose events affect this space's answers. */
    watch?: Array<WatchEntry<z.output<Z>, K>>;
  };
  /**
   * Per-record enrichment applied to every answer, receiving the normalized
   * query the record answers. Defaults to identity.
   */
  compose?: (record: V, query: Params) => D;
  /** Custom single space for domains whose single query is richer than a key. */
  single?: {
    /**
     * Discriminates and canonicalizes single params: parse success routes the
     * query to the single space, and the output is the space's query. Object
     * schemas must reject unknown keys so request params fall through.
     */
    schema: z.ZodType<SN, SP>;
    /** The space answering single queries, built via the cache. */
    space: Retrieves<SN, D>;
  };
}

/**
 * A domain client's cached read surface, routing each query to the single or
 * request answer space. The arity of the result follows the query: params
 * addressing one record yield a record, every other shape yields an array.
 *
 * Extend it and call `super(cache, { ... })`:
 *
 * ```ts
 * export class Client extends query.Retriever<typeof requestZ, Key, Label> {
 *   constructor(client: UnaryClient, cache: query.Cache) {
 *     const store = cache.createTable({ ... });
 *     super(cache, { name: "label", table: store, request: { ... } });
 *   }
 * }
 * ```
 *
 * Anything evaluated eagerly inside `super` must come from a local, since
 * `this` is unavailable until it returns. Closures may reference `this`: an
 * arrow captures it lexically and does not read it until called.
 *
 * The read surface is not designed for overriding: a domain with more query
 * kinds exposes them as named {@link Retrieves} members instead.
 */
export abstract class Retriever<
  Z extends z.ZodType<Params>,
  K extends record.Key,
  V extends state.State & record.Keyed<K>,
  D extends Data = V,
  SP = { key: K },
  SN extends Params = K,
> {
  private readonly singleSpace: Retrieves<SN, D>;
  private readonly requestSpace: Retrieves<z.output<Z>, D[]>;
  private readonly trySingle: (params: unknown) => { query: SN } | null;
  private readonly normalizeRequest: (params: unknown) => z.output<Z>;

  constructor(cache: Cache, params: RetrieverParams<Z, K, V, D, SP, SN>) {
    const {
      name,
      table,
      request,
      compose = (record) => record as unknown as D,
      single,
    } = params;
    const {
      schema,
      fetch,
      matches,
      serverFields = DEFAULT_SERVER_FIELDS,
      watch,
    } = request;
    this.requestSpace = cache.queries<z.output<Z>, D[], K, V>({
      name,
      table,
      fetch: async (query, options) => {
        if (isKeysOnly(query))
          return (await table.retrieve(query.keys as K[])).map((r) => r.key);
        const records = await fetch(query, options);
        table.ingest(records);
        return records.map((r) => r.key);
      },
      compose: (records, q) => records.map((r) => compose(r, q)),
      matches,
      serverFields,
      watch,
    });
    this.normalizeRequest = (p) => schema.parse(p);
    if (single != null) {
      this.singleSpace = single.space;
      const singleSchema = single.schema;
      // Params carrying `key` that fail the single schema are malformed
      // single queries; falling through would silently fetch every record.
      this.trySingle = (p) => {
        const res = singleSchema.safeParse(p);
        if (res.success) return { query: res.data };
        if (typeof p === "object" && p !== null && "key" in p)
          throw new ValidationError(
            `${name} params contain 'key' but do not match the single-query ` +
              `schema: ${res.error.issues.map((i) => i.message).join("; ")}`,
          );
        return null;
      };
    } else {
      // The default path binds SN = K: `{ key }` params resolve through the
      // table's fetch. The casts are the one seam the defaults can't prove.
      this.singleSpace = cache.queries<K, D, K, V>({
        name,
        table,
        fetch: async (key) => {
          const records = await table.retrieve([key]);
          if (records.length === 0)
            throw new NotFoundError(`${name} with key ${key} not found`);
          return [key];
        },
        compose: (records, q) => compose(records[0], q),
        keyOf: (query) => query,
        single: true,
      }) as Retrieves<Params, D>;
      this.trySingle = (p) =>
        typeof p === "object" && p !== null && "key" in p
          ? { query: (p as { key: K }).key as Params as SN }
          : null;
    }
  }

  /**
   * Reads the record the params address, or every record matching them.
   * Serves the cache when the answer is fresh, fetching otherwise.
   * @throws {NotFoundError} if a single-record query matches nothing or the
   * record was deleted.
   * @throws {ValidationError} if params contain `key` but fail the
   * single-query schema.
   */
  retrieve(params: SP, options?: FetchOptions): Promise<D>;
  retrieve(params: z.input<Z>, options?: FetchOptions): Promise<D[]>;
  async retrieve(params: SP | z.input<Z>, options?: FetchOptions): Promise<D | D[]> {
    const single = this.trySingle(params);
    if (single != null) return await this.singleSpace.retrieve(single.query, options);
    return await this.requestSpace.retrieve(this.normalizeRequest(params), options);
  }

  /**
   * Subscribes to changes in the cached answer to the given query. The handler
   * fires on every change or deletion. Returns a destructor that unsubscribes.
   */
  onChange(params: SP, handler: ChangeHandler<D>): destructor.Destructor;
  onChange(params: z.input<Z>, handler: ChangeHandler<D[]>): destructor.Destructor;
  onChange(
    params: SP | z.input<Z>,
    handler: ChangeHandler<D> | ChangeHandler<D[]>,
  ): destructor.Destructor {
    const single = this.trySingle(params);
    if (single != null)
      return this.singleSpace.onChange(single.query, handler as ChangeHandler<D>);
    return this.requestSpace.onChange(
      this.normalizeRequest(params),
      handler as ChangeHandler<D[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the network,
   * or undefined when nothing is cached. May be stale for unsubscribed
   * queries.
   */
  getCached(params: SP): Cached<D> | undefined;
  getCached(params: z.input<Z>): Cached<D[]> | undefined;
  getCached(params: SP | z.input<Z>): Cached<D> | Cached<D[]> | undefined {
    const single = this.trySingle(params);
    if (single != null) return this.singleSpace.getCached(single.query);
    return this.requestSpace.getCached(this.normalizeRequest(params));
  }
}
