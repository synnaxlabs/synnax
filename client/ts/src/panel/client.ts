// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, destructor, primitive } from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
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
  TYPE_ONTOLOGY_ID,
} from "@/panel/types.gen";
import { query } from "@/query";

export const SET_CHANNEL_NAME = "sy_panel_set";
export const DELETE_CHANNEL_NAME = "sy_panel_delete";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  parent: ontology.idZ.optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
  ignoreNotFoundError: z.boolean().optional(),
});
export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}
const createReqZ = z.object({ panels: panelZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveResZ = z.object({ panels: panelZ.array().default(() => []) });
const createResZ = z.object({ panels: panelZ.array() });
const emptyResZ = z.object({});

const isChildOf = (rel: ontology.Relationship, parent: ontology.ID): boolean =>
  ontology.matchRelationship(rel, {
    from: parent,
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to: { type: TYPE_ONTOLOGY_ID.type },
  });

/**
 * Client-side matching for a request: exact for key sets. Server-computed
 * shapes (search, limit/offset) never reach this filter; they refetch instead.
 */
const requestFilter = (req: RetrieveRequest): ((p: Panel) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (p) => keySet == null || keySet.has(p.key);
};

export interface ClientConfig {
  unary: UnaryClient;
  ontology: ontology.Client;
  cache: query.Cache;
  ontologyStores: ontology.Stores;
}

export class Client extends query.Retriever<typeof retrieveReqZ, Key, Panel> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Panel>;
  private readonly dispatcher: actions.Controller<Key, Panel, Action>;

  constructor(cfg: ClientConfig) {
    const { cache, ontologyStores } = cfg;
    // Dispatch mutates documents server-side, so fetched copies never clobber
    // a doc holding locally replayed edits: the table hydrates if-absent.
    const store = cache.createTable<Key, Panel>({
      name: "panels",
      hydrate: "if-absent",
      fetch: async (keys) =>
        await this.execRetrieve({ keys, ignoreNotFoundError: true }),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    const dispatcher = new actions.Controller<Key, Panel, Action>({
      store,
      onError: cache.onError,
      reduce: reduceAll,
      kindOf,
    });
    cache.listen(dispatcher.listener(SET_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "panel",
      table: store,
      request: {
        schema: retrieveReqZ,
        fetch: async (req) => await this.fetchRequest(req),
        matches: (panel, req) =>
          requestFilter(req)(panel) &&
          (req.parent == null || this.isCachedChild(req.parent, panel.key)),
        watch: [
          query.watch(ontologyStores.relationships, (event, req: RetrieveRequest) => {
            if (req.parent == null) return null;
            const rel =
              event.variant === "set"
                ? event.value
                : ontology.relationshipZ.parse(event.key);
            if (!isChildOf(rel, req.parent)) return null;
            return [rel.to.key];
          }),
        ],
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
  }

  /** All panels currently in the cache. */
  listCached(): Panel[] {
    return this.store.get();
  }

  private isCachedChild(parent: ontology.ID, key: Key): boolean {
    return this.cfg.ontologyStores.relationships.has(
      ontology.relationshipToString({
        from: parent,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: ontologyID(key),
      }),
    );
  }

  async create(panel: New, opts?: query.WriteOptions<Panel[]>): Promise<Panel>;
  async create(panels: New[], opts?: query.WriteOptions<Panel[]>): Promise<Panel[]>;
  async create(
    panels: New | New[],
    opts: query.WriteOptions<Panel[]> = {},
  ): Promise<Panel | Panel[]> {
    const isMany = Array.isArray(panels);
    const optimistic = array.toArray(panels).map((p) => panelZ.parse(p));
    const rollback = new destructor.Chain();
    rollback.add(this.store.set(optimistic));
    await opts.onOptimistic?.(optimistic);
    // onOptimistic may dispatch further local mutations against these keys
    // before the panels exist on the cluster. Send the latest cached docs so
    // the server response doesn't stomp those changes back out.
    const latest = optimistic.map((p) => this.store.get(p.key) ?? p);
    const res = await rollback.guard(
      async () =>
        await this.cfg.unary.send(
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
    const rename = () => [
      query.partialUpdate(this.store, key, { name }),
      ontology.renameCachedResource(this.cfg.ontologyStores, ontologyID(key), name),
    ];
    const rollback = new destructor.Chain();
    rollback.add(...rename());
    // Rename routes through dispatch so the action channel broadcasts the change
    // to other connected clients.
    await rollback.guard(
      async () => await this.sendDispatch(key, "", [renameAction({ name })]),
    );
    rename();
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
    opts?: actions.Options<Panel, Action>,
  ): Promise<boolean>;
  /**
   * Legacy raw-send form used by pre-cutover flux. Removed in the
   * pluto rebind; new callers use the two-argument controller form above.
   */
  async dispatch(key: Key, dispatchKey: string, actions: Action[]): Promise<void>;
  async dispatch(
    key: Key,
    actionsOrKey: Action | Action[] | string,
    opts: actions.Options<Panel, Action> | Action[] = {},
  ): Promise<boolean | void> {
    if (typeof actionsOrKey === "string")
      return await this.sendDispatch(key, actionsOrKey, opts as Action[]);
    return await this.dispatcher.dispatch(
      key,
      array.toArray(actionsOrKey),
      this.dispatchSender(key),
      (opts as actions.Options<Panel, Action>).preprocess,
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
    await this.cfg.unary.send(
      "/panel/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async delete(key: Key, opts?: query.WriteOptions): Promise<void>;
  async delete(keys: Key[], opts?: query.WriteOptions): Promise<void>;
  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [
      ontology.deleteCachedResources(this.cfg.ontologyStores, ontologyID(keysArr)),
      this.store.delete(keysArr),
    ];
    const rollback = new destructor.Chain();
    rollback.add(...drop());
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.cfg.unary.send(
          "/panel/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    );
    drop();
  }

  /** Subscribes to every panel delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  /** Subscribes to every panel set delivered to the cache. */
  onSet(handler: (panel: Panel) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "set") handler(event.value);
    });
  }

  private async execRetrieve(req: RetrieveRequest): Promise<Panel[]> {
    const res = await this.cfg.unary.send(
      "/panel/retrieve",
      req,
      retrieveReqZ,
      retrieveResZ,
    );
    return res.panels;
  }

  // Parent queries resolve membership through the ontology graph, then fetch
  // the member panels; the panel retrieve endpoint knows nothing of parents.
  private async fetchRequest(req: RetrieveRequest): Promise<Panel[]> {
    const { parent, ...rest } = req;
    if (parent != null) {
      const children = await this.cfg.ontology.children.retrieve({
        ids: parent,
        types: [TYPE_ONTOLOGY_ID.type],
      });
      const keys = children.map(({ id }) => id.key);
      if (keys.length === 0) return [];
      rest.keys = keys;
    }
    return await this.execRetrieve(rest);
  }
}
