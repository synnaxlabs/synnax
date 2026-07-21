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
import { project } from "@/project";
import {
  type Action,
  dispatchReqZ,
  rename as renameAction,
} from "@/schematic/actions.gen";
import { bindStore, STORE_KEY } from "@/schematic/store";
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

export class Client {
  readonly symbols: symbol.Client;
  private readonly client: UnaryClient;
  private readonly cache_: cache.Cache;
  private readonly dispatcher_: dispatch.Controller<Key, Schematic, Action>;
  private readonly answers_: {
    single: cache.Answers<Key, Schematic, Key, Schematic>;
    request: cache.Answers<RetrieveRequest, Schematic[], Key, Schematic>;
  };

  constructor(
    client: UnaryClient,
    ontologyClient: ontology.Client,
    engine: cache.Cache,
  ) {
    this.client = client;
    this.symbols = new symbol.Client(client, ontologyClient, engine);
    this.dispatcher_ = bindStore(engine);
    this.cache_ = engine;
    this.answers_ = {
      single: engine.answers({
        name: "schematic",
        table: this.schematicStore,
        fetch: async (query) => [await this.fetchSingle(query)].map((s) => s.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: engine.answers({
        name: "schematics",
        table: this.schematicStore,
        fetch: async (query) => (await this.fetchRequest(query)).map((s) => s.key),
        compose: (records) => records,
        matches: (schematic, query) => requestFilter(query)(schematic),
      }),
    };
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
    rollback.add(this.schematicStore.setMany(optimistic));
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
    this.schematicStore.setMany(res.schematics);
    return isMany ? res.schematics : res.schematics[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    const rollback = new cache.Rollback();
    rollback.add(cache.partialUpdate(this.schematicStore, key, { name }));
    rollback.add(ontology.renameCachedResource(this.cache_, ontologyID(key), name));
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
    opts: dispatch.Options<Schematic, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher_.dispatch(
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
    return await this.dispatcher_.undo(key, this.dispatchSender(key));
  }

  /**
   * Re-applies the document's most recently undone entry. Returns false when
   * nothing is redoable.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher_.redo(key, this.dispatchSender(key));
  }

  /** Whether the document has a live undo entry. */
  hasUndo(key: Key): boolean {
    return this.dispatcher_.hasUndo(key);
  }

  /** Whether the document has a live redo entry. */
  hasRedo(key: Key): boolean {
    return this.dispatcher_.hasRedo(key);
  }

  /**
   * Subscribes to changes in the document's undo/redo stacks. Returns a
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
      "/schematic/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async retrieve(params: RetrieveSingleParams): Promise<Schematic>;
  async retrieve(params: RetrieveMultipleParams): Promise<Schematic[]>;
  async retrieve(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<Schematic | Schematic[]> {
    const isSingle = "key" in params;
    if (isSingle) return await this.answers_.single.retrieve(params.key);
    return await this.answers_.request.retrieve(retrieveReqZ.parse(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a schematic; every other shape delivers the matching
   * schematics.
   */
  onChange(
    params: RetrieveSingleParams,
    handler: cache.ChangeHandler<Schematic>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveMultipleParams,
    handler: cache.ChangeHandler<Schematic[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveSingleParams | RetrieveMultipleParams,
    handler: cache.ChangeHandler<Schematic> | cache.ChangeHandler<Schematic[]>,
  ): destructor.Destructor {
    const answers = this.answers_;
    if ("key" in params)
      return answers.single.onChange(
        params.key,
        handler as cache.ChangeHandler<Schematic>,
      );
    return answers.request.onChange(
      retrieveReqZ.parse(params),
      handler as cache.ChangeHandler<Schematic[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Schematic> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Schematic[]> | undefined;
  getCached(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): cache.Cached<Schematic> | cache.Cached<Schematic[]> | undefined {
    const answers = this.answers_;
    if ("key" in params) return answers.single.getCached(params.key);
    return answers.request.getCached(retrieveReqZ.parse(params));
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    rollback.add(ontology.deleteCachedRelationships(this.cache_, ontologyID(keysArr)));
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
    this.schematicStore.delete(keysArr);
  }

  async copy(params: CopyParams): Promise<Schematic> {
    const res = await this.client.send("/schematic/copy", params, copyReqZ, copyResZ);
    this.schematicStore.set(res.schematic.key, res.schematic);
    return res.schematic;
  }

  private get schematicStore(): cache.Table<Key, Schematic> {
    return this.cache_.table(STORE_KEY);
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
    this.schematicStore.setIfAbsent(schematics);
    return schematics[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Schematic[]> {
    const schematics = await this.execRetrieve(query);
    this.schematicStore.setIfAbsent(schematics);
    return schematics;
  }
}
