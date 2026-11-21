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
import { createOf, encode, reduceAll } from "@/color/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
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
import { group } from "@/group";
import { query } from "@/query";

export const ACTION_CHANNEL_NAME = "sy_color_action_set";
export const DELETE_CHANNEL_NAME = "sy_color_delete";

const createReqZ = z.object({ colors: namedZ.array(), dispatchKey: z.string() });
const createResZ = z.object({ colors: namedZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const retrieveBuiltinsResZ = z.object({ colors: namedZ.array() });
const emptyZ = z.object({});

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
  /** Builds requests that move named colors between palettes. */
  readonly setGroupRequest: group.MoveRequest<Key>;

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
        matches: (named, req) => matches(group.MEMBER_LINES, named, req.where),
        serverComputed: (req) => serverComputed(group.MEMBER_LINES, req.where),
      },
    });
    this.unary = unary;
    this.store = store;
    this.dispatcher = dispatcher;
    this.setGroupRequest = group.createMoveRequest((key, acts) =>
      dispatcher.request("/color/dispatch", key, acts, dispatchResZ),
    );
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
              { colors: optimistic, dispatchKey },
              createReqZ,
              createResZ,
            ),
        ),
    });
    this.store.set(res.colors);
    return isMany ? res.colors : res.colors[0];
  }

  /**
   * Applies actions to the cached named color and sends them to the Core.
   * @throws {NotCachedError} if the named color is not cached, before any change.
   * @throws {Error} if the named color cannot hold an action, before any send.
   * @throws {Error} if the send fails, after the cached named color drops the actions.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<boolean> {
    return await this.dispatcher.dispatch(key, array.toArray(actions));
  }

  /**
   * Stages actions that commit atomically as one dispatch.
   * @throws {NotCachedError} if the document is not cached.
   */
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
        await this.unary.send("/color/delete", { keys: keysArr }, deleteReqZ, emptyZ),
    });
    drop();
  }

  /** Builds a batch entry that deletes the named colors. */
  deleteRequest(keys: Key | Key[]): batch.Request<z.infer<typeof deleteReqZ>> {
    const payload = { keys: array.toArray(keys) };
    return {
      endpoint: "/color/delete",
      payload,
      apply: () => this.store.delete(payload.keys),
    };
  }

  /**
   * Builds a batch entry that sets the position of the named color in its palette. The
   * entry's cache write moves the cached named color.
   */
  setPositionRequest(
    key: Key,
    position: string,
  ): batch.Request<actions.DispatchPayload<Key, Action>> {
    return {
      endpoint: "/color/dispatch",
      payload: { key, actions: [set({ path: ["position"], value: position })] },
      apply: () => {
        const prev = this.store.get(key)?.position;
        query.partialUpdate(this.store, key, { position });
        return () => query.partialUpdate(this.store, key, { position: prev });
      },
    };
  }

  /**
   * Returns the named colors that the Core ships, with their shipped values. A stored
   * built-in can differ from its shipped values, because users can edit it.
   */
  async retrieveBuiltins(): Promise<Named[]> {
    const res = await this.unary.send(
      "/color/retrieve-builtins",
      {},
      emptyZ,
      retrieveBuiltinsResZ,
    );
    return res.colors;
  }

  private async execRetrieve(params: RetrieveRequest): Promise<Named[]> {
    const res = await this.unary.send(
      "/color/retrieve",
      params,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.colors;
  }
}
