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

const MOUNT_SCOPE = "panel.mounts";

/**
 * Client-side approximation of the server's matching for a request: exact for
 * key sets, permissive for server-computed shapes (search), which accept
 * every change and drift toward the server's answer.
 */
const requestFilter = (req: RetrieveRequest): ((p: Panel) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (p) => keySet == null || keySet.has(p.key);
};

const toRequest = (params: Key[] | RetrieveRequest): RetrieveRequest =>
  retrieveReqZ.parse(Array.isArray(params) ? { keys: params } : params);

export class Client {
  private readonly client: UnaryClient;
  private readonly engine_?: cache.Engine;
  private readonly dispatcher_?: dispatch.Controller<Key, Panel, Action>;
  private readonly queries_?: {
    single: cache.Queries<Key, Panel>;
    request: cache.Queries<RetrieveRequest, Panel[]>;
  };

  constructor(client: UnaryClient, engine?: cache.Engine) {
    this.client = client;
    if (engine == null) return;
    this.dispatcher_ = bindStore(engine);
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "panel",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "panels",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
      }),
    };
  }

  private get dispatcher(): dispatch.Controller<Key, Panel, Action> {
    if (this.dispatcher_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.dispatcher_;
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
    if (this.writes != null) rollback.add(this.writes.set(optimistic));
    await opts.onOptimistic?.(optimistic);
    // onOptimistic may dispatch further local mutations against these keys
    // before the panels exist on the cluster. Send the latest cached docs so
    // the server response doesn't stomp those changes back out.
    const latest = optimistic.map((p) => this.writes?.get(p.key) ?? p);
    const res = await rollback.guard(
      async () =>
        await this.client.send(
          "/panel/create",
          { panels: latest },
          createReqZ,
          createResZ,
        ),
    );
    this.writes?.set(res.panels);
    return isMany ? res.panels : res.panels[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    const rollback = new cache.Rollback();
    const writes = this.writes;
    if (this.engine_ != null && writes != null) {
      rollback.add(cache.partialUpdate(writes, key, { name }));
      rollback.add(ontology.renameCachedResource(this.engine_, ontologyID(key), name));
    }
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
   * @throws when the cache was disabled at client construction.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: dispatch.Options<Panel, Action> = {},
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
   * Reverts the panel's most recent undoable entry. Returns false when
   * nothing is undoable.
   * @throws when the cache was disabled at client construction.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher.undo("", key, this.dispatchSender(key));
  }

  /**
   * Re-applies the panel's most recently undone entry. Returns false when
   * nothing is redoable.
   * @throws when the cache was disabled at client construction.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo("", key, this.dispatchSender(key));
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
    if (this.queries_ == null) {
      const req =
        isSingle || Array.isArray(keys) ? { keys: array.toArray(keys) } : keys;
      const panels = await this.execRetrieve(req);
      checkForMultipleOrNoResults("Panel", keys, panels, isSingle);
      return isSingle ? panels[0] : panels;
    }
    if (isSingle) return await this.queries_.single.retrieve(keys);
    return await this.queries_.request.retrieve(toRequest(keys));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a panel; every other shape delivers the matching panels.
   * @throws when the cache was disabled at client construction.
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
    const queries = this.requireQueries();
    if (typeof keys === "string")
      return queries.single.onChange(keys, handler as cache.ChangeHandler<Panel>);
    return queries.request.onChange(
      toRequest(keys),
      handler as cache.ChangeHandler<Panel[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   * @throws when the cache was disabled at client construction.
   */
  getCached(key: Key): cache.Cached<Panel> | undefined;
  getCached(keys: Key[]): cache.Cached<Panel[]> | undefined;
  getCached(req: RetrieveRequest): cache.Cached<Panel[]> | undefined;
  getCached(
    keys: Key | Key[] | RetrieveRequest,
  ): cache.Cached<Panel> | cache.Cached<Panel[]> | undefined {
    const queries = this.requireQueries();
    if (typeof keys === "string") return queries.single.getCached(keys);
    return queries.request.getCached(toRequest(keys));
  }

  async delete(key: Key, opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key[], opts?: cache.WriteOptions): Promise<void>;
  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    if (this.engine_ != null)
      rollback.add(ontology.deleteCachedResources(this.engine_, ontologyID(keysArr)));
    if (this.writes != null) rollback.add(this.writes.delete(keysArr));
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

  private get writes(): cache.UnaryStore<Key, Panel> | undefined {
    return this.engine_?.store(STORE_KEY);
  }

  private get panelStore(): cache.UnaryStore<Key, Panel> {
    return this.requireEngine().store(STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  private get panelEvents(): cache.UnaryStore<Key, Panel> {
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
  // store without clobbering a doc holding locally replayed edits. Answers
  // read back from the store so replayed edits and reference identity win.
  private async fetchSingle(query: Key): Promise<Panel> {
    const panels = await this.execRetrieve({ keys: [query] });
    checkForMultipleOrNoResults("Panel", query, panels, true);
    this.panelStore.setIfAbsent(panels);
    return this.panelStore.get(query) ?? panels[0];
  }

  private mountSingle({ query, update, remove }: cache.MountParams<Key, Panel>) {
    return [
      this.panelEvents.onSet((panel) => {
        if (panel.key === query) update(panel);
      }),
      this.panelEvents.onDelete((key) => {
        if (key === query) remove(this.panelStore.getTombstone(key)?.corpse);
      }),
    ];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Panel[]> {
    const panels = await this.execRetrieve(query);
    this.panelStore.setIfAbsent(panels);
    return panels.map((p) => this.panelStore.get(p.key) ?? p);
  }

  private mountRequest({ query, update }: cache.MountParams<RetrieveRequest, Panel[]>) {
    const matches = requestFilter(query);
    return [
      this.panelEvents.onSet((panel) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.some((p) => p.key === panel.key);
          if (!matches(panel))
            return existing ? prev.filter((p) => p.key !== panel.key) : prev;
          if (existing) return prev.map((p) => (p.key === panel.key ? panel : p));
          return [...prev, panel];
        });
      }),
      this.panelEvents.onDelete((key) => {
        update((prev) => prev?.filter((p) => p.key !== key));
      }),
    ];
  }
}
