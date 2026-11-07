// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, type record, zod } from "@synnaxlabs/x";
import z from "zod";

import { NotFoundError, ValidationError } from "@/errors";
import { type Cache } from "@/query/cache";
import {
  type Cached,
  type ChangeHandler,
  type Retrieves,
  type Watch,
} from "@/query/query";
import { type Keyed, type Table } from "@/query/table";
import { type Data, type FetchOptions, type Params } from "@/query/types";
import { checkForMultipleOrNoResults, singleQueryError } from "@/util/retrieve";

/** Query fields only the server can evaluate, shared by most request shapes. */
const DEFAULT_SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

const isObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

/** @returns the one member of value that is not nullish, or null. */
const soleMember = (value: Record<string, unknown>): [string, unknown] | null => {
  const set = Object.entries(value).filter(([, v]) => v != null);
  return set.length === 1 ? set[0] : null;
};

/** @returns the value of a filter that only applies op to field, or undefined. */
const whereOp = (where: unknown, field: string, op: string): unknown => {
  if (!isObject(where)) return undefined;
  const line = soleMember(where);
  if (line?.[0] !== field || !isObject(line[1])) return undefined;
  const member = soleMember(line[1]);
  return member?.[0] === op ? member[1] : undefined;
};

/** @returns the list of a filter that only matches field against a list, or null. */
const whereIn = (where: unknown, field: string): unknown[] | null => {
  const list = whereOp(where, field, "in");
  return Array.isArray(list) ? list : null;
};

/** @returns the keys of a request whose only field is a `where` on keys, or null. */
const whereKeys = (query: unknown): unknown[] | null => {
  if (!isObject(query)) return null;
  const field = soleMember(query);
  return field?.[0] === "where" ? whereIn(field[1], "key") : null;
};

/**
 * @returns the key of a request whose only field is a `where` on one key, or undefined.
 */
const whereKey = (query: unknown): unknown => {
  if (!isObject(query)) return undefined;
  const field = soleMember(query);
  return field?.[0] === "where" ? whereOp(field[1], "key", "eq") : undefined;
};

/**
 * Returns the keys of a request that addresses keys and nothing else: a `where` that
 * only matches keys in a list, or a non-empty `keys` field, with every other field
 * nullish. Such requests resolve through the table's fetch primitive instead of the
 * request fetch. Returns null for any other request.
 */
const keysOnly = (query: unknown): unknown[] | null => {
  const keys = whereKeys(query);
  if (keys != null || !isObject(query)) return keys;
  const field = soleMember(query);
  if (field?.[0] !== "keys" || !Array.isArray(field[1]) || field[1].length === 0)
    return null;
  return field[1];
};

/**
 * The query objects a request schema accepts that return a list. It leaves out a bare
 * key list, which each overload takes on its own, a plain `key` or `name` value, and
 * `one: true`, which each return one record. A recursive `where` schema infers no input
 * type, so `where` takes its output type, which the schema also accepts.
 */
export type RequestInput<Z extends z.ZodType> =
  z.input<Z> extends infer I
    ? I extends readonly unknown[]
      ? never
      : { one?: false } & ("where" extends keyof I
          ? Omit<I, "where"> & Pick<Extract<z.output<Z>, { where?: unknown }>, "where">
          : Omit<I, "key" | "name"> & {
              [F in ("key" | "name") & keyof I]?: Exclude<
                I[F],
                F extends "key" ? record.Key : string
              >;
            })
    : never;

/**
 * The query objects a request schema accepts that return one record: those with a plain
 * `key` or `name` value beside any other fields, and those that set `one`. Neither
 * takes a limit or an offset, since a page could hide a second match.
 */
export type OneInput<Z extends z.ZodType> =
  | (z.input<Z> extends infer I
      ? I extends readonly unknown[]
        ? never
        : "where" extends keyof I
          ? never
          : {
              [F in ("key" | "name") & keyof I]: Omit<I, F | "limit" | "offset"> & {
                [P in F]: Extract<I[P], P extends "key" ? record.Key : string>;
              } & { limit?: never; offset?: never };
            }[("key" | "name") & keyof I]
      : never)
  | (RequestInput<Z> extends infer R
      ? R extends unknown
        ? Omit<R, "one" | "limit" | "offset"> & {
            one: true;
            limit?: never;
            offset?: never;
          }
        : never
      : never);

/** True for params that are a bare record key rather than a query object. */
const isBareKey = (params: unknown): params is record.Key =>
  typeof params === "string" || typeof params === "number";

/**
 * True for params that return one record: a bare key or name, an object with a plain
 * `key` or `name` value, or an object that sets `one`. A list or an operator on those
 * fields returns a list.
 */
export const addressesOne = (params: unknown): boolean =>
  isBareKey(params) ||
  (isObject(params) &&
    (params.one === true || isBareKey(params.key) || typeof params.name === "string"));

