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
import { project } from "@/project";
import { kindOf, reduceAll } from "@/schematic/actions";
import {
  type Action,
  dispatchReqZ,
  rename as renameAction,
  scopedActionZ,
} from "@/schematic/actions.gen";
import { symbol } from "@/schematic/symbol";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Schematic,
  schematicZ,
} from "@/schematic/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_schematic_set";
export const DELETE_CHANNEL_NAME = "sy_schematic_delete";

const deleteReqZ = z.object({ keys: keyZ.array() });

const copyReqZ = z.object({
  key: keyZ,
  name: z.string(),
  snapshot: z.boolean(),
});

const retrieveReqZ = z.object({ keys: keyZ.array() });
const singleRetrieveParamsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

export const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveReqZ]);
export type RetrieveParams = z.input<typeof retrieveParamsZ>;
export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveReqZ>;
export type CopyParams = z.input<typeof copyReqZ>;

interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}

const retrieveResZ = z.object({ schematics: schematicZ.array() });

const createReqZ = z.object({
  project: project.keyZ,
  schematics: schematicZ.array(),
});
const createResZ = z.object({ schematics: schematicZ.array() });

const copyResZ = z.object({ schematic: schematicZ });
const emptyResZ = z.object({});

/**
 * Client-side matching for a request: exact for key sets, the only shape a
 * schematic request carries.
 */
const requestFilter = (req: RetrieveRequest): ((s: Schematic) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (s) => keySet == null || keySet.has(s.key);
};

export class Client extends cache.Reader<
  RetrieveSingleParams,
  RetrieveMultipleParams,
  Key,
  RetrieveRequest,
  Schematic
> {
  readonly symbols: symbol.Client;
  private readonly client: UnaryClient;
  private readonly store: cache.Table<Key, Schematic>;
  private readonly ontology: ontology.Stores;
  private readonly dispatcher: actions.Controller<Key, Schematic, Action>;

  constructor(
    client: UnaryClient,
    ontologyClient: ontology.Client,
    engine: cache.Cache,
    ontologyStores: ontology.Stores,
  ) {
    const store = engine.createTable<Key, Schematic>({ name: "schematics" });
    const dispatcher = new actions.Controller<Key, Schematic, Action>({
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
        name: "schematic",
        table: store,
        fetch: async (query) => [await this.fetchSingle(query)].map((s) => s.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: engine.answers({
        name: "schematics",
        table: store,
        fetch: async (query) => (await this.fetchRequest(query)).map((s) => s.key),
        compose: (records) => records,
        matches: (schematic, query) => requestFilter(query)(schematic),
      }),
      isSingle: (params) => "key" in params,
      normalizeSingle: ({ key }) => key,
      normalizeRequest: (params) => retrieveReqZ.parse(params),
    });
    this.client = client;
    this.symbols = new symbol.Client(client, ontologyClient, engine, ontologyStores);
    this.store = store;
    this.dispatcher = dispatcher;
    this.ontology = ontologyStores;
  }

  async create(
    project: project.Key,
    schematic: New,
    opts?: cache.WriteOptions<Schematic[]>,
  ): Promise<Schematic>;
  async create(
    project: project.Key,
    schematics: New[],
    opts?: cache.WriteOptions<Schematic[]>,
  ): Promise<Schematic[]>;
  async create(
    project: project.Key,
    schematics: New | New[],
    opts: cache.WriteOptions<Schematic[]> = {},
  ): Promise<Schematic | Schematic[]> {
    const isMany = Array.isArray(schematics);
    const optimistic = array.toArray(schematics).map((s) => schematicZ.parse(s));
    const rollback = new cache.Rollback();
    rollback.add(this.store.set(optimistic));
    await opts.onOptimistic?.(optimistic);
    const res = await rollback.guard(
      async () =>
        await this.client.send(
          "/schematic/create",
          { project, schematics: optimistic },
          createReqZ,
          createResZ,
        ),
    );
    this.store.set(res.schematics);
    return isMany ? res.schematics : res.schematics[0];
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
   * Applies actions to the cached document and sends them to the server,
   * recording an undoable entry. Returns false without side effects when the
   * document isn't cached. Rolls back the local apply and rethrows on send
   * failure.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: actions.Options<Schematic, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher.dispatch(
      key,
      array.toArray(actions),
      this.dispatchSender(key),
      opts.preprocess,
    );
  }

  /**
   * Reverts the document's most recent undoable entry. Returns false when
   * nothing is undoable.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher.undo(key, this.dispatchSender(key));
  }

  /**
   * Re-applies the document's most recently undone entry. Returns false when
   * nothing is redoable.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo(key, this.dispatchSender(key));
  }

  /** Whether the document has a live undo entry. */
  hasUndo(key: Key): boolean {
    return this.dispatcher.hasUndo(key);
  }

  /** Whether the document has a live redo entry. */
  hasRedo(key: Key): boolean {
    return this.dispatcher.hasRedo(key);
  }

  /**
   * Subscribes to changes in the document's undo/redo stacks. Returns a
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
      "/schematic/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    rollback.add(
      ontology.deleteCachedRelationships(this.ontology, ontologyID(keysArr)),
    );
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send(
          "/schematic/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    );
    this.store.delete(keysArr);
  }

  async copy(params: CopyParams): Promise<Schematic> {
    const res = await this.client.send("/schematic/copy", params, copyReqZ, copyResZ);
    this.store.set(res.schematic);
    return res.schematic;
  }

  private async execRetrieve(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<Schematic[]> {
    const res = await this.client.send(
      "/schematic/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return res.schematics;
  }

  // Dispatch mutates documents server-side, so a cached copy is only as fresh
  // as the streamer. Fetches always hit the network; setIfAbsent hydrates the
  // table without clobbering a doc holding locally replayed edits.
  private async fetchSingle(query: Key): Promise<Schematic> {
    const schematics = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("Schematic", query, schematics, true);
    this.store.setIfAbsent(schematics);
    return schematics[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Schematic[]> {
    const schematics = await this.execRetrieve(query);
    this.store.setIfAbsent(schematics);
    return schematics;
  }
}
