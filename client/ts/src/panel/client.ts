// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, filter, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { group } from "@/group";
import { createOf, kindOf, reduceAll } from "@/panel/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/panel/actions.gen";
import {
  type Key,
  keyZ,
  type New,
  type Panel,
  panelZ,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
} from "@/panel/types.gen";
import {
  compareCursors,
  cursor,
  type Lines,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
} from "@/panel/where.gen";
import { project } from "@/project";
import { query } from "@/query";

export const SET_CHANNEL_NAME = "sy_panel_set";
export const DELETE_CHANNEL_NAME = "sy_panel_delete";

const LINES: Lines = {
  ...group.MEMBER_LINES,
  ...project.MEMBER_LINES,
  owners: ({ owner }, keys) => owner != null && keys.includes(owner),
  owned: ({ owner }, owned) => (owner != null) === owned,
};

const retrieveMultiParamsZ = z.union([
  query.keysRequestZ(keyZ),
  filter.requestInputZ(
    retrieveRequestZ,
    retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
  ),
]);
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;
const createReqZ = z.object({ dispatchKey: z.string(), panels: panelZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const createResZ = z.object({ panels: panelZ.array() });
const emptyResZ = z.object({});

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Panel> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Panel>;
  /**
   * Builds requests that move panels between groups. A move into a group of another
   * project also moves the panel into that project.
   */
  readonly setGroupRequest: group.MoveRequest<Key>;
  /** Builds requests that move panels out of their group and into a project. */
  readonly setProjectRequest: project.MoveRequest<Key>;
  private readonly dispatcher: actions.Controller<Key, Panel, Action>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    // Dispatch mutates documents server-side, so fetched copies never clobber
    // a doc holding locally replayed edits: the table hydrates if-absent.
    const store = cache.createTable<Key, Panel>({
      name: "panels",
      hydrate: "if-absent",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    const dispatcher = new actions.Controller<Key, Panel, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      reduce: reduceAll,
      send: actions.sender(cfg.unary, "/panel/dispatch", dispatchReqZ, dispatchResZ),
      kindOf,
      createOf,
    });
    cache.listen(dispatcher.listener(SET_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "panel",
      table: store,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (panel, req) => matches(LINES, panel, req.where),
        serverComputed: (req) => serverComputed(LINES, req.where),
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.setGroupRequest = project.createGroupMoveRequest(
      "/panel/dispatch",
      store,
      dispatchResZ,
      reduceAll,
    );
    this.setProjectRequest = project.createMoveRequest("/panel/dispatch", (keys, to) =>
      project.moveCached(store, keys, to),
    );
    this.dispatcher = dispatcher;
  }

  /** All panels currently in the cache. */
  listCached(): Panel[] {
    return this.store.get();
  }

  async create(panel: New, opts?: query.WriteOptions<Panel[]>): Promise<Panel>;
  async create(panels: New[], opts?: query.WriteOptions<Panel[]>): Promise<Panel[]>;
  async create(
    panels: New | New[],
    opts: query.WriteOptions<Panel[]> = {},
  ): Promise<Panel | Panel[]> {
    const isMany = Array.isArray(panels);
    const optimistic = array
      .toArray(panels)
      .map((p) => zod.parse(panelZ, p, { label: "panel" }));
    const res = await query.optimistic({
      rollbacks: [this.store.set(optimistic)],
      onOptimistic: () => opts.onOptimistic?.(optimistic),
      commit: async () => {
        // onOptimistic may dispatch further local mutations against these keys
        // before the panels exist on the cluster. Send the latest cached docs so
        // the server response doesn't stomp those changes back out.
        const latest = optimistic.map((p) => this.store.get(p.key) ?? p);
        return await this.dispatcher.create(
          latest.map(({ key }) => key),
          async (dispatchKey) =>
            await this.cfg.unary.send(
              "/panel/create",
              { dispatchKey, panels: latest },
              createReqZ,
              createResZ,
            ),
        );
      },
    });
    this.store.set(res.panels);
    return isMany ? res.panels : res.panels[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatcher.send(key, [set({ path: ["name"], value: name })]);
  }

  /**
   * Applies actions to the cached panel and sends them to the server, recording an
   * undoable entry. Returns false without side effects when the panel isn't cached.
   * Rolls back the local apply and rethrows on send failure.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: actions.Options<Panel, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher.dispatch(key, array.toArray(actions), opts);
  }

  /**
   * Reverts the panel's most recent undoable entry. Returns false when
   * nothing is undoable.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher.undo(key);
  }

  /**
   * Re-applies the panel's most recently undone entry. Returns false when
   * nothing is redoable.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo(key);
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
    return this.dispatcher.transaction(key, kind);
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [this.store.delete(keysArr)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/panel/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
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
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.panels;
  }
}
