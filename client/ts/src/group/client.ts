// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import z from "zod";

import { type Group, groupZ, type Key, keyZ } from "@/group/payload";
import { idZ as ontologyIDZ } from "@/ontology/payload";

export const SET_CHANNEL_NAME = "sy_group_set";
export const DELETE_CHANNEL_NAME = "sy_group_delete";

const resZ = z.object({ group: groupZ });

const createReqZ = z.object({
  parent: ontologyIDZ,
  key: keyZ.optional(),
  name: z.string(),
});

const renameReqZ = z.object({ key: keyZ, name: z.string() });

const deleteReqZ = z.object({ keys: z.array(keyZ) });

export interface CreateArgs extends z.infer<typeof createReqZ> {}

export class Client {
  client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(args: CreateArgs): Promise<Group> {
    const res = await sendRequired(
      this.client,
      "/ontology/create-group",
      args,
      createReqZ,
      resZ,
    );
    return res.group;
  }

  async rename(key: Key, name: string): Promise<void> {
    await sendRequired(
      this.client,
      "/ontology/rename-group",
      { key, name },
      renameReqZ,
      z.object({}),
    );
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired(
      this.client,
      "/ontology/delete-group",
      { keys: array.toArray(keys) },
      deleteReqZ,
      z.object({}),
    );
  }
}
