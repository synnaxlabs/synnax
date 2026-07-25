// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type primitive, type state } from "@synnaxlabs/x";

/** Options to control async operations. */
export interface FetchOptions {
  /** Optional AbortSignal to cancel the operation */
  signal?: AbortSignal;
}

/** Options for write methods that apply the change locally before the send. */
export interface WriteOptions<T = void> {
  /**
   * Called after the optimistic cache apply and before the network send, with
   * the optimistically applied value. UI layers use it to react to a write
   * (e.g. open a tab) without waiting on the server.
   */
  onOptimistic?: (value: T) => Promise<void> | void;
}

/**
 * Plain-data shape used to address a record in the cache query system. Queries
 * are hashed by {@link hash} to produce stable cache keys, so they must be
 * primitives, arrays of queries, plain string-keyed objects of queries, or
 * class instances implementing {@link primitive.Hashable}. Maps, Sets, Dates,
 * and non-Hashable class instances are rejected at compile time because their
 * fields don't recursively reduce to {@link Params}.
 *
 * Consumer query types must be declared with `type` aliases, not `interface`,
 * because TypeScript interfaces lack an implicit string index signature and
 * fail the `{ readonly [k: string]: Params }` branch.
 */
export type Params =
  | string
  | number
  | bigint
  | boolean
  | null
  | undefined
  | primitive.Hashable
  | readonly Params[]
  | { readonly [key: string]: Params };

/**
 * Shape of values returned and stored by the cache query system. Unlike
 * {@link Params}, data values do not need to be hashable — they may carry
 * arbitrary state including class instances or non-serializable references.
 */
export type Data = state.State;
