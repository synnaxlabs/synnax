// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import z from "zod";

import { actions } from "@/actions";
import { setDirectory } from "@/arc/actions.gen";
import { createOf, reduceAll } from "@/arc/directory/actions";
import {
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
  update,
} from "@/arc/directory/actions.gen";
import {
  type Action,
  type Directory,
  directoryZ,
  type Key,
  keyZ,
  retrieveRequestZ,
  retrieveResponseZ,
  type Update,
} from "@/arc/directory/types.gen";
import { LINES } from "@/arc/directory/where";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
  shortcutZ,
} from "@/arc/directory/where.gen";
import { type Arc, type Key as ArcKey } from "@/arc/types.gen";
import { type batch } from "@/batch";
import { query } from "@/query";
import { type resource } from "@/resource";

export const ACTION_CHANNEL_NAME = "sy_arc_directory_action_set";
export const DELETE_CHANNEL_NAME = "sy_arc_directory_delete";

const createReqZ = z.object({
  parent: keyZ,
  directory: z.object({ key: keyZ.optional(), name: z.string() }),
});
const createResZ = z.object({ directory: directoryZ });
const deleteReqZ = z.object({ keys: z.array(keyZ) });
const emptyResZ = z.object({});

const retrieveMultiParamsZ = z.union([
  query.keysRequestZ(keyZ),
  retrieveRequestZ,
  retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
]);

export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;

export interface CreateParams {
  /** parent is the key of the directory to create the new directory in. */
  parent: Key;
  /** key is the key of the new directory. The Core generates one when absent. */
  key?: Key;
  /** name is the name of the new directory. */
  name: string;
}

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  /** arcs holds the cached Arcs, whose directory a move changes. */
  arcs: query.Table<ArcKey, Arc>;
  batch: batch.Client;
}

/** Client retrieves, creates, renames, deletes, and moves the directories of Arcs. */
export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Directory
> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Directory>;
  private readonly dispatcher: actions.Controller<Key, Directory, Action>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Directory>({
      name: "Arc directories",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to a directory reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Directory, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      send: actions.sender(
        cfg.unary,
        "/arc/directory/dispatch",
        dispatchReqZ,
        dispatchResZ,
      ),
      sendAll: actions.batchSender(cfg.unary, "/arc/directory/dispatch", dispatchResZ),
      reduce: reduceAll,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "Arc directory",
      table: store,
      shortcut: shortcutZ,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (d, req) => matches(LINES, d, req.where),
        serverComputed: (req) => serverComputed(LINES, req.where),
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
  }

  /** Retrieves the directory that holds every other directory and Arc. */
  async retrieveRoot(): Promise<Directory> {
    const [root] = await this.retrieve({ nested: false });
    return root;
  }

  /**
   * Creates a directory in the directory parent. A create with a key writes the
   * directory to the cache first, and removes it if the Core rejects the create.
   */
  async create({ parent, key, name }: CreateParams): Promise<Directory> {
    const { directory } = await query.optimistic({
      rollbacks: key == null ? [] : [this.store.set({ key, name, parent })],
      commit: async () =>
        await this.cfg.unary.send(
          "/arc/directory/create",
          { parent, directory: { key, name } },
          createReqZ,
          createResZ,
        ),
    });
    this.store.set(directory);
    return directory;
  }

  /** Renames the directory with key. */
  async rename(key: Key, name: string): Promise<void> {
    await this.dispatch(key, set({ path: ["name"], value: name }));
  }

  /**
   * Applies each update to its cached directory until the Core echoes it. Several
   * updates apply in one transaction, all or none.
   * @throws {ValidationError} if a directory cannot take its update.
   * @throws {patch.StaleError} if an update names a list entry that is missing.
   */
  async update(updates: actions.Updates<Key, Update>): Promise<void> {
    await this.dispatcher.update(updates, update);
  }

  /**
   * Applies the actions to the cached directory until the Core echoes them, and sends
   * them to the Core to apply in one transaction. Actions hold the snake_case form that
   * the Core stores. A refused send drops the actions from the cached directory.
   * @throws {Error} if the cached directory cannot hold an action, before any send.
   * @throws {ValidationError} if the directory cannot hold an action.
   * @throws {NotFoundError} if the directory does not exist.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<void> {
    await this.dispatcher.send(key, array.toArray(actions));
  }

  /** Deletes the empty directories with keys. */
  async delete(keys: Key | Key[]): Promise<void> {
    const payload = { keys: array.toArray(keys) };
    await this.cfg.unary.send("/arc/directory/delete", payload, deleteReqZ, emptyResZ);
    this.store.delete(payload.keys);
  }

  /**
   * Moves Arcs and directories into the directory to in one transaction.
   * @throws {ValidationError} if the directory would hold two entries with one name.
   */
  async move(to: resource.ID, ...children: resource.ID[]): Promise<void> {
    await this.cfg.batch.send(
      children.map(({ type, key }) =>
        type === "arc"
          ? {
              endpoint: "/arc/dispatch",
              payload: { key, actions: [setDirectory({ directory: to.key })] },
              apply: () =>
                query.partialUpdate(this.cfg.arcs, key, { directory: to.key }),
            }
          : {
              endpoint: "/arc/directory/dispatch",
              payload: { key, actions: [set({ path: ["parent"], value: to.key })] },
              apply: () => query.partialUpdate(this.store, key, { parent: to.key }),
            },
      ),
    );
  }

  private async execRetrieve(
    req: z.input<typeof retrieveRequestZ>,
  ): Promise<Directory[]> {
    const res = await this.cfg.unary.send(
      "/arc/directory/retrieve",
      req,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.directories;
  }
}
