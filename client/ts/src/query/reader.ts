// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";

import { type Cached, type ChangeHandler } from "@/query/query";
import { type Data, type Query } from "@/query/types";

/** The read surface of one answer space: fetch, subscribe, snapshot. */
export interface Retrieves<Q extends Query, D extends Data> {
  retrieve: (query: Q) => Promise<D>;
  onChange: (query: Q, handler: ChangeHandler<D>) => destructor.Destructor;
  getCached: (query: Q) => Cached<D> | undefined;
}

export interface RetrieverParams<
  SingleParams,
  MultiParams,
  SingleQuery extends Query,
  MultiQuery extends Query,
  V extends Data,
> {
  /** The space answering queries that address exactly one record. */
  single: Retrieves<SingleQuery, V>;
  /** The space answering every other query shape. */
  request: Retrieves<MultiQuery, V[]>;
  /** Whether the params address exactly one record. */
  isSingle: (params: SingleParams | MultiParams) => boolean;
  /** Canonicalizes single params so equivalent queries hash identically. */
  normalizeSingle: (params: SingleParams) => SingleQuery;
  /** Canonicalizes multi params. Usually the request schema's parse. */
  normalizeRequest: (params: MultiParams) => MultiQuery;
}

/**
 * A domain client's cached read surface, routing each query to the single or
 * multi answer space. The arity of the result follows the query: params
 * addressing one record yield a record, every other shape yields an array.
 *
 * Extend it and call `super` with the two spaces:
 *
 * ```ts
 * export class Client extends cache.Reader<Single, Multi, Key, Request, Policy> {
 *   constructor(cache: cache.Cache) {
 *     const store = createTable(cache);
 *     super({ single: cache.answers({ table: store, ... }), request: ..., ... });
 *     this.store = store;
 *   }
 * }
 * ```
 *
 * Anything evaluated eagerly inside `super` must come from a local, since
 * `this` is unavailable until it returns. Closures may reference `this`: an
 * arrow captures it lexically and does not read it until called.
 *
 * A domain needing more than routing overrides the method and delegates to
 * `super` for the cached path.
 */
export abstract class Retriever<
  SingleParams,
  MultiParams,
  SingleQuery extends Query,
  MultiQuery extends Query,
  V extends Data,
> {
  private readonly reads: RetrieverParams<
    SingleParams,
    MultiParams,
    SingleQuery,
    MultiQuery,
    V
  >;

  constructor(
    reads: RetrieverParams<SingleParams, MultiParams, SingleQuery, MultiQuery, V>,
  ) {
    this.reads = reads;
  }

  /**
   * Reads the record the params address, or every record matching them.
   * Serves the cache when the answer is fresh, fetching otherwise.
   */
  retrieve(params: SingleParams): Promise<V>;
  retrieve(params: MultiParams): Promise<V[]>;
  async retrieve(params: SingleParams | MultiParams): Promise<V | V[]> {
    const { space, query } = this.route(params);
    return await space.retrieve(query);
  }

  /**
   * Subscribes to changes in the cached answer to the given query. The handler
   * fires on every change or deletion. Returns a destructor that unsubscribes.
   */
  onChange(params: SingleParams, handler: ChangeHandler<V>): destructor.Destructor;
  onChange(params: MultiParams, handler: ChangeHandler<V[]>): destructor.Destructor;
  onChange(
    params: SingleParams | MultiParams,
    handler: ChangeHandler<V> | ChangeHandler<V[]>,
  ): destructor.Destructor {
    const { space, query } = this.route(params);
    return space.onChange(query, handler as ChangeHandler<V | V[]>);
  }

  /**
   * Returns the cached answer to the given query without touching the network,
   * or undefined when nothing is cached. May be stale for unsubscribed
   * queries.
   */
  getCached(params: SingleParams): Cached<V> | undefined;
  getCached(params: MultiParams): Cached<V[]> | undefined;
  getCached(params: SingleParams | MultiParams): Cached<V> | Cached<V[]> | undefined {
    const { space, query } = this.route(params);
    return space.getCached(query) as Cached<V> | Cached<V[]> | undefined;
  }

  // Routing is the one place the single/multi correlation is known.
  // TypeScript cannot carry it across separate parameters, so the casts that
  // would otherwise appear in every domain client live only here.
  private route(params: SingleParams | MultiParams): {
    space: Retrieves<SingleQuery | MultiQuery, V | V[]>;
    query: SingleQuery | MultiQuery;
  } {
    type Either = Retrieves<SingleQuery | MultiQuery, V | V[]>;
    const { single, request, isSingle, normalizeSingle, normalizeRequest } = this.reads;
    return isSingle(params)
      ? { space: single as Either, query: normalizeSingle(params as SingleParams) }
      : { space: request as Either, query: normalizeRequest(params as MultiParams) };
  }
}
