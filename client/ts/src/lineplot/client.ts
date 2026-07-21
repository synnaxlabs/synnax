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
import { dispatch } from "@/dispatch";
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

export class Client {
  private readonly client: UnaryClient;
  private readonly store: cache.Table<Key, LinePlot>;
  private readonly ontology: ontology.Stores;
  private readonly dispatcher: dispatch.Controller<Key, LinePlot, Action>;
  private readonly answers: {
    single: cache.Answers<Key, LinePlot, Key, LinePlot>;
    request: cache.Answers<RetrieveRequest, LinePlot[], Key, LinePlot>;
  };

  constructor(
    client: UnaryClient,
    engine: cache.Cache,
    ontologyStores: ontology.Stores,
  ) {
    this.client = client;
    this.store = engine.createTable<Key, LinePlot>({ name: "lineplots" });
    this.dispatcher = new dispatch.Controller<Key, LinePlot, Action>({
      store: this.store,
      onError: engine.onError,
      reduce: reduceAll,
      kindOf,
    });
    const del: cache.ChannelListener<typeof keyZ> = {
      channel: DELETE_CHANNEL_NAME,
      schema: keyZ,
      onChange: (changed) => this.store.delete(changed),
    };
    engine.addListeners(
      this.store,
      del,
      this.dispatcher.listener(SET_CHANNEL_NAME, scopedActionZ),
    );
    this.ontology = ontologyStores;
    this.answers = {
      single: engine.answers({
        name: "line plot",
        table: this.store,
        fetch: async (query) => [await this.fetchSingle(query)].map((p) => p.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: engine.answers({
        name: "line plots",
        table: this.store,
        fetch: async (query) => (await this.fetchRequest(query)).map((p) => p.key),
        compose: (records) => records,
        matches: (plot, query) => requestFilter(query)(plot),
      }),
    };
  }

  async create(
    project: project.Key,
    linePlot: New,
    opts?: cache.WriteOptions<LinePlot[]>,
  ): Promise<LinePlot>;
  async create(
    project: project.Key,
    linePlots: New[],
    opts?: cache.WriteOptions<LinePlot[]>,
  ): Promise<LinePlot[]>;
  async create(
    project: project.Key,
    linePlots: New | New[],
    opts: cache.WriteOptions<LinePlot[]> = {},
  ): Promise<LinePlot | LinePlot[]> {
    const isMany = Array.isArray(linePlots);
    const optimistic = array.toArray(linePlots).map((p) => linePlotZ.parse(p));
    const rollback = new cache.Rollback();
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
    const rollback = new cache.Rollback();
    rollback.add(cache.partialUpdate(this.store, key, { name }));
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
    opts: dispatch.Options<LinePlot, Action> = {},
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
  beginTransaction(key: Key, kind?: string): dispatch.Transaction<Action> {
    return this.dispatcher.transaction(key, this.dispatchSender(key), kind);
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
      "/lineplot/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async retrieve(params: RetrieveSingleParams): Promise<LinePlot>;
  async retrieve(params: RetrieveMultipleParams): Promise<LinePlot[]>;
  async retrieve(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<LinePlot | LinePlot[]> {
    const isSingle = "key" in params;
    if (isSingle) return await this.answers.single.retrieve(params.key);
    return await this.answers.request.retrieve(retrieveReqZ.parse(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a line plot; every other shape delivers the matching
   * line plots.
   */
  onChange(
    params: RetrieveSingleParams,
    handler: cache.ChangeHandler<LinePlot>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveMultipleParams,
    handler: cache.ChangeHandler<LinePlot[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveSingleParams | RetrieveMultipleParams,
    handler: cache.ChangeHandler<LinePlot> | cache.ChangeHandler<LinePlot[]>,
  ): destructor.Destructor {
    const { request, single } = this.answers;
    if ("key" in params)
      return single.onChange(params.key, handler as cache.ChangeHandler<LinePlot>);
    return request.onChange(
      retrieveReqZ.parse(params),
      handler as cache.ChangeHandler<LinePlot[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<LinePlot> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<LinePlot[]> | undefined;
  getCached(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): cache.Cached<LinePlot> | cache.Cached<LinePlot[]> | undefined {
    const { request, single } = this.answers;
    if ("key" in params) return single.getCached(params.key);
    return request.getCached(retrieveReqZ.parse(params));
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
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
