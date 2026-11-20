// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { array, filter, zod } from "@synnaxlabs/x";
import z from "zod";

import { actions } from "@/actions";
import type { batch } from "@/batch";
import { createOf, encode, reduceAll } from "@/enumeration/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
} from "@/enumeration/actions.gen";
import {
  type Enumeration,
  enumerationZ,
  type Key,
  keyZ,
  type New,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
} from "@/enumeration/types.gen";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
  shortcutZ,
} from "@/enumeration/where.gen";
import { query } from "@/query";
import { resource } from "@/resource";

export const ACTION_CHANNEL_NAME = "sy_enumeration_action_set";
export const DELETE_CHANNEL_NAME = "sy_enumeration_delete";

const createReqZ = z.object({
  enumerations: enumerationZ.array(),
  dispatchKey: z.string(),
});
const createResZ = z.object({ enumerations: enumerationZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const replaceReqZ = z.object({ from: keyZ, to: keyZ });
const retrieveBuiltinsResZ = z.object({ enumerations: enumerationZ.array() });
const retrieveDependentsReqZ = z.object({ key: keyZ });
const retrieveDependentsResZ = z.object({
  dependents: resource.idZ.array().default(() => []),
});
const emptyZ = z.object({});

const retrieveMultiParamsZ = filter.requestInputZ(
  retrieveRequestZ,
  retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
);

/** Names one enumeration. */
export type RetrieveSingleParams = { key: Key };
/** Everything a retrieval of many enumerations can filter on. */
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;
/** Params for a retrieval of one or many enumerations. */
export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

/** Config for {@link Client}. */
export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
}

/**
 * Creates, reads, changes, and deletes enumerations on a Core. Reach it through
 * `client.enumerations`.
 */
export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Enumeration
> {
  readonly type: string = "enumeration";
  /** The enumeration table. */
  readonly store: query.Table<Key, Enumeration>;
  private readonly unary: UnaryClient;
  private readonly dispatcher: actions.Controller<Key, Enumeration, Action>;

  constructor({ unary, cache }: ClientConfig) {
    const store = cache.createTable<Key, Enumeration>({
      name: "enumerations",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to an enumeration reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Enumeration, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(unary, "/enumeration/dispatch", dispatchReqZ, dispatchResZ),
      reduce: reduceAll,
      encode,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "enumeration",
      table: store,
      shortcut: shortcutZ,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (enumeration, req) => matches(enumeration, req.where),
        serverComputed: (req) => serverComputed(req.where),
      },
    });
    this.unary = unary;
    this.store = store;
    this.dispatcher = dispatcher;
  }

  async create(
    enumeration: New,
    opts?: query.WriteOptions<Enumeration[]>,
  ): Promise<Enumeration>;
  async create(
    enumerations: New[],
    opts?: query.WriteOptions<Enumeration[]>,
  ): Promise<Enumeration[]>;
  async create(
    enumerations: New | New[],
    opts: query.WriteOptions<Enumeration[]> = {},
  ): Promise<Enumeration | Enumeration[]> {
    const isMany = Array.isArray(enumerations);
    const optimistic = array
      .toArray(enumerations)
      .map((e) => zod.parse(enumerationZ, e, { label: "enumeration" }));
    const res = await query.optimistic({
      rollbacks: [this.store.set(optimistic)],
      onOptimistic: () => opts.onOptimistic?.(optimistic),
      commit: async () =>
        await this.dispatcher.create(
          optimistic.map(({ key }) => key),
          async (dispatchKey) =>
            await this.unary.send(
              "/enumeration/create",
              { enumerations: optimistic, dispatchKey },
              createReqZ,
              createResZ,
            ),
        ),
    });
    this.store.set(res.enumerations);
    return isMany ? res.enumerations : res.enumerations[0];
  }

  /**
   * Applies actions to the cached enumeration and sends them to the Core. Returns false
   * without side effects when the enumeration is not cached.
   * @throws {Error} if the enumeration cannot hold an action, before any send.
   * @throws {Error} if the send fails, after the cached enumeration drops the actions.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<boolean> {
    return await this.dispatcher.dispatch(key, array.toArray(actions));
  }

  /** Stages actions that commit atomically as one dispatch. */
  beginTransaction(key: Key): actions.Transaction<Action> {
    return this.dispatcher.transaction(key);
  }

  /**
   * Deletes the enumerations.
   * @throws {Error} if one is built in or a record uses it. The Core then deletes none.
   */
  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [this.store.delete(keysArr)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.unary.send(
          "/enumeration/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyZ,
        ),
    });
    drop();
  }

  /**
   * Moves each record that uses the enumeration from to the enumeration to, then
   * deletes from.
   * @throws {Error} if from is built in or equals to, if from or to does not exist, or
   * if the caller cannot delete from, retrieve to, or update a record that uses from.
   * The Core then changes nothing.
   */
  async replace(from: Key, to: Key, opts: query.WriteOptions = {}): Promise<void> {
    const drop = () => [this.store.delete([from])];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.unary.send(
          "/enumeration/replace",
          { from, to },
          replaceReqZ,
          emptyZ,
        ),
    });
    drop();
  }

  /** Builds a batch entry that deletes the enumerations. */
  deleteRequest(keys: Key | Key[]): batch.Request<z.infer<typeof deleteReqZ>> {
    const payload = { keys: array.toArray(keys) };
    return {
      endpoint: "/enumeration/delete",
      payload,
      apply: () => this.store.delete(payload.keys),
    };
  }

  /**
   * Returns the enumerations that the Core ships, with their shipped values. A stored
   * built-in can differ from its shipped values, because users can edit it.
   */
  async retrieveBuiltins(): Promise<Enumeration[]> {
    const res = await this.unary.send(
      "/enumeration/retrieve-builtins",
      {},
      emptyZ,
      retrieveBuiltinsResZ,
    );
    return res.enumerations;
  }

  /**
   * Returns the records that use the enumeration and that the caller can retrieve.
   * Each record that uses the enumeration refuses its delete.
   */
  async retrieveDependents(key: Key): Promise<resource.ID[]> {
    const res = await this.unary.send(
      "/enumeration/retrieve-dependents",
      { key },
      retrieveDependentsReqZ,
      retrieveDependentsResZ,
    );
    return res.dependents;
  }

  private async execRetrieve(params: RetrieveRequest): Promise<Enumeration[]> {
    const res = await this.unary.send(
      "/enumeration/retrieve",
      params,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.enumerations;
  }
}
