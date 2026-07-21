// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { cache } from "@/cache";
import { dispatch } from "@/dispatch";
import { kindOf, reduceAll } from "@/log/actions";
import {
  type Action,
  dispatchReqZ,
  rename as renameAction,
  scopedActionZ,
} from "@/log/actions.gen";
import { type Key, keyZ, type Log, logZ, type New, ontologyID } from "@/log/types.gen";
import { ontology } from "@/ontology";
import { project } from "@/project";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_log_set";
export const DELETE_CHANNEL_NAME = "sy_log_delete";

const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveReqZ = z.object({ keys: keyZ.array() });
const singleRetrieveParamsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

export const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveReqZ]);
export type RetrieveParams = z.input<typeof retrieveParamsZ>;
export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveReqZ>;

interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}

const retrieveResZ = z.object({ logs: logZ.array().default(() => []) });

const createReqZ = z.object({ project: project.keyZ, logs: logZ.array() });
const createResZ = z.object({ logs: logZ.array() });

const emptyResZ = z.object({});

/**
 * Client-side matching for a request: exact for the requested key set, the
 * only field a request carries.
 */
const requestFilter = (req: RetrieveRequest): ((l: Log) => boolean) => {
  const keySet = new Set(req.keys);
  return (l) => keySet.has(l.key);
};

export class Client extends cache.Reader<
  RetrieveSingleParams,
  RetrieveMultipleParams,
  Key,
  RetrieveRequest,
  Log
> {
  private readonly client: UnaryClient;
  private readonly store: cache.Table<Key, Log>;
  private readonly ontology: ontology.Stores;
  private readonly dispatcher: dispatch.Controller<Key, Log, Action>;

  constructor(
    client: UnaryClient,
    engine: cache.Cache,
    ontologyStores: ontology.Stores,
  ) {
    const store = engine.createTable<Key, Log>({ name: "logs" });
    const dispatcher = new dispatch.Controller<Key, Log, Action>({
      store,
      onError: engine.onError,
      reduce: reduceAll,
      kindOf,
    });
    const del: cache.ChannelListener<typeof keyZ> = {
      channel: DELETE_CHANNEL_NAME,
      schema: keyZ,
      onChange: (changed) => store.delete(changed),
    };
    engine.addListeners(
      store,
      del,
      dispatcher.listener(SET_CHANNEL_NAME, scopedActionZ),
    );
    super({
      single: engine.answers({
        name: "log",
        table: store,
        fetch: async (query) => [await this.fetchSingle(query)].map((l) => l.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: engine.answers({
        name: "logs",
        table: store,
        fetch: async (query) => (await this.fetchRequest(query)).map((l) => l.key),
        compose: (records) => records,
        matches: (log, query) => requestFilter(query)(log),
      }),
      isSingle: (params) => "key" in params,
      normalizeSingle: ({ key }) => key,
      normalizeRequest: (params) => retrieveReqZ.parse(params),
    });
    this.client = client;
    this.store = store;
    this.dispatcher = dispatcher;
    this.ontology = ontologyStores;
  }

  async create(
    project: project.Key,
    log: New,
    opts?: cache.WriteOptions<Log[]>,
  ): Promise<Log>;
  async create(
    project: project.Key,
    logs: New[],
    opts?: cache.WriteOptions<Log[]>,
  ): Promise<Log[]>;
  async create(
    project: project.Key,
    logs: New | New[],
    opts: cache.WriteOptions<Log[]> = {},
  ): Promise<Log | Log[]> {
    const isMany = Array.isArray(logs);
    const optimistic = array.toArray(logs).map((l) => logZ.parse(l));
    const rollback = new cache.Rollback();
    rollback.add(this.store.set(optimistic));
    await opts.onOptimistic?.(optimistic);
    const res = await rollback.guard(
      async () =>
        await this.client.send(
          "/log/create",
          { project, logs: optimistic },
          createReqZ,
          createResZ,
        ),
    );
    this.store.set(res.logs);
    return isMany ? res.logs : res.logs[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    const rollback = new cache.Rollback();
    rollback.add(cache.partialUpdate(this.store, key, { name }));
    rollback.add(ontology.renameCachedResource(this.ontology, ontologyID(key), name));
    await rollback.guard(
      async () => await this.sendDispatch(key, "", [renameAction({ name })]),
    );
  }

  /**
   * Applies actions to the cached log and sends them to the server,
   * recording an undoable entry. Returns false without side effects when the
   * log isn't cached. Rolls back the local apply and rethrows on send
   * failure.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: dispatch.Options<Log, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher.dispatch(
      key,
      array.toArray(actions),
      this.dispatchSender(key),
      opts.preprocess,
    );
  }

  /**
   * Reverts the log's most recent undoable entry. Returns false when
   * nothing is undoable.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher.undo(key, this.dispatchSender(key));
  }

  /**
   * Re-applies the log's most recently undone entry. Returns false when
   * nothing is redoable.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo(key, this.dispatchSender(key));
  }

  /** Whether the log has a live undo entry. */
  hasUndo(key: Key): boolean {
    return this.dispatcher.hasUndo(key);
  }

  /** Whether the log has a live redo entry. */
  hasRedo(key: Key): boolean {
    return this.dispatcher.hasRedo(key);
  }

  /**
   * Subscribes to changes in the log's undo/redo stacks. Returns a
   * destructor that unsubscribes.
   */
  onUndoStateChange(callback: () => void, key?: Key): destructor.Destructor {
    return this.dispatcher.onUndoStateChange(callback, key);
  }

  /** Stages actions committed atomically as one undoable entry. */
  beginTransaction(key: Key, kind?: string): dispatch.Transaction<Action> {
    return this.dispatcher.transaction(key, this.dispatchSender(key), kind);
  }

  private dispatchSender(key: Key): dispatch.SendDispatch<Action> {
    return async (actions, dispatchKey) =>
      await this.sendDispatch(key, dispatchKey, actions);
  }

  private async sendDispatch(
    key: Key,
    dispatchKey: string,
    actions: Action[],
  ): Promise<void> {
    await this.client.send(
      "/log/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    rollback.add(
      ontology.deleteCachedRelationships(this.ontology, ontologyID(keysArr)),
    );
    rollback.add(this.store.delete(keysArr));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send("/log/delete", { keys: keysArr }, deleteReqZ, emptyResZ),
    );
  }

  private async execRetrieve(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<Log[]> {
    const res = await this.client.send(
      "/log/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return res.logs;
  }

  // Dispatch mutates documents server-side, so a cached copy is only as fresh
  // as the streamer. Fetches always hit the network; setIfAbsent hydrates the
  // table without clobbering a doc holding locally replayed edits.
  private async fetchSingle(query: Key): Promise<Log> {
    const logs = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("Log", query, logs, true);
    this.store.setIfAbsent(logs);
    return logs[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Log[]> {
    const logs = await this.execRetrieve(query);
    this.store.setIfAbsent(logs);
    return logs;
  }
}
