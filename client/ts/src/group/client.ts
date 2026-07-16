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

import { type cache } from "@/cache";
import { bindStore, STORE_KEY } from "@/group/store";
import { type Group, groupZ, type Key, keyZ } from "@/group/types.gen";
import { idZ as ontologyIDZ } from "@/ontology/payload";

const resZ = z.object({ group: groupZ });

const createReqZ = z.object({
  parent: ontologyIDZ,
  key: keyZ.optional(),
  name: z.string(),
});

const renameReqZ = z.object({ key: keyZ, name: z.string() });

const deleteReqZ = z.object({ keys: z.array(keyZ) });

export interface CreateParams extends z.infer<typeof createReqZ> {}

export class Client {
  client: UnaryClient;
  private readonly store_?: cache.Store<Key, Group>;

  constructor(client: UnaryClient, engine?: cache.Engine) {
    this.client = client;
    if (engine == null) return;
    bindStore(engine);
    this.store_ = engine.store(STORE_KEY);
  }

  /**
   * Read surface of the group cache.
   * @throws when the cache was disabled at client construction.
   */
  get store(): cache.Store<Key, Group> {
    if (this.store_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.store_;
  }

  async create(params: CreateParams): Promise<Group> {
    const res = await this.client.send(
      "/ontology/create-group",
      params,
      createReqZ,
      resZ,
    );
    return res.group;
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.client.send(
      "/ontology/rename-group",
      { key, name },
      renameReqZ,
      z.object({}),
    );
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await this.client.send(
      "/ontology/delete-group",
      { keys: array.toArray(keys) },
      deleteReqZ,
      z.object({}),
    );
  }
}
