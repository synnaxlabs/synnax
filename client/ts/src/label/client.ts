// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, filter, unique, zod } from "@synnaxlabs/x";
import z from "zod";

import { actions } from "@/actions";
import { type batch } from "@/batch";
import { createOf, encode, reduceAll } from "@/label/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
} from "@/label/actions.gen";
import {
  type Key,
  keyZ,
  type Label,
  labelZ,
  type New,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
} from "@/label/types.gen";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
  shortcutZ,
} from "@/label/where.gen";
import { query } from "@/query";
import { resource } from "@/resource";

export const ACTION_CHANNEL_NAME = "sy_label_action_set";
export const DELETE_CHANNEL_NAME = "sy_label_delete";

const createReqZ = z.object({ labels: labelZ.array(), dispatchKey: z.string() });
const createResZ = z.object({ labels: labelZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const setReqZ = z.object({
  id: resource.idZ,
  labels: keyZ.array(),
  replace: z.boolean().optional(),
});

interface SetReq extends z.infer<typeof setReqZ> {}
/** Options for `Client.label`. Set `replace` to swap the whole label set at once. */
export interface SetOptions extends Pick<SetReq, "replace"> {}

const removeReqZ = setReqZ.omit({ replace: true });
const emptyResZ = z.object({});

const retrieveMultiParamsZ = filter.requestInputZ(
  retrieveRequestZ,
  retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
);

/** Names one label. */
export type RetrieveSingleParams = { key: Key };
/** Everything a multi-label retrieval can filter on. */
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;
/** Params for a label retrieval, single or multiple. */
export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

/**
 * Changes the labels of the cached record under the key with update, and returns the
 * rollback. It skips a record the cache does not hold.
 */
export type Relabel = (
  key: string,
  update: (labels: Key[]) => Key[],
) => destructor.Destructor;

/** Config for {@link Client}. */
export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  /** Changes the cached labels of the records of each resource type that holds them. */
  relabel: Record<string, Relabel>;
}

/**
 * Creates, reads, deletes, and attaches labels on a Core. Reach it through
 * `client.labels`.
 */
export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Label> {
  readonly type: string = "label";
  /** The label record table; injected into sibling clients at wiring. */
  readonly store: query.Table<Key, Label>;
  private readonly cfg: ClientConfig;
  private readonly dispatcher: actions.Controller<Key, Label, Action>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Label>({
      name: "labels",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to a label reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Label, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(cfg.unary, "/label/dispatch", dispatchReqZ, dispatchResZ),
      reduce: reduceAll,
      encode,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "label",
      table: store,
      shortcut: shortcutZ,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (label, req) => matches(label, req.where),
        serverComputed: (req) => serverComputed(req.where),
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
  }

  /**
   * Attaches labels to the range or status that the ID names. The record keeps the
   * labels it holds unless `replace` is set.
   * @throws {ValidationError} if the ID does not name a range or a status.
   */
  async label(id: resource.ID, labels: Key[], opts: SetOptions = {}): Promise<void> {
    await this.send(this.labelRequest(id, labels, opts), setReqZ);
  }

  /** Builds a request that attaches labels, as label does. */
  labelRequest(
    id: resource.ID,
    labels: Key[],
    opts: SetOptions = {},
  ): batch.Request<z.input<typeof setReqZ>> {
    return {
      endpoint: "/label/set",
      payload: { id, labels, replace: opts.replace },
      apply: () =>
        this.relabel(id, (prev) =>
          opts.replace === true ? labels : unique.unique([...prev, ...labels]),
        ),
    };
  }

  /**
   * Detaches labels from the range or status that the ID names.
   * @throws {ValidationError} if the ID does not name a range or a status.
   */
  async remove(id: resource.ID, labels: Key[]): Promise<void> {
    await this.send(
      {
        endpoint: "/label/remove",
        payload: { id, labels },
        apply: () =>
          this.relabel(id, (prev) => prev.filter((k) => !labels.includes(k))),
      },
      removeReqZ,
    );
  }

  private relabel(
    id: resource.ID,
    update: (labels: Key[]) => Key[],
  ): destructor.Destructor {
    return this.cfg.relabel[id.type]?.(id.key, update) ?? (() => {});
  }

  /**
   * Sends the request, applying its cache write before and again after the Core commits
   * it, since a change that lands in between can undo the first write.
   */
  private async send<P>(request: batch.Request<P>, reqZ: z.ZodType<P>): Promise<void> {
    const { endpoint, payload, apply } = request;
    await query.optimistic({
      rollbacks: apply == null ? [] : [apply()],
      commit: async () => await this.cfg.unary.send(endpoint, payload, reqZ, emptyResZ),
    });
    apply?.();
  }

  async create(label: New, opts?: query.WriteOptions<Label[]>): Promise<Label>;
  async create(labels: New[], opts?: query.WriteOptions<Label[]>): Promise<Label[]>;
  async create(
    labels: New | New[],
    opts: query.WriteOptions<Label[]> = {},
  ): Promise<Label | Label[]> {
    const isMany = Array.isArray(labels);
    const optimistic = array
      .toArray(labels)
      .map((l) => zod.parse(labelZ, l, { label: "label" }));
    const res = await query.optimistic({
      rollbacks: [this.store.set(optimistic)],
      onOptimistic: () => opts.onOptimistic?.(optimistic),
      commit: async () =>
        await this.dispatcher.create(
          optimistic.map(({ key }) => key),
          async (dispatchKey) =>
            await this.cfg.unary.send(
              "/label/create",
              { labels: optimistic, dispatchKey },
              createReqZ,
              createResZ,
            ),
        ),
    });
    this.store.set(res.labels);
    return isMany ? res.labels : res.labels[0];
  }

  /**
   * Applies actions to the cached label and sends them to the Core. Returns false
   * without side effects when the label is not cached.
   * @throws {Error} if the label cannot hold an action, before any send.
   * @throws {Error} if the send fails, after the cached label drops the actions.
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
        await this.cfg.unary.send(
          "/label/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  private async execRetrieve(params: RetrieveRequest): Promise<Label[]> {
    const res = await this.cfg.unary.send(
      "/label/retrieve",
      params,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.labels;
  }
}