/**
 * @returns params without the `one` option.
 * @throws {ValidationError} if params return one record beside a limit or an offset.
 */
export const dropOne = <P>(params: P): P extends unknown ? Omit<P, "one"> : never => {
  if (!isObject(params)) return params as P extends unknown ? Omit<P, "one"> : never;
  if (addressesOne(params) && (params.limit != null || params.offset != null))
    throw new ValidationError(
      "a query for one record takes no limit or offset, since a page could hide a " +
        "second match",
    );
  if (!("one" in params)) return params as P extends unknown ? Omit<P, "one"> : never;
  const { one: _, ...rest } = params;
  return rest as P extends unknown ? Omit<P, "one"> : never;
};

/** True for params that are a bare key or an object with a plain `key` value. */
const addressesKey = (params: unknown): boolean =>
  isBareKey(params) || (isObject(params) && isBareKey(params.key));

/**
 * Request schema arm reading a bare key list as `{ keys }`. Union it onto a domain's
 * request schema to opt that domain into the shorthand:
 * `retrieveRequestZ.or(keyListZ(keyZ))`.
 */
export const keyListZ = <K extends z.ZodType<record.Key>>(key: K) =>
  z.array(key).transform((keys) => ({ keys }));

/** @returns a request whose `where` matches the records with the given keys. */
export const keysRequest = <K extends record.Key>(keys: K[]) => ({
  where: { key: { in: keys } },
});

/**
 * Request schema arm reading a bare key list as a `where` on the keys. Union it onto a
 * domain's request schema: `z.union([keysRequestZ(keyZ), retrieveRequestZ])`.
 */
export const keysRequestZ = <K extends z.ZodType<record.Key>>(key: K) =>
  z.array(key).transform(keysRequest);

interface ComposeProp<V, D> {
  /**
   * Per-record enrichment applied to every answer, receiving the normalized query the
   * record answers. Defaults to identity.
   */
  compose: (record: V, query: Params) => D;
}

/** Optional only when a record is a valid answer as-is (V assignable to D): the
 *  identity default is unsound for any narrower D. */
type ComposeParam<V, D> = [V] extends [D]
  ? Partial<ComposeProp<V, D>>
  : ComposeProp<V, D>;

/**
 * Declaration of a domain client's read surface. The single space is derived from the
 * table and its fetch: `{ key }` params resolve one record, with deletion flipping the
 * answer to deleted. Declare `single` only when the domain's single query is richer
 * than a key.
 */
export type RetrieverParams<
  Z extends z.ZodType<Params>,
  K extends record.Key,
  V extends Keyed<K>,
  D extends Data = V,
  SingleInput = { key: K },
  SingleQuery extends Params = K,
> = ComposeParam<V, D> & {
  /** Resource name used in error messages, e.g. "range". */
  name: string;
  /** The table owning this domain's record content. */
  table: Table<K, V>;
  /** The space answering every non-single query shape. */
  request: {
    /**
     * Parses and canonicalizes request params so equivalent queries hash identically;
     * its output type is the space's query type.
     */
    schema: Z;
    /**
     * Fetches matching records from the cluster. Keys-only requests never reach it;
     * they resolve through the table's fetch. Results hydrate the table under its
     * declared mode.
     */
    fetch: (query: z.output<Z>, options?: FetchOptions) => Promise<V[]>;
    /**
     * Rule 2: whether a record satisfies the query. Pure; no network. Null when the
     * client cannot decide the record, which refetches the query.
     */
    matches?: (record: V, query: z.output<Z>) => boolean | null;
    /** Overrides {@link DEFAULT_SERVER_FIELDS} for this request shape. */
    serverFields?: readonly string[];
    /** Rule 3: whether only the server can evaluate the query, such as a regex. */
    serverComputed?: (query: z.output<Z>) => boolean;
    /** Foreign tables whose events affect this space's answers. */
    watch?: Array<Watch<z.output<Z>, K>>;
  };
  /**
   * Reads a bare key, a bare name, or a list of either into a request in place of the
   * request schema. A bare name then returns one record. Domains whose records have
   * names set it to their generated `shortcutZ`.
   */
  shortcut?: z.ZodType<Params>;
  /** Custom single space for domains whose single query is richer than a key. */
  single?: {
    /**
     * Discriminates and canonicalizes single params: parse success routes the query to
     * the single space, and the output is the space's query. Object schemas must reject
     * unknown keys so request params fall through. Union in a bare-key arm to accept
     * `retrieve(key)`.
     */
    schema: z.ZodType<SingleQuery, SingleInput | record.Key>;
    /** The space answering single queries, built via the cache. */
    space: Retrieves<SingleQuery, D>;
  };
};

