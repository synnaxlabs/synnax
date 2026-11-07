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
import { matches, retrieveInputZ, serverComputed } from "@/arc/directory/where.gen";
import { type Arc, type Key as ArcKey } from "@/arc/types.gen";
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
      request: {
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
    const [root] = await this.retrieve({ where: { nested: false } });
    return root;
  }

  /** Creates a directory in the directory parent. */
  async create({ parent, key, name }: CreateParams): Promise<Directory> {
    const res = await this.cfg.unary.send(
      "/arc/directory/create",
      { parent, directory: { key, name } },
      createReqZ,
      createResZ,
    );
    this.store.set(res.directory);
    return res.directory;
  }

  /** Renames the directory with key. */
  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
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

  /** Moves Arcs and directories from the directory from into the directory to. */
  async move(
    from: resource.ID,
    to: resource.ID,
    ...children: resource.ID[]
  ): Promise<void> {
    const move = (): destructor.Destructor[] =>
      children.map(({ type, key }) =>
        type === "arc"
          ? query.partialUpdate(this.cfg.arcs, key, { directory: to.key })
          : query.partialUpdate(this.store, key, { parent: to.key }),
      );
    await query.optimistic({
      rollbacks: move(),
      commit: async () =>
        await this.cfg.unary.send(
          "/arc/move",
          { from: from.key, to: to.key, ids: children },
          moveReqZ,
          emptyResZ,
        ),
    });
    move();
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
