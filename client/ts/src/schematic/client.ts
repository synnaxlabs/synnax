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

const MOUNT_SCOPE = "schematic.mounts";

/**
 * Client-side approximation of the server's matching for a request: exact for
 * key sets, the only shape a schematic request carries.
 */
const requestFilter = (req: RetrieveRequest): ((s: Schematic) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (s) => keySet == null || keySet.has(s.key);
};

export class Client {
  readonly symbols: symbol.Client;
  private readonly client: UnaryClient;
  private readonly engine_?: cache.Engine;
  private readonly dispatcher_?: dispatch.Controller<Key, Schematic, Action>;
  private readonly queries_?: {
    single: cache.Queries<Key, Schematic>;
    request: cache.Queries<RetrieveRequest, Schematic[]>;
  };

  constructor(
    client: UnaryClient,
    ontologyClient: ontology.Client,
    engine?: cache.Engine,
  ) {
    this.client = client;
    this.symbols = new symbol.Client(client, ontologyClient, engine);
    if (engine == null) return;
    this.dispatcher_ = bindStore(engine);
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "schematic",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "schematics",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
      }),
    };
  }

  private get dispatcher(): dispatch.Controller<Key, Schematic, Action> {
    if (this.dispatcher_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.dispatcher_;
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
    if (this.writes != null) rollback.add(this.writes.set(optimistic));
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
    this.writes?.set(res.schematics);
    return isMany ? res.schematics : res.schematics[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    const rollback = new cache.Rollback();
    const writes = this.writes;
    if (this.engine_ != null && writes != null) {
      rollback.add(cache.partialUpdate(writes, key, { name }));
      rollback.add(ontology.renameCachedResource(this.engine_, ontologyID(key), name));
    }
    await rollback.guard(
      async () => await this.sendDispatch(key, "", [renameAction({ name })]),
    );
  }

  /**
   * Applies actions to the cached document and sends them to the server,
   * recording an undoable entry. Returns false without side effects when the
   * document isn't cached. Rolls back the local apply and rethrows on send
   * failure.
   * @throws when the cache was disabled at client construction.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: dispatch.Options<Schematic, Action> = {},
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
   * Reverts the document's most recent undoable entry. Returns false when
   * nothing is undoable.
   * @throws when the cache was disabled at client construction.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher.undo("", key, this.dispatchSender(key));
  }

  /**
   * Re-applies the document's most recently undone entry. Returns false when
   * nothing is redoable.
   * @throws when the cache was disabled at client construction.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo("", key, this.dispatchSender(key));
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
    if (this.queries_ == null) {
      const schematics = await this.execRetrieve(params);
      checkForMultipleOrNoResults("Schematic", params, schematics, isSingle);
      return isSingle ? schematics[0] : schematics;
    }
    if (isSingle) return await this.queries_.single.retrieve(params.key);
    return await this.queries_.request.retrieve(retrieveReqZ.parse(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a schematic; every other shape delivers the matching
   * schematics.
   * @throws when the cache was disabled at client construction.
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
    const queries = this.requireQueries();
    if ("key" in params)
      return queries.single.onChange(
        params.key,
        handler as cache.ChangeHandler<Schematic>,
      );
    return queries.request.onChange(
      retrieveReqZ.parse(params),
      handler as cache.ChangeHandler<Schematic[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   * @throws when the cache was disabled at client construction.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Schematic> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Schematic[]> | undefined;
  getCached(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): cache.Cached<Schematic> | cache.Cached<Schematic[]> | undefined {
    const queries = this.requireQueries();
    if ("key" in params) return queries.single.getCached(params.key);
    return queries.request.getCached(retrieveReqZ.parse(params));
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    if (this.engine_ != null)
      rollback.add(
        ontology.deleteCachedRelationships(this.engine_, ontologyID(keysArr)),
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
    this.writes?.delete(keysArr);
  }

  async copy(params: CopyParams): Promise<Schematic> {
    const res = await this.client.send("/schematic/copy", params, copyReqZ, copyResZ);
    this.writes?.set(res.schematic);
    return res.schematic;
  }

  private get writes(): cache.UnaryStore<Key, Schematic> | undefined {
    return this.engine_?.store(STORE_KEY);
  }

  private get schematicStore(): cache.UnaryStore<Key, Schematic> {
    return this.requireEngine().store(STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  private get schematicEvents(): cache.UnaryStore<Key, Schematic> {
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
  // store without clobbering a doc holding locally replayed edits. Answers
  // read back from the store so replayed edits and reference identity win.
  private async fetchSingle(query: Key): Promise<Schematic> {
    const schematics = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("Schematic", query, schematics, true);
    this.schematicStore.setIfAbsent(schematics);
    return this.schematicStore.get(query) ?? schematics[0];
  }

  private mountSingle({ query, update, remove }: cache.MountParams<Key, Schematic>) {
    return [
      this.schematicEvents.onSet((schematic) => {
        if (schematic.key === query) update(schematic);
      }),
      this.schematicEvents.onDelete((key) => {
        if (key === query) remove(this.schematicStore.getTombstone(key)?.corpse);
      }),
    ];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Schematic[]> {
    const schematics = await this.execRetrieve(query);
    this.schematicStore.setIfAbsent(schematics);
    return schematics.map((s) => this.schematicStore.get(s.key) ?? s);
  }

  private mountRequest({
    query,
    update,
  }: cache.MountParams<RetrieveRequest, Schematic[]>) {
    const matches = requestFilter(query);
    return [
      this.schematicEvents.onSet((schematic) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.some((s) => s.key === schematic.key);
          if (!matches(schematic))
            return existing ? prev.filter((s) => s.key !== schematic.key) : prev;
          if (existing)
            return prev.map((s) => (s.key === schematic.key ? schematic : s));
          return [...prev, schematic];
        });
      }),
      this.schematicEvents.onDelete((key) => {
        update((prev) => prev?.filter((s) => s.key !== key));
      }),
    ];
  }
}
