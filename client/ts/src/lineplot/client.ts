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
import { createOf, kindOf, reduceAll } from "@/lineplot/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/lineplot/actions.gen";
import {
  type Key,
  keyZ,
  type LinePlot,
  linePlotZ,
  type New,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
} from "@/lineplot/types.gen";
import {
  compareCursors,
  cursor,
  type Lines,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
} from "@/lineplot/where.gen";
import { project } from "@/project";
import { query } from "@/query";

export const SET_CHANNEL_NAME = "sy_lineplot_set";
export const DELETE_CHANNEL_NAME = "sy_lineplot_delete";

const deleteReqZ = z.object({ keys: keyZ.array() });

const LINES: Lines = { ...group.MEMBER_LINES, ...project.MEMBER_LINES };

const retrieveMultiParamsZ = z.union([
  query.keysRequestZ(keyZ),
  filter.requestInputZ(
    retrieveRequestZ,
    retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
  ),
]);

export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;
export type RetrieveSingleParams = { key: Key };
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;

const createReqZ = z.object({
  dispatchKey: z.string(),
  linePlots: linePlotZ.array(),
});
const createResZ = z.object({ linePlots: linePlotZ.array() });

const emptyResZ = z.object({});

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
}

export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  LinePlot
> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, LinePlot>;
  /**
   * Builds requests that move line plots between groups. A move into a group of another
   * project also moves the line plot into that project.
   */
  readonly setGroupRequest: group.MoveRequest<Key>;
  /** Builds requests that move line plots out of their group and into a project. */
  readonly setProjectRequest: project.MoveRequest<Key>;
  private readonly dispatcher: actions.Controller<Key, LinePlot, Action>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    // Dispatch mutates documents server-side, so fetched copies never clobber
    // a doc holding locally replayed edits: the table hydrates if-absent.
    const store = cache.createTable<Key, LinePlot>({
      name: "line plots",
      hydrate: "if-absent",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    const dispatcher = new actions.Controller<Key, LinePlot, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      reduce: reduceAll,
      send: actions.sender(cfg.unary, "/lineplot/dispatch", dispatchReqZ, dispatchResZ),
      kindOf,
      createOf,
    });
    cache.listen(dispatcher.listener(SET_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "line plot",
      table: store,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (plot, req) => matches(LINES, plot, req.where),
        serverComputed: (req) => serverComputed(LINES, req.where),
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.setGroupRequest = project.createGroupMoveRequest(
      "/lineplot/dispatch",
      store,
      dispatchResZ,
      reduceAll,
    );
    this.setProjectRequest = project.createMoveRequest(
      "/lineplot/dispatch",
      (keys, to) => project.moveCached(store, keys, to),
    );
    this.dispatcher = dispatcher;
  }

  async create(linePlot: New, opts?: query.WriteOptions<LinePlot[]>): Promise<LinePlot>;
  async create(
    linePlots: New[],
    opts?: query.WriteOptions<LinePlot[]>,
  ): Promise<LinePlot[]>;
  async create(
    linePlots: New | New[],
    opts: query.WriteOptions<LinePlot[]> = {},
  ): Promise<LinePlot | LinePlot[]> {
    const isMany = Array.isArray(linePlots);
    const optimistic = array
      .toArray(linePlots)
      .map((p) => zod.parse(linePlotZ, p, { label: "line plot" }));
    const res = await query.optimistic({
      rollbacks: [this.store.set(optimistic)],
      onOptimistic: () => opts.onOptimistic?.(optimistic),
      commit: async () =>
        await this.dispatcher.create(
          optimistic.map(({ key }) => key),
          async (dispatchKey) =>
            await this.cfg.unary.send(
              "/lineplot/create",
              { dispatchKey, linePlots: optimistic },
              createReqZ,
              createResZ,
            ),
        ),
    });
    this.store.set(res.linePlots);
    return isMany ? res.linePlots : res.linePlots[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatcher.send(key, [set({ path: ["name"], value: name })]);
  }

  /**
   * Applies actions to the cached line plot and sends them to the server, recording an
   * undoable entry. Returns false without side effects when the line plot isn't cached.
   * Rolls back the local apply and rethrows on send failure.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: actions.Options<LinePlot, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher.dispatch(key, array.toArray(actions), opts);
  }

  /**
   * Applies actions to the cached line plot, if any, and sends them to the server. It
   * records no undo entry, so it suits edits the user did not make.
   * @throws {Error} if the send fails, after the local apply rolls back.
   */
  async send(key: Key, actions: Action | Action[]): Promise<void> {
    await this.dispatcher.send(key, array.toArray(actions));
  }

  /**
   * Reverts the line plot's most recent undoable entry. Returns false when
   * nothing is undoable.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher.undo(key);
  }

  /**
   * Re-applies the line plot's most recently undone entry. Returns false when
   * nothing is redoable.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo(key);
  }

  /** Whether the line plot has a live undo entry. */
  hasUndo(key: Key): boolean {
    return this.dispatcher.hasUndo(key);
  }

  /** Whether the line plot has a live redo entry. */
  hasRedo(key: Key): boolean {
    return this.dispatcher.hasRedo(key);
  }

  /**
   * Subscribes to changes in the line plot's undo/redo stacks. Returns a
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
          "/lineplot/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  /** Subscribes to every line plot delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  private async execRetrieve(params: RetrieveRequest): Promise<LinePlot[]> {
    const res = await this.cfg.unary.send(
      "/lineplot/retrieve",
      params,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.linePlots;
  }
}
