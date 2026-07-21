// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, primitive } from "@synnaxlabs/x";
import { z } from "zod";

import { cache } from "@/cache";
import { type dispatch } from "@/dispatch";
import { ontology } from "@/ontology";
import { type Action, dispatchReqZ, rename as renameAction } from "@/panel/actions.gen";
import { bindStore, STORE_KEY } from "@/panel/store";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Panel,
  panelZ,
} from "@/panel/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
});
export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}
const createReqZ = z.object({ panels: panelZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveResZ = z.object({ panels: panelZ.array().default(() => []) });
const createResZ = z.object({ panels: panelZ.array() });
const emptyResZ = z.object({});

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

/**
 * Client-side matching for a request: exact for key sets. Server-computed
 * shapes (search, limit/offset) never reach this filter; they refetch instead.
 */
const requestFilter = (req: RetrieveRequest): ((p: Panel) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (p) => keySet == null || keySet.has(p.key);
};

const toRequest = (params: Key[] | RetrieveRequest): RetrieveRequest =>
  retrieveReqZ.parse(Array.isArray(params) ? { keys: params } : params);

export class Client {
  private readonly client: UnaryClient;
  private readonly cache_: cache.Cache;
  private readonly dispatcher_: dispatch.Controller<Key, Panel, Action>;
  private readonly answers_: {
    single: cache.Answers<Key, Panel, Key, Panel>;
    request: cache.Answers<RetrieveRequest, Panel[], Key, Panel>;
  };

  constructor(client: UnaryClient, engine: cache.Cache) {
    this.client = client;
    this.dispatcher_ = bindStore(engine);
    this.cache_ = engine;
    this.answers_ = {
      single: engine.answers({
        name: "panel",
        table: this.panelStore,
        fetch: async (query) => [await this.fetchSingle(query)].map((p) => p.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: engine.answers({
        name: "panels",
        table: this.panelStore,
        fetch: async (query) => (await this.fetchRequest(query)).map((p) => p.key),
        compose: (records) => records,
        matches: (panel, query) => requestFilter(query)(panel),
        serverFields: SERVER_FIELDS,
      }),
    };
  }

  async create(panel: New, opts?: cache.WriteOptions<Panel[]>): Promise<Panel>;
  async create(panels: New[], opts?: cache.WriteOptions<Panel[]>): Promise<Panel[]>;
  async create(
    panels: New | New[],
    opts: cache.WriteOptions<Panel[]> = {},
  ): Promise<Panel | Panel[]> {
    const isMany = Array.isArray(panels);
    const optimistic = array.toArray(panels).map((p) => panelZ.parse(p));
    const rollback = new cache.Rollback();
    rollback.add(this.panelStore.setMany(optimistic));
    await opts.onOptimistic?.(optimistic);
    // onOptimistic may dispatch further local mutations against these keys
    // before the panels exist on the cluster. Send the latest cached docs so
    // the server response doesn't stomp those changes back out.
    const latest = optimistic.map((p) => this.panelStore.get(p.key) ?? p);
    const res = await rollback.guard(
      async () =>
        await this.client.send(
          "/panel/create",
          { panels: latest },
          createReqZ,
          createResZ,
        ),
    );
    this.panelStore.setMany(res.panels);
    return isMany ? res.panels : res.panels[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    const rollback = new cache.Rollback();
    rollback.add(cache.partialUpdate(this.panelStore, key, { name }));
    rollback.add(ontology.renameCachedResource(this.cache_, ontologyID(key), name));
    // Rename routes through dispatch so the action channel broadcasts the change
    // to other connected clients.
    await rollback.guard(
      async () => await this.sendDispatch(key, "", [renameAction({ name })]),
    );
  }

  /**
   * Applies actions to the cached panel and sends them to the server,
   * recording an undoable entry. Returns false without side effects when the
   * panel isn't cached. Rolls back the local apply and rethrows on send
   * failure.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: dispatch.Options<Panel, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher_.dispatch(
      key,
      array.toArray(actions),
      this.dispatchSender(key),
      opts.preprocess,
    );
  }

  /**
   * Reverts the panel's most recent undoable entry. Returns false when
   * nothing is undoable.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher_.undo(key, this.dispatchSender(key));
  }

  /**
   * Re-applies the panel's most recently undone entry. Returns false when
   * nothing is redoable.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher_.redo(key, this.dispatchSender(key));
  }

  /** Whether the panel has a live undo entry. */
  hasUndo(key: Key): boolean {
    return this.dispatcher_.hasUndo(key);
  }

  /** Whether the panel has a live redo entry. */
  hasRedo(key: Key): boolean {
    return this.dispatcher_.hasRedo(key);
  }

  /**
   * Subscribes to changes in the panel's undo/redo stacks. Returns a
   * destructor that unsubscribes.
   */
  onUndoStateChange(callback: () => void, key?: Key): destructor.Destructor {
    return this.dispatcher_.onUndoStateChange(callback, key);
  }

  /** Stages actions committed atomically as one undoable entry. */
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
      "/panel/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async retrieve(key: Key): Promise<Panel>;
  async retrieve(keys: Key[]): Promise<Panel[]>;
  async retrieve(req: RetrieveRequest): Promise<Panel[]>;
  async retrieve(keys: Key | Key[] | RetrieveRequest): Promise<Panel | Panel[]> {
    const isSingle = typeof keys === "string";
    if (isSingle) return await this.answers_.single.retrieve(keys);
    return await this.answers_.request.retrieve(toRequest(keys));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a panel; every other shape delivers the matching panels.
   */
  onChange(key: Key, handler: cache.ChangeHandler<Panel>): destructor.Destructor;
  onChange(keys: Key[], handler: cache.ChangeHandler<Panel[]>): destructor.Destructor;
  onChange(
    req: RetrieveRequest,
    handler: cache.ChangeHandler<Panel[]>,
  ): destructor.Destructor;
  onChange(
    keys: Key | Key[] | RetrieveRequest,
    handler: cache.ChangeHandler<Panel> | cache.ChangeHandler<Panel[]>,
  ): destructor.Destructor {
    const answers = this.answers_;
    if (typeof keys === "string")
      return answers.single.onChange(keys, handler as cache.ChangeHandler<Panel>);
    return answers.request.onChange(
      toRequest(keys),
      handler as cache.ChangeHandler<Panel[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(key: Key): cache.Cached<Panel> | undefined;
  getCached(keys: Key[]): cache.Cached<Panel[]> | undefined;
  getCached(req: RetrieveRequest): cache.Cached<Panel[]> | undefined;
  getCached(
    keys: Key | Key[] | RetrieveRequest,
  ): cache.Cached<Panel> | cache.Cached<Panel[]> | undefined {
    const answers = this.answers_;
    if (typeof keys === "string") return answers.single.getCached(keys);
    return answers.request.getCached(toRequest(keys));
  }

  async delete(key: Key, opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key[], opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    rollback.add(ontology.deleteCachedResources(this.cache_, ontologyID(keysArr)));
    rollback.add(this.panelStore.delete(keysArr));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send(
          "/panel/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    );
  }

  private get panelStore(): cache.Table<Key, Panel> {
    return this.cache_.table(STORE_KEY);
  }

  private async execRetrieve(req: RetrieveRequest): Promise<Panel[]> {
    const res = await this.client.send(
      "/panel/retrieve",
      req,
      retrieveReqZ,
      retrieveResZ,
    );
    return res.panels;
  }

  // Dispatch mutates documents server-side, so a cached copy is only as fresh
  // as the streamer. Fetches always hit the network; setIfAbsent hydrates the
  // table without clobbering a doc holding locally replayed edits.
  private async fetchSingle(query: Key): Promise<Panel> {
    const panels = await this.execRetrieve({ keys: [query] });
    checkForMultipleOrNoResults("Panel", query, panels, true);
    this.panelStore.setIfAbsent(panels);
    return panels[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Panel[]> {
    const panels = await this.execRetrieve(query);
    this.panelStore.setIfAbsent(panels);
    return panels;
  }
}
