// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FileTransport, type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, filter, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { group } from "@/group";
import { project } from "@/project";
import { query } from "@/query";
import { createOf, encode, kindOf, reduceAll } from "@/schematic/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/schematic/actions.gen";
import { symbol } from "@/schematic/symbol";
import {
  type Key,
  keyZ,
  type New,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  type Schematic,
  schematicZ,
} from "@/schematic/types.gen";
import {
  compareCursors,
  cursor,
  type Lines,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
} from "@/schematic/where.gen";

export const SET_CHANNEL_NAME = "sy_schematic_set";
export const DELETE_CHANNEL_NAME = "sy_schematic_delete";

const deleteReqZ = z.object({ keys: keyZ.array() });

const copyReqZ = z.object({ key: keyZ, name: z.string() });

const LINES: Lines = { ...group.MEMBER_LINES, ...project.OWNER_LINES };

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
export type CopyParams = z.input<typeof copyReqZ>;

const createReqZ = z.object({
  dispatchKey: z.string(),
  schematics: schematicZ.array(),
});
const createResZ = z.object({ schematics: schematicZ.array() });

const copyResZ = z.object({ schematic: schematicZ });
const emptyResZ = z.object({});

export interface ClientConfig {
  unary: UnaryClient;
  file: FileTransport;
  cache: query.Cache;
  groupStore: query.Table<group.Key, group.Group>;
}

export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Schematic
> {
  readonly symbols: symbol.Client;
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Schematic>;
  /**
   * Builds requests that move schematics between groups. A move into a group of another
   * project also moves the schematic into that project.
   */
  readonly setGroupRequest: group.MoveRequest<Key>;
  /** Builds requests that move schematics out of their group and into a project. */
  readonly setProjectRequest: project.MoveRequest<Key>;
  private readonly dispatcher: actions.Controller<Key, Schematic, Action>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    // Dispatch mutates documents server-side, so fetched copies never clobber
    // a doc holding locally replayed edits: the table hydrates if-absent.
    const store = cache.createTable<Key, Schematic>({
      name: "schematics",
      hydrate: "if-absent",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    const dispatcher = new actions.Controller<Key, Schematic, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      reduce: reduceAll,
      send: actions.sender(
        cfg.unary,
        "/schematic/dispatch",
        dispatchReqZ,
        dispatchResZ,
      ),
      encode,
      kindOf,
      createOf,
    });
    cache.listen(dispatcher.listener(SET_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "schematic",
      table: store,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (schematic, req) => matches(LINES, schematic, req.where),
        serverComputed: (req) => serverComputed(LINES, req.where),
      },
    });
    this.cfg = cfg;
    this.symbols = new symbol.Client(cfg);
    this.store = store;
    this.setGroupRequest = project.createGroupMoveRequest(
      "/schematic/dispatch",
      store,
      dispatchResZ,
      reduceAll,
    );
    this.setProjectRequest = project.createMoveRequest(
      "/schematic/dispatch",
      (keys, to) => project.moveCached(store, keys, to),
    );
    this.dispatcher = dispatcher;
  }

  async create(
    schematic: New,
    opts?: query.WriteOptions<Schematic[]>,
  ): Promise<Schematic>;
  async create(
    schematics: New[],
    opts?: query.WriteOptions<Schematic[]>,
  ): Promise<Schematic[]>;
  async create(
    schematics: New | New[],
    opts: query.WriteOptions<Schematic[]> = {},
  ): Promise<Schematic | Schematic[]> {
    const isMany = Array.isArray(schematics);
    const optimistic = array
      .toArray(schematics)
      .map((s) => zod.parse(schematicZ, s, { label: "schematic" }));
    const res = await query.optimistic({
      rollbacks: [this.store.set(optimistic)],
      onOptimistic: () => opts.onOptimistic?.(optimistic),
      commit: async () =>
        await this.dispatcher.create(
          optimistic.map(({ key }) => key),
          async (dispatchKey) =>
            await this.cfg.unary.send(
              "/schematic/create",
              { dispatchKey, schematics: optimistic },
              createReqZ,
              createResZ,
            ),
        ),
    });
    this.store.set(res.schematics);
    return isMany ? res.schematics : res.schematics[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatcher.send(key, [set({ path: ["name"], value: name })]);
  }

  /**
   * Applies actions to the cached document and sends them to the server, recording an
   * undoable entry. Returns false without side effects when the document isn't cached.
   * Rolls back the local apply and rethrows on send failure.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: actions.Options<Schematic, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher.dispatch(key, array.toArray(actions), opts);
  }

  /**
   * Reverts the document's most recent undoable entry. Returns false when
   * nothing is undoable.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher.undo(key);
  }

  /**
   * Re-applies the document's most recently undone entry. Returns false when
   * nothing is redoable.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo(key);
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
          "/schematic/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  /** Subscribes to every schematic delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  async copy(params: CopyParams): Promise<Schematic> {
    const res = await this.cfg.unary.send(
      "/schematic/copy",
      params,
      copyReqZ,
      copyResZ,
    );
    this.store.set(res.schematic);
    return res.schematic;
  }

  private async execRetrieve(params: RetrieveRequest): Promise<Schematic[]> {
    const res = await this.cfg.unary.send(
      "/schematic/retrieve",
      params,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.schematics;
  }
}
