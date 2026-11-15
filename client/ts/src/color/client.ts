// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, filter, zod } from "@synnaxlabs/x";
import z from "zod";

import { actions } from "@/actions";
import { createOf, encode, reduceAll } from "@/color/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
} from "@/color/actions.gen";
import {
  type Key,
  keyZ,
  type Named,
  namedZ,
  type New,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
} from "@/color/types.gen";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
  shortcutZ,
} from "@/color/where.gen";
import { query } from "@/query";

export const ACTION_CHANNEL_NAME = "sy_color_action_set";
export const DELETE_CHANNEL_NAME = "sy_color_delete";

const createReqZ = z.object({ nameds: namedZ.array(), dispatchKey: z.string() });
const createResZ = z.object({ nameds: namedZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const retrieveMultiParamsZ = filter.requestInputZ(
  retrieveRequestZ,
  retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
);

/** Names one named color. */
export type RetrieveSingleParams = { key: Key };
/** Everything a retrieval of many named colors can filter on. */
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;
/** Params for a retrieval of one or many named colors. */
export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

/** Config for {@link Client}. */
export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
}

/**
 * Creates, reads, changes, and deletes named colors on a Core. Reach it through
 * `client.colors`.
 */
export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Named> {
  readonly type: string = "color";
  /** The named color table. */
  readonly store: query.Table<Key, Named>;
  private readonly unary: UnaryClient;
  private readonly dispatcher: actions.Controller<Key, Named, Action>;

  constructor({ unary, cache }: ClientConfig) {
    const store = cache.createTable<Key, Named>({
      name: "colors",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to a named color reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Named, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(unary, "/color/dispatch", dispatchReqZ, dispatchResZ),
      reduce: reduceAll,
      encode,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "color",
      table: store,
      shortcut: shortcutZ,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (named, req) => matches(named, req.where),
        serverComputed: (req) => serverComputed(req.where),
      },
    });
    this.unary = unary;
    this.store = store;
    this.dispatcher = dispatcher;
  }

  async create(named: New, opts?: query.WriteOptions<Named[]>): Promise<Named>;
  async create(named: New[], opts?: query.WriteOptions<Named[]>): Promise<Named[]>;
  async create(
    named: New | New[],
    opts: query.WriteOptions<Named[]> = {},
  ): Promise<Named | Named[]> {
    const isMany = Array.isArray(named);
    const optimistic = array
      .toArray(named)
      .map((n) => zod.parse(namedZ, n, { label: "named color" }));
    const res = await query.optimistic({
      rollbacks: [this.store.set(optimistic)],
      onOptimistic: () => opts.onOptimistic?.(optimistic),
      commit: async () =>
        await this.dispatcher.create(
          optimistic.map(({ key }) => key),
          async (dispatchKey) =>
            await this.unary.send(
              "/color/create",
              { nameds: optimistic, dispatchKey },
              createReqZ,
              createResZ,
            ),
        ),
    });
    this.store.set(res.nameds);
    return isMany ? res.nameds : res.nameds[0];
  }

  /**
   * Applies actions to the cached named color and sends them to the Core. Returns
   * false without side effects when the named color is not cached.
   * @throws {Error} if the named color cannot hold an action, before any send.
   * @throws {Error} if the send fails, after the cached named color drops the actions.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<boolean> {
    return await this.dispatcher.dispatch(key, array.toArray(actions));
  }

  /** Stages actions that commit atomically as one dispatch. */
  beginTransaction(key: Key): actions.Transaction<Action> {
    return this.dispatcher.transaction(key);
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [this.store.delete(keysArr)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.unary.send(
          "/color/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  private async execRetrieve(params: RetrieveRequest): Promise<Named[]> {
    const res = await this.unary.send(
      "/color/retrieve",
      params,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.nameds;
  }
}
