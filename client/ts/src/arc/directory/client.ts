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
import z from "zod";

import { actions } from "@/actions";
import { setDirectory } from "@/arc/actions.gen";
import { createOf, reduceAll } from "@/arc/directory/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/arc/directory/actions.gen";
import {
  type Directory,
  directoryZ,
  type Key,
  keyZ,
  retrieveRequestZ,
  retrieveResponseZ,
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
import { validateIdentifier } from "@/arc/identifier";
import { type Arc, type Key as ArcKey } from "@/arc/types.gen";
import { type batch } from "@/batch";
import { NotFoundError } from "@/errors";
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
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(
        cfg.unary,
        "/arc/directory/dispatch",
        dispatchReqZ,
        dispatchResZ,
      ),
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
   * Retrieves every directory above the directories that is not one of them, each once.
   * The chain ends at the root.
   * @throws {NotFoundError} if a directory or one above it does not exist.
   */
  async retrieveAncestors(keys: Key | Key[]): Promise<Directory[]> {
    await this.store.track(async (since) => {
      this.store.ingest(await this.execRetrieve({}), { since });
    });
    const ancestors = this.getCachedAncestors(keys);
    if (ancestors != null) return ancestors;
    const listed = array.toArray(keys).join(", ");
    throw new NotFoundError(`Arc directory ${listed} or one above it not found`);
  }

  /**
   * Returns every cached directory above the directories that is not one of them, each
   * once, or undefined when a directory on the chain is not cached.
   */
  getCachedAncestors(keys: Key | Key[]): Directory[] | undefined {
    const starts = array.toArray(keys);
    const visited = new Set<Key>(starts);
    const ancestors: Directory[] = [];
    for (const key of starts) {
      const start = this.store.get(key);
      if (start == null) return undefined;
      let { parent } = start;
      while (parent != null && !visited.has(parent)) {
        const d = this.store.get(parent);
        if (d == null) return undefined;
        visited.add(d.key);
        ancestors.push(d);
        parent = d.parent;
      }
    }
    return ancestors;
  }

  /**
   * Calls the handler with the cached ancestors of the directories after each change to
   * a cached directory, since a rename or a move of any directory can change them.
   */
  onAncestorsChange(
    keys: Key | Key[],
    handler: query.ChangeHandler<Directory[]>,
  ): destructor.Destructor {
    return this.store.subscribe(() => handler(this.getCachedAncestors(keys)));
  }

  /**
   * Builds a request that creates a directory in the directory parent. A request with a
   * key writes the directory to the cache.
   * @throws {ValidationError} if name is not an identifier.
   */
  createRequest({
    parent,
    key,
    name,
  }: CreateParams): batch.Request<z.infer<typeof createReqZ>> {
    validateIdentifier(name);
    const payload = { parent, directory: { key, name } };
    if (key == null) return { endpoint: "/arc/directory/create", payload };
    return {
      endpoint: "/arc/directory/create",
      payload,
      apply: () => this.store.set({ key, name, parent }),
    };
  }

  /**
   * Creates a directory in the directory parent. A create with a key writes the
   * directory to the cache first, and removes it if the Core rejects the create.
   * @throws {ValidationError} if name is not an identifier.
   */
  async create(params: CreateParams): Promise<Directory> {
    const { endpoint, payload, apply } = this.createRequest(params);
    const { directory } = await query.optimistic({
      rollbacks: apply == null ? [] : [apply()],
      commit: async () =>
        await this.cfg.unary.send(endpoint, payload, createReqZ, createResZ),
    });
    this.store.set(directory);
    return directory;
  }

  /**
   * Renames the directory with key.
   * @throws {ValidationError} if name is not an identifier.
   */
  async rename(key: Key, name: string): Promise<void> {
    validateIdentifier(name);
    await this.dispatch(key, set({ path: ["name"], value: name }));
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
   * Builds the requests that move Arcs and directories into the directory to. Their
   * cache writes move the cached ones.
   */
  moveRequests(to: resource.ID, ...children: resource.ID[]): batch.Request[] {
    return children.map(({ type, key }) =>
      type === "arc"
        ? {
            endpoint: "/arc/dispatch",
            payload: { key, actions: [setDirectory({ directory: to.key })] },
            apply: () => query.partialUpdate(this.cfg.arcs, key, { directory: to.key }),
          }
        : {
            endpoint: "/arc/directory/dispatch",
            payload: { key, actions: [set({ path: ["parent"], value: to.key })] },
            apply: () => query.partialUpdate(this.store, key, { parent: to.key }),
          },
    );
  }

  /**
   * Moves Arcs and directories into the directory to in one transaction.
   * @throws {ValidationError} if the directory would hold two entries with one name.
   */
  async move(to: resource.ID, ...children: resource.ID[]): Promise<void> {
    await this.cfg.batch.send(this.moveRequests(to, ...children));
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
