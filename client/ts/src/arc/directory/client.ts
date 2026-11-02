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
  ontologyID,
} from "@/arc/directory/types.gen";
import { ontology } from "@/ontology";
import { idZ as ontologyIDZ } from "@/ontology/payload";
import { query } from "@/query";

const createReqZ = z.object({
  parent: keyZ,
  directory: z.object({ key: keyZ.optional(), name: z.string() }),
});
const createResZ = z.object({ directory: directoryZ });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const deleteReqZ = z.object({ keys: z.array(keyZ) });
const moveReqZ = z.object({ to: keyZ, ids: z.array(ontologyIDZ) });

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
  ontology: ontology.Client;
}

const parentRel = (from: ontology.ID, to: ontology.ID): ontology.Relationship => ({
  from,
  type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
  to,
});

/** Client creates, renames, deletes, and moves the directories that hold Arcs. */
export class Client {
  private readonly cfg: ClientConfig;

  constructor(cfg: ClientConfig) {
    this.cfg = cfg;
  }

  /** Creates a directory in the directory parent. */
  async create({ parent, key, name }: CreateParams): Promise<Directory> {
    const res = await this.cfg.unary.send(
      "/arc/directory/create",
      { parent, directory: { key, name } },
      createReqZ,
      createResZ,
    );
    const rel = parentRel(ontologyID(parent), ontologyID(res.directory.key));
    this.cfg.ontology.cache.relationships.set(ontology.relationshipToString(rel), rel);
    return res.directory;
  }

  /** Renames the directory with key. */
  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      this.cfg.ontology.cache.renameResource(ontologyID(key), name),
    ];
    await query.optimistic({
      rollbacks: rename(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/arc/directory/rename",
          { key, name },
          renameReqZ,
          z.object({}),
        ),
    });
    rename();
  }

  /** Deletes the empty directories with keys. */
  async delete(keys: Key | Key[]): Promise<void> {
    await this.cfg.unary.send(
      "/arc/directory/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      z.object({}),
    );
  }

  /**
   * Moves Arcs and directories from the directory from into the directory to. Takes
   * the same arguments as {@link ontology.Client.moveChildren}.
   */
  async move(
    from: ontology.ID,
    to: ontology.ID,
    ...children: ontology.ID[]
  ): Promise<void> {
    const rels = this.cfg.ontology.cache.relationships;
    const move = (): destructor.Destructor[] => [
      ...children.map((child) =>
        rels.delete(ontology.relationshipToString(parentRel(from, child))),
      ),
      ...children.map((child) => {
        const rel = parentRel(to, child);
        return rels.set(ontology.relationshipToString(rel), rel);
      }),
    ];
    await query.optimistic({
      rollbacks: move(),
      commit: async () =>
        await this.cfg.unary.send(
          "/arc/move",
          { to: to.key, ids: children },
          moveReqZ,
          z.object({}),
        ),
    });
    move();
  }
}