/**
 * A domain client's cached read surface, routing each query to the single or request
 * answer space. The arity of the result follows the query: params addressing one record
 * yield a record, every other shape yields an array.
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
 * Anything evaluated eagerly inside `super` must come from a local, since `this` is
 * unavailable until it returns. Closures may reference `this`: an arrow captures it
 * lexically and does not read it until called.
 *
 * The read surface is not designed for overriding: a domain with more query kinds
 * exposes them as named {@link Retrieves} members instead.
 *
 * `Shortcut` is the bare value that addresses one record: a key, or a name too for a
 * domain that sets a shortcut and whose keys are not strings.
 */
export abstract class Retriever<
  Z extends z.ZodType<Params>,
  K extends record.Key,
  V extends Keyed<K>,
  D extends Keyed<K> = V,
  SingleInput = { key: K },
  SingleQuery extends Params = K,
  Shortcut extends record.Key = K,
> {
  private readonly name: string;
  private readonly singleSpace: Retrieves<Params, D | D[]>;
  private readonly oneSpace: Retrieves<Params, D | D[]>;
  private readonly requestSpace: Retrieves<Params, D | D[]>;
  private readonly trySingle: (params: unknown) => { query: SingleQuery } | null;
  private readonly normalizeRequest: (params: unknown) => z.output<Z>;
  private readonly readShortcut: ((params: unknown) => Params) | null;
  private readonly routes = new WeakMap<object, [Retrieves<Params, D | D[]>, Params]>();

  constructor(
    cache: Cache,
    params: RetrieverParams<Z, K, V, D, SingleInput, SingleQuery>,
  ) {
    const {
      name,
      table,
      request,
      compose = (record: V) => record as unknown as D,
      shortcut,
      single,
    } = params;
    this.name = name;
    const {
      schema,
      fetch,
      matches,
      serverFields = DEFAULT_SERVER_FIELDS,
      serverComputed,
      watch,
    } = request;
    this.requestSpace = cache.queries<z.output<Z>, D[], K, V>({
      name,
      table,
      fetch: async (query, options) => {
        const keys = keysOnly(query);
        if (keys != null) return (await table.retrieve(keys as K[])).map((r) => r.key);
        const since = table.stamp();
        const records = await fetch(query, options);
        table.ingest(records, { since });
        return records.map((r) => r.key);
      },
      compose: (records, q) => records.map((r) => compose(r, q)),
      keysOf: (query) => keysOnly(query) as K[] | null,
      matches,
      serverFields,
      serverComputed,
      watch,
    }) as Retrieves<Params, D | D[]>;
    this.oneSpace = cache.queries<z.output<Z>, D, K, V>({
      name,
      table,
      fetch: async (query, options) => {
        const since = table.stamp();
        const records = await fetch(query, options);
        table.ingest(records, { since });
        const { where, searchTerm } = isObject(query) ? query : {};
        checkForMultipleOrNoResults(
          name,
          {
            where: isObject(where) ? where : undefined,
            searchTerm: typeof searchTerm === "string" ? searchTerm : undefined,
          },
          records,
          true,
        );
        return records.map((r) => r.key);
      },
      compose: (records, q) => compose(records[0], q),
      matches,
      serverFields,
      serverComputed,
      watch,
      single: true,
    }) as Retrieves<Params, D | D[]>;
    const singleSchema = single?.schema;
    this.normalizeRequest = (p) => {
      const res = schema.safeParse(p);
      if (res.success) return res.data;
      // A key the single schema rejects is the likely cause, and the request union
      // reports only its strict arm, which calls the key an unexpected field.
      const one =
        singleSchema != null && addressesKey(p) ? singleSchema.safeParse(p) : null;
      if (one?.success === false) throw singleQueryError(name, p, one.error);
      throw new zod.ParseError({
        issues: res.error.issues,
        input: p,
        label: `${name} query`,
        cause: res.error,
      });
    };
    this.readShortcut =
      shortcut == null
        ? null
        : (p) => zod.parse(shortcut, p, { label: `${name} query` });
    if (single != null) {
      this.singleSpace = single.space as Retrieves<Params, D | D[]>;
      const singleSchema = single.schema;
      this.trySingle = (p) => {
        const res = singleSchema.safeParse(p);
        return res.success ? { query: res.data } : null;
      };
    } else {
      // The default path binds SingleQuery = K: `{ key }` params resolve through the
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
      }) as Retrieves<Params, D | D[]>;
      this.trySingle = (p) => {
        if (isBareKey(p)) return { query: p as Params as SingleQuery };
        if (!isObject(p) || !isBareKey(p.key) || soleMember(p)?.[0] !== "key")
          return null;
        return { query: p.key as Params as SingleQuery };
      };
    }
  }

  /**
   * Routes params to their answer space, memoized per params object identity.
   * {@link Params} is readonly, so a given object always routes identically; repeat
   * reads of the same object skip schema validation entirely.
   */
  private route(params: unknown): [Retrieves<Params, D | D[]>, Params] {
    const cacheable = typeof params === "object" && params !== null;
    if (cacheable) {
      const held = this.routes.get(params);
      if (held !== undefined) return held;
    }
    const route = this.resolve(params);
    if (cacheable) this.routes.set(params, route);
    return route;
  }

  private resolve(params: unknown): [Retrieves<Params, D | D[]>, Params] {
    if (this.readShortcut != null && (isBareKey(params) || Array.isArray(params))) {
      const request = this.readShortcut(params);
      if (Array.isArray(params))
        return [this.requestSpace, this.normalizeRequest(request)];
      if (whereKey(request) === undefined)
        return [this.oneSpace, this.normalizeRequest(request)];
    }
    const one = addressesOne(params);
    const request = dropOne(params);
    const single = this.trySingle(request);
    if (single != null) return [this.singleSpace, single.query];
    return [one ? this.oneSpace : this.requestSpace, this.normalizeRequest(request)];
  }

  /**
   * Reads the record the params address, or every record matching them. Serves the
   * cache when the answer is fresh, fetching otherwise. A plain `key` or `name` value
   * returns one record, even beside other fields, and a list or an operator on them
   * returns a list. Any other query returns a list unless it sets `one`. A bare key is
   * shorthand for `{ key }`, and in a domain with a shortcut, a bare name is shorthand
   * for `{ name }`. A domain accepts a bare key list when it sets a shortcut or its
   * request schema unions in {@link keysRequestZ} or {@link keyListZ}. Read through
   * either of the first two, the list returns each record once, in the order its key
   * first appears. A filter on keys returns the records that exist.
   * @throws {NotFoundError} if a single-record query matches nothing, the record was
   * deleted, or a key in a bare key list has no record.
   * @throws {MultipleFoundError} if a single-record query matches more than one record.
   * @throws {ValidationError} if params set `one` beside a limit or an offset.
   * @throws {zod.ParseError} if no schema of the domain accepts the params.
   */
  retrieve(
    params: Shortcut | SingleInput | OneInput<Z>,
    options?: FetchOptions,
  ): Promise<D>;
  retrieve(params: Shortcut[] | RequestInput<Z>, options?: FetchOptions): Promise<D[]>;
  async retrieve(
    params: Shortcut | Shortcut[] | SingleInput | OneInput<Z> | RequestInput<Z>,
    options?: FetchOptions,
  ): Promise<D | D[]> {
    const [space, query] = this.route(params);
    const res = await space.retrieve(query, options);
    const keys = Array.isArray(params) ? (whereKeys(query) as K[] | null) : null;
    return keys == null ? res : this.inKeyOrder(keys, res as D[]);
  }

  private inKeyOrder(keys: K[], records: D[]): D[] {
    const byKey = new Map<K, D>(records.map((r) => [r.key, r]));
    const unique = [...new Set(keys)];
    const missing = unique.filter((k) => !byKey.has(k));
    if (missing.length === 1)
      throw new NotFoundError(`${this.name} with key ${missing[0]} not found`);
    if (missing.length > 1)
      throw new NotFoundError(`${this.name} with keys ${missing.join(", ")} not found`);
    return unique.map((k) => byKey.get(k) as D);
  }

  /**
   * Subscribes to changes in the cached answer to the given query. The handler fires on
   * every change or deletion. Returns a destructor that unsubscribes.
   */
  onChange(
    params: Shortcut | SingleInput | OneInput<Z>,
    handler: ChangeHandler<D>,
  ): destructor.Destructor;
  onChange(
    params: Shortcut[] | RequestInput<Z>,
    handler: ChangeHandler<D[]>,
  ): destructor.Destructor;
  onChange(
    params: Shortcut | Shortcut[] | SingleInput | OneInput<Z> | RequestInput<Z>,
    handler: ChangeHandler<D> | ChangeHandler<D[]>,
  ): destructor.Destructor {
    const [space, query] = this.route(params);
    return space.onChange(query, handler as ChangeHandler<D | D[]>);
  }

  /**
   * Returns the cached answer to the given query without touching the network, or
   * undefined when nothing is cached. May be stale for unsubscribed queries.
   */
  getCached(params: Shortcut | SingleInput | OneInput<Z>): Cached<D> | undefined;
  getCached(params: Shortcut[] | RequestInput<Z>): Cached<D[]> | undefined;
  getCached(
    params: Shortcut | Shortcut[] | SingleInput | OneInput<Z> | RequestInput<Z>,
  ): Cached<D> | Cached<D[]> | undefined {
    const [space, query] = this.route(params);
    return space.getCached(query) as Cached<D> | Cached<D[]> | undefined;
  }
}
