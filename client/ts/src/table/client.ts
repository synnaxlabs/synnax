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
import { type dispatch } from "@/dispatch";
import { ontology } from "@/ontology";
import { project } from "@/project";
import { type Action, dispatchReqZ, rename as renameAction } from "@/table/actions.gen";
import { bindStore, STORE_KEY } from "@/table/store";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Table,
  tableZ,
} from "@/table/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

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

const retrieveResZ = z.object({ tables: tableZ.array().default(() => []) });

const createReqZ = z.object({ project: project.keyZ, tables: tableZ.array() });
const createResZ = z.object({ tables: tableZ.array() });

const emptyResZ = z.object({});

/**
 * Client-side matching for a request: exact for the requested key set, the
 * only field a request carries.
 */
const requestFilter = (req: RetrieveRequest): ((t: Table) => boolean) => {
  const keySet = new Set(req.keys);
  return (t) => keySet.has(t.key);
};

export class Client {
  private readonly client: UnaryClient;
  private readonly cache_: cache.Cache;
  private readonly dispatcher_: dispatch.Controller<Key, Table, Action>;
  private readonly answers_: {
    single: cache.Answers<Key, Table, Key, Table>;
    request: cache.Answers<RetrieveRequest, Table[], Key, Table>;
  };

  constructor(client: UnaryClient, engine: cache.Cache) {
    this.client = client;
    this.dispatcher_ = bindStore(engine);
    this.cache_ = engine;
    this.answers_ = {
      single: engine.answers({
        name: "table",
        table: this.tableStore,
        fetch: async (query) => [await this.fetchSingle(query)].map((t) => t.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: engine.answers({
        name: "tables",
        table: this.tableStore,
        fetch: async (query) => (await this.fetchRequest(query)).map((t) => t.key),
        compose: (records) => records,
        matches: (table, query) => requestFilter(query)(table),
      }),
    };
  }

  async create(
    project: project.Key,
    table: New,
    opts?: cache.WriteOptions<Table[]>,
  ): Promise<Table>;
  async create(
    project: project.Key,
    tables: New[],
    opts?: cache.WriteOptions<Table[]>,
  ): Promise<Table[]>;
  async create(
    project: project.Key,
    tables: New | New[],
    opts: cache.WriteOptions<Table[]> = {},
  ): Promise<Table | Table[]> {
    const isMany = Array.isArray(tables);
    const optimistic = array.toArray(tables).map((t) => tableZ.parse(t));
    const rollback = new cache.Rollback();
    rollback.add(this.tableStore.setMany(optimistic));
    await opts.onOptimistic?.(optimistic);
    const res = await rollback.guard(
      async () =>
        await this.client.send(
          "/table/create",
          { project, tables: optimistic },
          createReqZ,
          createResZ,
        ),
    );
    this.tableStore.setMany(res.tables);
    return isMany ? res.tables : res.tables[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    const rollback = new cache.Rollback();
    rollback.add(cache.partialUpdate(this.tableStore, key, { name }));
    rollback.add(ontology.renameCachedResource(this.cache_, ontologyID(key), name));
    await rollback.guard(
      async () => await this.sendDispatch(key, "", [renameAction({ name })]),
    );
  }

  /**
   * Applies actions to the cached table and sends them to the server,
   * recording an undoable entry. Returns false without side effects when the
   * table isn't cached. Rolls back the local apply and rethrows on send
   * failure.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: dispatch.Options<Table, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher_.dispatch(
      key,
      array.toArray(actions),
      this.dispatchSender(key),
      opts.preprocess,
    );
  }

  /**
   * Reverts the table's most recent undoable entry. Returns false when
   * nothing is undoable.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher_.undo(key, this.dispatchSender(key));
  }

  /**
   * Re-applies the table's most recently undone entry. Returns false when
   * nothing is redoable.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher_.redo(key, this.dispatchSender(key));
  }

  /** Whether the table has a live undo entry. */
  hasUndo(key: Key): boolean {
    return this.dispatcher_.hasUndo(key);
  }

  /** Whether the table has a live redo entry. */
  hasRedo(key: Key): boolean {
    return this.dispatcher_.hasRedo(key);
  }

  /**
   * Subscribes to changes in the table's undo/redo stacks. Returns a
   * destructor that unsubscribes.
   */
  onUndoStateChange(callback: () => void, key?: Key): destructor.Destructor {
    return this.dispatcher_.onUndoStateChange(callback, key);
  }

  /**
   * Stages actions committed atomically as one undoable entry.
   */
  beginTransaction(key: Key, kind?: string): dispatch.Transaction<Action> {
    return this.dispatcher_.transaction(key, this.dispatchSender(key), kind);
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
      "/table/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async retrieve(params: RetrieveSingleParams): Promise<Table>;
  async retrieve(params: RetrieveMultipleParams): Promise<Table[]>;
  async retrieve(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<Table | Table[]> {
    const isSingle = "key" in params;
    if (isSingle) return await this.answers_.single.retrieve(params.key);
    return await this.answers_.request.retrieve(retrieveReqZ.parse(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a table; every other shape delivers the matching tables.
   */
  onChange(
    params: RetrieveSingleParams,
    handler: cache.ChangeHandler<Table>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveMultipleParams,
    handler: cache.ChangeHandler<Table[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveSingleParams | RetrieveMultipleParams,
    handler: cache.ChangeHandler<Table> | cache.ChangeHandler<Table[]>,
  ): destructor.Destructor {
    const answers = this.answers_;
    if ("key" in params)
      return answers.single.onChange(params.key, handler as cache.ChangeHandler<Table>);
    return answers.request.onChange(
      retrieveReqZ.parse(params),
      handler as cache.ChangeHandler<Table[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Table> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Table[]> | undefined;
  getCached(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): cache.Cached<Table> | cache.Cached<Table[]> | undefined {
    const answers = this.answers_;
    if ("key" in params) return answers.single.getCached(params.key);
    return answers.request.getCached(retrieveReqZ.parse(params));
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    rollback.add(ontology.deleteCachedRelationships(this.cache_, ontologyID(keysArr)));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send(
          "/table/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    );
    this.tableStore.delete(keysArr);
  }

  private get tableStore(): cache.Table<Key, Table> {
    return this.cache_.table(STORE_KEY);
  }

  private async execRetrieve(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<Table[]> {
    const res = await this.client.send(
      "/table/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return res.tables;
  }

  // Dispatch mutates documents server-side, so a cached copy is only as fresh
  // as the streamer. Fetches always hit the network; setIfAbsent hydrates the
  // table without clobbering a doc holding locally replayed edits.
  private async fetchSingle(query: Key): Promise<Table> {
    const tables = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("Table", query, tables, true);
    this.tableStore.setIfAbsent(tables);
    return tables[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Table[]> {
    const tables = await this.execRetrieve(query);
    this.tableStore.setIfAbsent(tables);
    return tables;
  }
}
