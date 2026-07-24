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
import { ontology } from "@/ontology";
import { project } from "@/project";
import { query } from "@/query";
import { kindOf, reduceAll } from "@/table/actions";
import {
  type Action,
  dispatchReqZ,
  rename as renameAction,
  scopedActionZ,
} from "@/table/actions.gen";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Table,
  tableZ,
} from "@/table/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_table_set";
export const DELETE_CHANNEL_NAME = "sy_table_delete";

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

export class Client extends query.Retriever<
  RetrieveSingleParams,
  RetrieveMultipleParams,
  Key,
  RetrieveRequest,
  Table
> {
  private readonly client: UnaryClient;
  private readonly store: query.Table<Key, Table>;
  private readonly ontology: ontology.Stores;
  private readonly dispatcher: actions.Controller<Key, Table, Action>;

  constructor(
    client: UnaryClient,
    cache: query.Cache,
    ontologyStores: ontology.Stores,
  ) {
    const store = cache.createTable<Key, Table>({
      name: "tables",
      refetch: async (keys) =>
        await this.execRetrieve({ keys, ignoreNotFoundError: true }),
    });
    const dispatcher = new actions.Controller<Key, Table, Action>({
      store,
      onError: cache.onError,
      reduce: reduceAll,
      kindOf,
    });
    const del: query.ChannelListener<typeof keyZ> = {
      channel: DELETE_CHANNEL_NAME,
      schema: keyZ,
      onChange: (changed) => store.delete(changed),
    };
    cache.addListeners(
      store,
      del,
      dispatcher.listener(SET_CHANNEL_NAME, scopedActionZ),
    );
    super({
      single: cache.queries({
        name: "table",
        table: store,
        fetch: async (query) => [await this.fetchSingle(query)].map((t) => t.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: cache.queries({
        name: "tables",
        table: store,
        fetch: async (query) => (await this.fetchRequest(query)).map((t) => t.key),
        compose: (records) => records,
        matches: (table, query) => requestFilter(query)(table),
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
    table: New,
    opts?: query.WriteOptions<Table[]>,
  ): Promise<Table>;
  async create(
    project: project.Key,
    tables: New[],
    opts?: query.WriteOptions<Table[]>,
  ): Promise<Table[]>;
  async create(
    project: project.Key,
    tables: New | New[],
    opts: query.WriteOptions<Table[]> = {},
  ): Promise<Table | Table[]> {
    const isMany = Array.isArray(tables);
    const optimistic = array.toArray(tables).map((t) => tableZ.parse(t));
    const rollback = new destructor.Chain();
    rollback.add(this.store.set(optimistic));
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
    this.store.set(res.tables);
    return isMany ? res.tables : res.tables[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    const rollback = new destructor.Chain();
    rollback.add(query.partialUpdate(this.store, key, { name }));
    rollback.add(ontology.renameCachedResource(this.ontology, ontologyID(key), name));
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
    opts?: actions.Options<Table, Action>,
  ): Promise<boolean>;
  /**
   * @deprecated Legacy raw-send form used by pre-cutover flux. Removed in the
   * pluto rebind; new callers use the two-argument controller form above.
   */
  async dispatch(key: Key, dispatchKey: string, actions: Action[]): Promise<void>;
  async dispatch(
    key: Key,
    actionsOrKey: Action | Action[] | string,
    opts: actions.Options<Table, Action> | Action[] = {},
  ): Promise<boolean | void> {
    if (typeof actionsOrKey === "string")
      return await this.sendDispatch(key, actionsOrKey, opts as Action[]);
    return await this.dispatcher.dispatch(
      key,
      array.toArray(actionsOrKey),
      this.dispatchSender(key),
      (opts as actions.Options<Table, Action>).preprocess,
    );
  }

  /**
   * Reverts the table's most recent undoable entry. Returns false when
   * nothing is undoable.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher.undo(key, this.dispatchSender(key));
  }

  /**
   * Re-applies the table's most recently undone entry. Returns false when
   * nothing is redoable.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo(key, this.dispatchSender(key));
  }

  /** Whether the table has a live undo entry. */
  hasUndo(key: Key): boolean {
    return this.dispatcher.hasUndo(key);
  }

  /** Whether the table has a live redo entry. */
  hasRedo(key: Key): boolean {
    return this.dispatcher.hasRedo(key);
  }

  /**
   * Subscribes to changes in the table's undo/redo stacks. Returns a
   * destructor that unsubscribes.
   */
  onUndoStateChange(callback: () => void, key?: Key): destructor.Destructor {
    return this.dispatcher.onUndoStateChange(callback, key);
  }

  /**
   * Stages actions committed atomically as one undoable entry.
   */
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
    await this.client.send(
      "/table/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new destructor.Chain();
    rollback.add(
      ontology.deleteCachedRelationships(this.ontology, ontologyID(keysArr)),
    );
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
    this.store.delete(keysArr);
  }

  /** Subscribes to every table delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
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
    this.store.setIfAbsent(tables);
    return tables[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Table[]> {
    const tables = await this.execRetrieve(query);
    this.store.setIfAbsent(tables);
    return tables;
  }
}
