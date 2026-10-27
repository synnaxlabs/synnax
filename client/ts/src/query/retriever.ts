// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, filter, type record, zod } from "@synnaxlabs/x";
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
import { checkForMultipleOrNoResults } from "@/util/retrieve";

/** Query fields only the server can evaluate, shared by most request shapes. */
const DEFAULT_SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

const isObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

/** @returns the one member of value that is not nullish, or null. */
const soleMember = (value: Record<string, unknown>): [string, unknown] | null => {
  const set = Object.entries(value).filter(([, v]) => v != null);
  return set.length === 1 ? set[0] : null;
};

/** @returns the list of a filter that only matches field against a list, or null. */
const whereIn = (where: unknown, field: string): unknown[] | null => {
  if (!isObject(where)) return null;
  const line = soleMember(where);
  if (line?.[0] !== field || !isObject(line[1])) return null;
  const op = soleMember(line[1]);
  return op?.[0] === "in" && Array.isArray(op[1]) ? op[1] : null;
};

/**
 * Returns the keys of a request that addresses keys and nothing else: a `where` that
 * only matches keys in a list, or a non-empty `keys` field, with every other field
 * nullish. Such requests resolve through the table's fetch primitive instead of the
 * request fetch. Returns null for any other request.
 */
const keysOnly = (query: unknown): unknown[] | null => {
  if (!isObject(query)) return null;
  const field = soleMember(query);
  if (field?.[0] === "where") return whereIn(field[1], "key");
  if (field?.[0] !== "keys" || !Array.isArray(field[1]) || field[1].length === 0)
    return null;
  return field[1];
};

/**
 * The params a request schema accepts. A recursive `where` schema infers no input type,
 * so `where` takes its output type, which the schema also accepts.
 */
type RequestInput<Z extends z.ZodType> =
  z.input<Z> extends infer I
    ? I extends unknown
      ? "where" extends keyof I
        ? Omit<I, "where"> & Pick<Extract<z.output<Z>, { where?: unknown }>, "where">
        : I
      : never
    : never;

/** The request fields that an option of a retrieve may set. */
type OptionField<Z extends z.ZodType> =
  RequestInput<Z> extends infer I
    ? I extends readonly unknown[]
      ? never
      : Exclude<keyof I, "where"> & string
    : never;

/**
 * A part of a retrieve: a condition of the type C, or an option that sets a field of
 * the request schema Z.
 */
export type Part<C extends filter.Condition<string>, Z extends z.ZodType> = filter.Part<
  C,
  OptionField<Z>
>;

/**
 * The parts of a retrieve that returns a list. A retriever with no condition type takes
 * none.
 */
export type ListParts<C extends filter.Condition<string>, Z extends z.ZodType> = [
  C,
] extends [never]
  ? [never]
  : Array<Part<C, Z>>;

/** The parts of a retrieve that returns one record: parts that end with `one()`. */
export type OneParts<C extends filter.Condition<string>, Z extends z.ZodType> = [
  C,
] extends [never]
  ? [never]
  : [...Array<Part<C, Z>>, filter.One];

