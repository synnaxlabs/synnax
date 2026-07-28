// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { kindOf, reduceAll } from "@/log/actions";
import {
  type Action,
  dispatchReqZ,
  rename as renameAction,
  scopedActionZ,
} from "@/log/actions.gen";
import { type Key, keyZ, type Log, logZ, type New, ontologyID } from "@/log/types.gen";
import { type ontology } from "@/ontology";
import { project } from "@/project";
import { query } from "@/query";

export const SET_CHANNEL_NAME = "sy_log_set";
export const DELETE_CHANNEL_NAME = "sy_log_delete";

const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveReqZ = z.object({
  keys: keyZ.array(),
  ignoreNotFoundError: z.boolean().optional(),
});
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

export interface ClientParams {
  unary: UnaryClient;
  cache: query.Cache;
  ontology: ontology.Client;
}

export class Client extends query.Retriever<typeof retrieveReqZ, Key, Log> {
  private readonly cfg: ClientParams;
  private readonly store: query.Table<Key, Log>;
  private readonly dispatcher: actions.Controller<Key, Log, Action>;

  constructor(cfg: ClientParams) {
    const { cache } = cfg;
    // Dispatch mutates documents server-side, so fetched copies never clobber
    // a doc holding locally replayed edits: the table hydrates if-absent.
    const store = cache.createTable<Key, Log>({
      name: "logs",
      hydrate: "if-absent",
      fetch: async (keys) =>
        await this.execRetrieve({ keys, ignoreNotFoundError: true }),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    const dispatcher = new actions.Controller<Key, Log, Action>({
      store,
      onError: cache.onError,
      reduce: reduceAll,
      kindOf,
    });
    cache.listen(dispatcher.listener(SET_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "log",
      table: store,
      request: {
        schema: retrieveReqZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (log, req) => requestFilter(req)(log),
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
  }

  async create(
    project: project.Key,
    log: New,
    opts?: query.WriteOptions<Log[]>,
  ): Promise<Log>;
  async create(
    project: project.Key,
    logs: New[],
    opts?: query.WriteOptions<Log[]>,
  ): Promise<Log[]>;
  async create(
    project: project.Key,
    logs: New | New[],
    opts: query.WriteOptions<Log[]> = {},
  ): Promise<Log | Log[]> {
    const isMany = Array.isArray(logs);
    const optimistic = array.toArray(logs).map((l) => logZ.parse(l));
    const rollback = new destructor.Chain();
    rollback.add(this.store.set(optimistic));
    await opts.onOptimistic?.(optimistic);
    const res = await rollback.guard(
      async () =>
        await this.cfg.unary.send(
          "/log/create",
          { project, logs: optimistic },
          createReqZ,
          createResZ,
        ),
    );
    this.store.set(res.logs);
    return isMany ? res.logs : res.logs[0];
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      query.partialUpdate(this.store, key, { name }),
      this.cfg.ontology.cache.renameResource(ontologyID(key), name),
    ];
    const rollback = new destructor.Chain();
    rollback.add(...rename());
    await opts.onOptimistic?.();
    await rollback.guard(
      async () => await this.sendDispatch(key, "", [renameAction({ name })]),
    );
    rename();
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
    opts: actions.Options<Log, Action> = {},
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
  beginTransaction(key: Key, kind?: string): actions.Transaction<Action> {
    return this.dispatcher.transaction(key, this.dispatchSender(key), kind);
  }

  private dispatchSender(key: Key): actions.SendDispatch<Action> {
    return async (actions, dispatchKey) =>
      await this.sendDispatch(key, dispatchKey, actions);
  }

  private async sendDispatch(
    key: Key,
    dispatchKey: string,
    actions: Action[],
  ): Promise<void> {
    await this.cfg.unary.send(
      "/log/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [
      this.cfg.ontology.cache.deleteRelationships(ontologyID(keysArr)),
      this.store.delete(keysArr),
    ];
    const rollback = new destructor.Chain();
    rollback.add(...drop());
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.cfg.unary.send(
          "/log/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    );
    drop();
  }

  /** Subscribes to every log delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  private async execRetrieve(params: RetrieveMultipleParams): Promise<Log[]> {
    const res = await this.cfg.unary.send(
      "/log/retrieve",
      params,
      retrieveReqZ,
      retrieveResZ,
    );
    return res.logs;
  }
}
