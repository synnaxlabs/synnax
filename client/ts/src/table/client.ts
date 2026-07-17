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

const MOUNT_SCOPE = "table.mounts";

/**
 * Client-side approximation of the server's matching for a request: exact for
 * the requested key set, the only field a request carries.
 */
const requestFilter = (req: RetrieveRequest): ((t: Table) => boolean) => {
  const keySet = new Set(req.keys);
  return (t) => keySet.has(t.key);
};

export class Client {
  private readonly client: UnaryClient;
  private readonly engine_?: cache.Engine;
  private readonly dispatcher_?: dispatch.Controller<Key, Table, Action>;
  private readonly queries_?: {
    single: cache.Queries<Key, Table>;
    request: cache.Queries<RetrieveRequest, Table[]>;
  };

  constructor(client: UnaryClient, engine?: cache.Engine) {
    this.client = client;
    if (engine == null) return;
    this.dispatcher_ = bindStore(engine);
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "table",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "tables",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
      }),
    };
  }

  private get dispatcher(): dispatch.Controller<Key, Table, Action> {
    if (this.dispatcher_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.dispatcher_;
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
    if (this.writes != null) rollback.add(this.writes.set(optimistic));
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
    this.writes?.set(res.tables);
    return isMany ? res.tables : res.tables[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    const rollback = new cache.Rollback();
    const writes = this.writes;
    if (this.engine_ != null && writes != null) {
      rollback.add(cache.partialUpdate(writes, key, { name }));
      rollback.add(ontology.renameCachedResource(this.engine_, ontologyID(key), name));
    }
    await rollback.guard(
      async () => await this.sendDispatch(key, "", [renameAction({ name })]),
    );
  }

  /**
   * Applies actions to the cached table and sends them to the server,
   * recording an undoable entry. Returns false without side effects when the
   * table isn't cached. Rolls back the local apply and rethrows on send
   * failure.
   * @throws when the cache was disabled at client construction.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: dispatch.Options<Table, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher.dispatch(
      "",
      key,
      array.toArray(actions),
      this.dispatchSender(key),
      opts.preprocess,
    );
  }

  /**
   * Reverts the table's most recent undoable entry. Returns false when
   * nothing is undoable.
   * @throws when the cache was disabled at client construction.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher.undo("", key, this.dispatchSender(key));
  }

  /**
   * Re-applies the table's most recently undone entry. Returns false when
   * nothing is redoable.
   * @throws when the cache was disabled at client construction.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo("", key, this.dispatchSender(key));
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
    return this.dispatcher.onUndoStateChange(MOUNT_SCOPE, callback, key);
  }

  /**
   * Stages actions committed atomically as one undoable entry.
   * @throws when the cache was disabled at client construction.
   */
  beginTransaction(key: Key, kind?: string): dispatch.Transaction<Action> {
    return this.dispatcher.transaction("", key, this.dispatchSender(key), kind);
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
    if (this.queries_ == null) {
      const tables = await this.execRetrieve(params);
      checkForMultipleOrNoResults("Table", params, tables, isSingle);
      return isSingle ? tables[0] : tables;
    }
    if (isSingle) return await this.queries_.single.retrieve(params.key);
    return await this.queries_.request.retrieve(retrieveReqZ.parse(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a table; every other shape delivers the matching tables.
   * @throws when the cache was disabled at client construction.
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
    const queries = this.requireQueries();
    if ("key" in params)
      return queries.single.onChange(params.key, handler as cache.ChangeHandler<Table>);
    return queries.request.onChange(
      retrieveReqZ.parse(params),
      handler as cache.ChangeHandler<Table[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   * @throws when the cache was disabled at client construction.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Table> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Table[]> | undefined;
  getCached(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): cache.Cached<Table> | cache.Cached<Table[]> | undefined {
    const queries = this.requireQueries();
    if ("key" in params) return queries.single.getCached(params.key);
    return queries.request.getCached(retrieveReqZ.parse(params));
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    if (this.engine_ != null)
      rollback.add(
        ontology.deleteCachedRelationships(this.engine_, ontologyID(keysArr)),
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
    this.writes?.delete(keysArr);
  }

  private get writes(): cache.UnaryStore<Key, Table> | undefined {
    return this.engine_?.store(STORE_KEY);
  }

  private get tableStore(): cache.UnaryStore<Key, Table> {
    return this.requireEngine().store(STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  private get tableEvents(): cache.UnaryStore<Key, Table> {
    return this.requireEngine().store(STORE_KEY, MOUNT_SCOPE);
  }

  private requireEngine(): cache.Engine {
    if (this.engine_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.engine_;
  }

  private requireQueries(): NonNullable<typeof this.queries_> {
    if (this.queries_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.queries_;
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
  // store without clobbering a doc holding locally replayed edits. Answers
  // read back from the store so replayed edits and reference identity win.
  private async fetchSingle(query: Key): Promise<Table> {
    const tables = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("Table", query, tables, true);
    this.tableStore.setIfAbsent(tables);
    return this.tableStore.get(query) ?? tables[0];
  }

  private mountSingle({ query, update, remove }: cache.MountParams<Key, Table>) {
    return [
      this.tableEvents.onSet((table) => {
        if (table.key === query) update(table);
      }),
      this.tableEvents.onDelete((key) => {
        if (key === query) remove(this.tableStore.getTombstone(key)?.corpse);
      }),
    ];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Table[]> {
    const tables = await this.execRetrieve(query);
    this.tableStore.setIfAbsent(tables);
    return tables.map((t) => this.tableStore.get(t.key) ?? t);
  }

  private mountRequest({ query, update }: cache.MountParams<RetrieveRequest, Table[]>) {
    const matches = requestFilter(query);
    return [
      this.tableEvents.onSet((table) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.some((t) => t.key === table.key);
          if (!matches(table))
            return existing ? prev.filter((t) => t.key !== table.key) : prev;
          if (existing) return prev.map((t) => (t.key === table.key ? table : t));
          return [...prev, table];
        });
      }),
      this.tableEvents.onDelete((key) => {
        update((prev) => prev?.filter((t) => t.key !== key));
      }),
    ];
  }
}