/** True for params that are a bare record key rather than a query object. */
const isBareKey = (params: unknown): params is record.Key =>
  typeof params === "string" || typeof params === "number";

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
    /** Rule 2: whether a record satisfies the query. Pure; no network. */
    matches?: (record: V, query: z.output<Z>) => boolean;
    /** Overrides {@link DEFAULT_SERVER_FIELDS} for this request shape. */
    serverFields?: readonly string[];
    /** Rule 3: whether only the server can evaluate the query, such as a regex. */
    serverComputed?: (query: z.output<Z>) => boolean;
    /** Foreign tables whose events affect this space's answers. */
    watch?: Array<Watch<z.output<Z>, K>>;
  };
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
 * export class Client extends query.Retriever<
 *   label.Condition,
 *   typeof requestZ,
 *   Key,
 *   Label
 * > {
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
 */
export abstract class Retriever<
  C extends filter.Condition<string>,
  Z extends z.ZodType<Params>,
  K extends record.Key,
  V extends Keyed<K>,
  D extends Data = V,
  SingleInput = { key: K },
  SingleQuery extends Params = K,
> {
  private readonly name: string;
  private readonly singleSpace: Retrieves<Params, D | D[]>;
  private readonly requestSpace: Retrieves<Params, D | D[]>;
  private readonly trySingle: (params: unknown) => { query: SingleQuery } | null;
  private readonly normalizeRequest: (params: unknown) => z.output<Z>;
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
    this.normalizeRequest = (p) => zod.parse(schema, p, { label: `${name} query` });
    if (single != null) {
      this.singleSpace = single.space as Retrieves<Params, D | D[]>;
      const singleSchema = single.schema;
      // Params carrying `key` that fail the single schema are malformed single queries;
      // falling through would silently fetch every record.
      this.trySingle = (p) => {
        const res = singleSchema.safeParse(p);
        if (res.success) return { query: res.data };
        if (isBareKey(p) || (typeof p === "object" && p !== null && "key" in p)) {
          const cause = new zod.ParseError({
            issues: res.error.issues,
            input: p,
            label: `${name} single query params`,
            cause: res.error,
          });
          throw new ValidationError(
            `${name} params address a single key but do not match the ` +
              `single-query schema.\n${cause.message}`,
            { cause },
          );
        }
        return null;
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
        return typeof p === "object" && p !== null && "key" in p
          ? { query: (p as { key: K }).key as Params as SingleQuery }
          : null;
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
    const single = this.trySingle(params);
    const route: [Retrieves<Params, D | D[]>, Params] =
      single != null
        ? [this.singleSpace, single.query]
        : [this.requestSpace, this.normalizeRequest(params)];
    if (cacheable) this.routes.set(params, route);
    return route;
  }

  /**
   * Reads the records that match every condition in parts, with each option applied.
   * With no parts, reads every record. Serves the cache when the answer is fresh,
   * fetching otherwise.
   *
   * Reads the record the params address, or every record matching them. A bare key is
   * shorthand for `{ key }`; a bare key list is accepted by domains whose request
   * schema unions in {@link keysRequestZ} or {@link keyListZ}.
   * @throws {NotFoundError} if a single-record query matches nothing or the record was
   * deleted, or if parts end with `one()` and no record matches.
   * @throws {MultipleFoundError} if parts end with `one()` and more than one record
   * matches.
   * @throws {ValidationError} if params contain `key` (or are a bare key) but fail the
   * single-query schema.
   */
  retrieve(...parts: OneParts<C, Z>): Promise<D>;
  retrieve(...parts: ListParts<C, Z>): Promise<D[]>;
  retrieve(params: K | SingleInput, options?: FetchOptions): Promise<D>;
  retrieve(params: K[] | RequestInput<Z>, options?: FetchOptions): Promise<D[]>;
  async retrieve(...args: unknown[]): Promise<D | D[]> {
    if (this.isParts(args)) return await this.retrieveParts(args);
    const [params, options] = args as [unknown, FetchOptions | undefined];
    const [space, query] = this.route(params);
    return await space.retrieve(query, options);
  }

  /** @returns whether args are the parts of a retrieve, rather than params. */
  protected isParts(args: readonly unknown[]): args is Array<Part<C, Z> | filter.One> {
    return args.length === 0 || filter.isPart(args[0]);
  }

  /**
   * Reads the records that parts select, or one record when parts end with `one()`.
   * A subclass that overrides retrieve passes its parts here.
   */
  protected async retrieveParts(
    parts: Array<Part<C, Z> | filter.One>,
  ): Promise<D | D[]> {
    const one = parts.at(-1) instanceof filter.One;
    const request = filter.toRequest(
      (one ? parts.slice(0, -1) : parts) as Array<Part<C, Z>>,
    );
    const records = (await this.requestSpace.retrieve(
      this.normalizeRequest(request),
    )) as D[];
    if (!one) return records;
    checkForMultipleOrNoResults(this.name, request, records, true);
    return records[0];
  }

  /**
   * Subscribes to changes in the cached answer to the given query. The handler fires on
   * every change or deletion. Returns a destructor that unsubscribes.
   */
  onChange(params: K | SingleInput, handler: ChangeHandler<D>): destructor.Destructor;
  onChange(
    params: K[] | RequestInput<Z>,
    handler: ChangeHandler<D[]>,
  ): destructor.Destructor;
  onChange(
    params: K | K[] | SingleInput | RequestInput<Z>,
    handler: ChangeHandler<D> | ChangeHandler<D[]>,
  ): destructor.Destructor {
    const [space, query] = this.route(params);
    return space.onChange(query, handler as ChangeHandler<D | D[]>);
  }

  /**
   * Returns the cached answer to the given query without touching the network, or
   * undefined when nothing is cached. May be stale for unsubscribed queries.
   */
  getCached(params: K | SingleInput): Cached<D> | undefined;
  getCached(params: K[] | RequestInput<Z>): Cached<D[]> | undefined;
  getCached(
    params: K | K[] | SingleInput | RequestInput<Z>,
  ): Cached<D> | Cached<D[]> | undefined {
    const [space, query] = this.route(params);
    return space.getCached(query) as Cached<D> | Cached<D[]> | undefined;
  }
}
