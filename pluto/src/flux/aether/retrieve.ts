// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, query, type Synnax as Client } from "@synnaxlabs/client";
import { deep, type destructor } from "@synnaxlabs/x";

export interface RetrieveParams<Query extends query.Params> {
  client: Client;
  query: Query;
}

/**
 * A domain query definition: fetch = `retrieve`, live updates = `onChange`, snapshot =
 * `getCached`, all delegating to the domain client's read surface. React-free, so one
 * definition serves both the React and aether flux bindings.
 */
export interface Definition<Query extends query.Params, Data extends query.Data> {
  name: string;
  retrieve: (params: RetrieveParams<Query>) => Promise<Data>;
  onChange?: (
    params: RetrieveParams<Query>,
    handler: query.ChangeHandler<Data>,
  ) => destructor.Destructor;
  getCached?: (params: RetrieveParams<Query>) => query.Cached<Data> | undefined;
}

export interface RetrieveArgs<Query extends query.Params, Data extends query.Data> {
  definition: Definition<Query, Data>;
  /** Fires whenever the answer changes: a live value, a deletion, or undefined when
   * nothing is cached. Invalidation gaps are held, not forwarded. */
  onChange?: (result: query.Cached<Data> | undefined) => void;
  /** Receives fetch failures. Not-found resolves to an undefined answer instead. */
  onError?: (error: Error) => void;
}

/**
 * Holds one query's answer live for an aether component: seeds synchronously from the
 * domain cache, fetches on a miss, subscribes for changes, and pushes updates through
 * `onChange`. Call {@link update} from `afterUpdate` and {@link close} from
 * `afterDelete`. Repeated updates with an equal query and client are no-ops.
 */
export class Retrieve<Query extends query.Params, Data extends query.Data> {
  private readonly definition: Definition<Query, Data>;
  private readonly notify?: (result: query.Cached<Data> | undefined) => void;
  private readonly onError?: (error: Error) => void;
  private disconnect: destructor.Destructor | null = null;
  private client: Client | null = null;
  private query: Query | null = null;
  private generation = 0;
  private result: query.Cached<Data> | undefined;

  constructor({ definition, onChange, onError }: RetrieveArgs<Query, Data>) {
    this.definition = definition;
    this.notify = onChange;
    this.onError = onError;
  }

  /** The live answer, or undefined while absent, deleted, or not yet fetched. */
  get value(): Data | undefined {
    return query.isLive<Data>(this.result) ? this.result : undefined;
  }

  /** The raw answer: live, deleted, or undefined when nothing is cached. */
  get cached(): query.Cached<Data> | undefined {
    return this.result;
  }

  /**
   * Points the observer at a query. Tears down the previous subscription, seeds from
   * the cache, and fetches on a miss. A null client clears the answer.
   */
  update(client: Client | null, q: Query): void {
    if (client === this.client && this.query != null && deep.equal(q, this.query))
      return;
    this.teardown();
    this.client = client;
    this.query = q;
    if (client == null) return this.set(undefined);
    const params = { client, query: q };
    const { onChange, getCached } = this.definition;
    if (onChange != null)
      this.disconnect = onChange(params, (result) => {
        // An undefined push is an invalidation the maintained entry will repair.
        // Hold the previous answer until the repair lands.
        if (result === undefined && this.result !== undefined) return;
        this.set(result);
      });
    const cached = getCached?.(params);
    if (cached !== undefined) return this.set(cached);
    void this.fetch(params);
  }

  /** Refetches the current query, superseding any fetch in flight. */
  refetch(): void {
    if (this.client == null || this.query == null) return;
    void this.fetch({ client: this.client, query: this.query });
  }

  /** Never rejects: a failure is routed to onError and a stale generation dropped. */
  private async fetch(params: RetrieveParams<Query>): Promise<void> {
    const generation = ++this.generation;
    try {
      const value = await this.definition.retrieve(params);
      if (generation !== this.generation) return;
      this.set(value);
    } catch (error) {
      if (generation !== this.generation) return;
      if (NotFoundError.matches(error)) return this.set(undefined);
      this.onError?.(error as Error);
    }
  }

  private set(result: query.Cached<Data> | undefined): void {
    if (result === this.result) return;
    this.result = result;
    this.notify?.(result);
  }

  private teardown(): void {
    this.generation++;
    this.disconnect?.();
    this.disconnect = null;
  }

  /** Stops the subscription and drops in-flight fetches. Safe to call twice. */
  close(): void {
    this.teardown();
    this.client = null;
    this.query = null;
  }
}
