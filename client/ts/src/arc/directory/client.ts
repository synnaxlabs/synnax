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
import { resource } from "@/resource";

export const SET_CHANNEL_NAME = "sy_arc_directory_set";
export const DELETE_CHANNEL_NAME = "sy_arc_directory_delete";

const createReqZ = z.object({
  parent: keyZ,
  directory: z.object({ key: keyZ.optional(), name: z.string() }),
});
const createResZ = z.object({ directory: directoryZ });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const deleteReqZ = z.object({ keys: z.array(keyZ) });
const moveReqZ = z.object({ from: keyZ, to: keyZ, ids: z.array(resource.idZ) });
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
}

/** Client retrieves, creates, renames, deletes, and moves the directories of Arcs. */
export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Directory
> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Directory>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Directory>({
      name: "Arc directories",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, directoryZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
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
  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    validateIdentifier(name);
    const rename = () => query.partialUpdate(this.store, key, { name });
    await query.optimistic({
      rollbacks: [rename()],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/arc/directory/rename",
          { key, name },
          renameReqZ,
          emptyResZ,
        ),
    });
    rename();
  }

  /** Deletes the empty directories with keys. */
  async delete(keys: Key | Key[]): Promise<void> {
    const payload = { keys: array.toArray(keys) };
    await this.cfg.unary.send("/arc/directory/delete", payload, deleteReqZ, emptyResZ);
    this.store.delete(payload.keys);
  }

  /**
   * Builds a request that moves Arcs and directories from the directory from into the
   * directory to. Its cache write moves the cached ones.
   */
  moveRequest(
    from: resource.ID,
    to: resource.ID,
    ...children: resource.ID[]
  ): batch.Request<z.infer<typeof moveReqZ>> {
    return {
      endpoint: "/arc/move",
      payload: { from: from.key, to: to.key, ids: children },
      apply: () => this.applyMove(to, children),
    };
  }

  private applyMove(to: resource.ID, children: resource.ID[]): destructor.Destructor {
    const undos = children.map(({ type, key }) =>
      type === "arc"
        ? query.partialUpdate(this.cfg.arcs, key, { directory: to.key })
        : query.partialUpdate(this.store, key, { parent: to.key }),
    );
    return () => undos.forEach((undo) => undo());
  }

  /** Moves Arcs and directories from the directory from into the directory to. */
  async move(
    from: resource.ID,
    to: resource.ID,
    ...children: resource.ID[]
  ): Promise<void> {
    const { endpoint, payload } = this.moveRequest(from, to, ...children);
    await query.optimistic({
      rollbacks: [this.applyMove(to, children)],
      commit: async () =>
        await this.cfg.unary.send(endpoint, payload, moveReqZ, emptyResZ),
    });
    this.applyMove(to, children);
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
