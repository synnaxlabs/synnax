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

import { actions } from "@/actions";
import { kindOf, reduceAll } from "@/lineplot/actions";
import {
  type Action,
  dispatchReqZ,
  rename as renameAction,
  scopedActionZ,
} from "@/lineplot/actions.gen";
import {
  type Key,
  keyZ,
  type LinePlot,
  linePlotZ,
  type New,
  ontologyID,
} from "@/lineplot/types.gen";
import { ontology } from "@/ontology";
import { project } from "@/project";
import { query } from "@/query";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_lineplot_set";
export const DELETE_CHANNEL_NAME = "sy_lineplot_delete";

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

const retrieveResZ = z.object({ linePlots: linePlotZ.array().default(() => []) });

const createReqZ = z.object({ project: project.keyZ, linePlots: linePlotZ.array() });
const createResZ = z.object({ linePlots: linePlotZ.array() });

const emptyResZ = z.object({});

/**
 * Client-side matching for a request: exact for the requested key set, the
 * only field a request carries.
 */
const requestFilter = (req: RetrieveRequest): ((p: LinePlot) => boolean) => {
  const keySet = new Set(req.keys);
  return (p) => keySet.has(p.key);
};

export class Client extends query.Retriever<
  RetrieveSingleParams,
  RetrieveMultipleParams,
  Key,
  RetrieveRequest,
  LinePlot
> {
  private readonly client: UnaryClient;
  private readonly store: query.Table<Key, LinePlot>;
  private readonly ontology: ontology.Stores;
  private readonly dispatcher: actions.Controller<Key, LinePlot, Action>;

  constructor(
    client: UnaryClient,
    cache: query.Cache,
    ontologyStores: ontology.Stores,
  ) {
    const store = cache.createTable<Key, LinePlot>({ name: "lineplots" });
    const dispatcher = new actions.Controller<Key, LinePlot, Action>({
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
        name: "line plot",
        table: store,
        fetch: async (query) => [await this.fetchSingle(query)].map((p) => p.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: cache.queries({
        name: "line plots",
        table: store,
        fetch: async (query) => (await this.fetchRequest(query)).map((p) => p.key),
        compose: (records) => records,
        matches: (plot, query) => requestFilter(query)(plot),
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
    linePlot: New,
    opts?: query.WriteOptions<LinePlot[]>,
  ): Promise<LinePlot>;
  async create(
    project: project.Key,
    linePlots: New[],
    opts?: query.WriteOptions<LinePlot[]>,
  ): Promise<LinePlot[]>;
  async create(
    project: project.Key,
    linePlots: New | New[],
    opts: query.WriteOptions<LinePlot[]> = {},
  ): Promise<LinePlot | LinePlot[]> {
    const isMany = Array.isArray(linePlots);
    const optimistic = array.toArray(linePlots).map((p) => linePlotZ.parse(p));
    const rollback = new query.Rollback();
    rollback.add(this.store.set(optimistic));
    await opts.onOptimistic?.(optimistic);
    const res = await rollback.guard(
      async () =>
        await this.client.send(
          "/lineplot/create",
          { project, linePlots: optimistic },
          createReqZ,
          createResZ,
        ),
    );
    this.store.set(res.linePlots);
    return isMany ? res.linePlots : res.linePlots[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    const rollback = new query.Rollback();
    rollback.add(query.partialUpdate(this.store, key, { name }));
    rollback.add(ontology.renameCachedResource(this.ontology, ontologyID(key), name));
    await rollback.guard(
      async () => await this.sendDispatch(key, "", [renameAction({ name })]),
    );
  }

  /**
   * Applies actions to the cached line plot and sends them to the server,
   * recording an undoable entry. Returns false without side effects when the
   * line plot isn't cached. Rolls back the local apply and rethrows on send
   * failure.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: actions.Options<LinePlot, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher.dispatch(
      key,
      array.toArray(actions),
      this.dispatchSender(key),
      opts.preprocess,
    );
  }

  /**
   * Reverts the line plot's most recent undoable entry. Returns false when
   * nothing is undoable.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher.undo(key, this.dispatchSender(key));
  }

  /**
   * Re-applies the line plot's most recently undone entry. Returns false when
   * nothing is redoable.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo(key, this.dispatchSender(key));
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
      "/lineplot/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new query.Rollback();
    rollback.add(
      ontology.deleteCachedRelationships(this.ontology, ontologyID(keysArr)),
    );
    rollback.add(this.store.delete(keysArr));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send(
          "/lineplot/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    );
  }

  private async execRetrieve(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<LinePlot[]> {
    const res = await this.client.send(
      "/lineplot/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return res.linePlots;
  }

  // Dispatch mutates documents server-side, so a cached copy is only as fresh
  // as the streamer. Fetches always hit the network; setIfAbsent hydrates the
  // table without clobbering a doc holding locally replayed edits.
  private async fetchSingle(query: Key): Promise<LinePlot> {
    const plots = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("LinePlot", query, plots, true);
    this.store.setIfAbsent(plots);
    return plots[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<LinePlot[]> {
    const plots = await this.execRetrieve(query);
    this.store.setIfAbsent(plots);
    return plots;
  }
}
