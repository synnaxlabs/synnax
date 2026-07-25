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
import { project } from "@/project";
import { query } from "@/query";
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

export const SET_CHANNEL_NAME = "sy_schematic_set";
export const DELETE_CHANNEL_NAME = "sy_schematic_delete";

const deleteReqZ = z.object({ keys: keyZ.array() });

const copyReqZ = z.object({
  key: keyZ,
  name: z.string(),
  snapshot: z.boolean(),
});

const retrieveReqZ = z.object({
  keys: keyZ.array(),
  ignoreNotFoundError: z.boolean().optional(),
});
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

export class Client extends query.Retriever<typeof retrieveReqZ, Key, Schematic> {
  readonly symbols: symbol.Client;
  private readonly client: UnaryClient;
  private readonly store: query.Table<Key, Schematic>;
  private readonly ontology: ontology.Stores;
  private readonly dispatcher: actions.Controller<Key, Schematic, Action>;

  constructor(
    client: UnaryClient,
    ontologyClient: ontology.Client,
    cache: query.Cache,
    ontologyStores: ontology.Stores,
  ) {
    // Dispatch mutates documents server-side, so fetched copies never clobber
    // a doc holding locally replayed edits: the table hydrates if-absent.
    const store = cache.createTable<Key, Schematic>({
      name: "schematics",
      hydrate: "if-absent",
      fetch: async (keys) =>
        await this.execRetrieve({ keys, ignoreNotFoundError: true }),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    const dispatcher = new actions.Controller<Key, Schematic, Action>({
      store,
      onError: cache.onError,
      reduce: reduceAll,
      kindOf,
    });
    cache.listen(dispatcher.listener(SET_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "schematic",
      table: store,
      request: {
        schema: retrieveReqZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (schematic, req) => requestFilter(req)(schematic),
      },
    });
    this.client = client;
    this.symbols = new symbol.Client(client, ontologyClient, cache, ontologyStores);
    this.store = store;
    this.dispatcher = dispatcher;
    this.ontology = ontologyStores;
  }

  async create(
    project: project.Key,
    schematic: New,
    opts?: query.WriteOptions<Schematic[]>,
  ): Promise<Schematic>;
  async create(
    project: project.Key,
    schematics: New[],
    opts?: query.WriteOptions<Schematic[]>,
  ): Promise<Schematic[]>;
  async create(
    project: project.Key,
    schematics: New | New[],
    opts: query.WriteOptions<Schematic[]> = {},
  ): Promise<Schematic | Schematic[]> {
    const isMany = Array.isArray(schematics);
    const optimistic = array.toArray(schematics).map((s) => schematicZ.parse(s));
    const rollback = new destructor.Chain();
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

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rollback = new destructor.Chain();
    rollback.add(query.partialUpdate(this.store, key, { name }));
    rollback.add(ontology.renameCachedResource(this.ontology, ontologyID(key), name));
    await opts.onOptimistic?.();
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
    opts?: actions.Options<Schematic, Action>,
  ): Promise<boolean>;
  /**
   * Legacy raw-send form used by pre-cutover flux. Removed in the
   * pluto rebind; new callers use the two-argument controller form above.
   */
  async dispatch(key: Key, dispatchKey: string, actions: Action[]): Promise<void>;
  async dispatch(
    key: Key,
    actionsOrKey: Action | Action[] | string,
    opts: actions.Options<Schematic, Action> | Action[] = {},
  ): Promise<boolean | void> {
    if (typeof actionsOrKey === "string")
      return await this.sendDispatch(key, actionsOrKey, opts as Action[]);
    return await this.dispatcher.dispatch(
      key,
      array.toArray(actionsOrKey),
      this.dispatchSender(key),
      (opts as actions.Options<Schematic, Action>).preprocess,
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

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new destructor.Chain();
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

  /** Subscribes to every schematic delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  async copy(params: CopyParams): Promise<Schematic> {
    const res = await this.client.send("/schematic/copy", params, copyReqZ, copyResZ);
    this.store.set(res.schematic);
    return res.schematic;
  }

  private async execRetrieve(params: RetrieveMultipleParams): Promise<Schematic[]> {
    const res = await this.client.send(
      "/schematic/retrieve",
      params,
      retrieveReqZ,
      retrieveResZ,
    );
    return res.schematics;
  }
}
