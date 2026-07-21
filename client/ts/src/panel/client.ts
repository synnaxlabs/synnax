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

import { actions } from "@/actions";
import { cache } from "@/cache";
import { ontology } from "@/ontology";
import { kindOf, reduceAll } from "@/panel/actions";
import {
  type Action,
  dispatchReqZ,
  rename as renameAction,
  scopedActionZ,
} from "@/panel/actions.gen";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Panel,
  panelZ,
} from "@/panel/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_panel_set";
export const DELETE_CHANNEL_NAME = "sy_panel_delete";

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

export class Client extends cache.Reader<
  Key,
  Key[] | RetrieveRequest,
  Key,
  RetrieveRequest,
  Panel
> {
  private readonly client: UnaryClient;
  private readonly store: cache.Table<Key, Panel>;
  private readonly dispatcher: actions.Controller<Key, Panel, Action>;
  private readonly ontology: ontology.Stores;

  constructor(
    client: UnaryClient,
    engine: cache.Cache,
    ontologyStores: ontology.Stores,
  ) {
    const store = engine.createTable<Key, Panel>({ name: "panels" });
    const dispatcher = new actions.Controller<Key, Panel, Action>({
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
        name: "panel",
        table: store,
        fetch: async (query) => [await this.fetchSingle(query)].map((p) => p.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: engine.answers({
        name: "panels",
        table: store,
        fetch: async (query) => (await this.fetchRequest(query)).map((p) => p.key),
        compose: (records) => records,
        matches: (panel, query) => requestFilter(query)(panel),
        serverFields: SERVER_FIELDS,
      }),
      isSingle: (params) => typeof params === "string",
      normalizeSingle: (key) => key,
      normalizeRequest: toRequest,
    });
    this.client = client;
    this.ontology = ontologyStores;
    this.store = store;
    this.dispatcher = dispatcher;
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
    rollback.add(this.store.set(optimistic));
    await opts.onOptimistic?.(optimistic);
    // onOptimistic may dispatch further local mutations against these keys
    // before the panels exist on the cluster. Send the latest cached docs so
    // the server response doesn't stomp those changes back out.
    const latest = optimistic.map((p) => this.store.get(p.key) ?? p);
    const res = await rollback.guard(
      async () =>
        await this.client.send(
          "/panel/create",
          { panels: latest },
          createReqZ,
          createResZ,
        ),
    );
    this.store.set(res.panels);
    return isMany ? res.panels : res.panels[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    const rollback = new cache.Rollback();
    rollback.add(cache.partialUpdate(this.store, key, { name }));
    rollback.add(ontology.renameCachedResource(this.ontology, ontologyID(key), name));
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
    opts: actions.Options<Panel, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher.dispatch(
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
    return await this.dispatcher.undo(key, this.dispatchSender(key));
  }

  /**
   * Re-applies the panel's most recently undone entry. Returns false when
   * nothing is redoable.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo(key, this.dispatchSender(key));
  }

  /** Whether the panel has a live undo entry. */
  hasUndo(key: Key): boolean {
    return this.dispatcher.hasUndo(key);
  }

  /** Whether the panel has a live redo entry. */
  hasRedo(key: Key): boolean {
    return this.dispatcher.hasRedo(key);
  }

  /**
   * Subscribes to changes in the panel's undo/redo stacks. Returns a
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
    await this.client.send(
      "/panel/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async delete(key: Key, opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key[], opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    rollback.add(ontology.deleteCachedResources(this.ontology, ontologyID(keysArr)));
    rollback.add(this.store.delete(keysArr));
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
    this.store.setIfAbsent(panels);
    return panels[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Panel[]> {
    const panels = await this.execRetrieve(query);
    this.store.setIfAbsent(panels);
    return panels;
  }
}
