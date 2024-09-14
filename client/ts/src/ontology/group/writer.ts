// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { groupZ, type Payload } from "@/ontology/group/payload";
import { type ID, idZ } from "@/ontology/payload";

const resZ = z.object({
  group: groupZ,
});

const createReqZ = z.object({
  parent: idZ,
  key: z.string().uuid().optional(),
  name: z.string(),
});

const renameReqZ = z.object({
  key: z.string(),
  name: z.string(),
});

const deleteReqZ = z.object({
  keys: z.array(z.string()),
});

export class Writer {
  private static readonly ENDPOINT = "/ontology/create-group";
  private static readonly ENDPOINT_RENAME = "/ontology/rename-group";
  private static readonly ENDPOINT_DELETE = "/ontology/delete-group";
  client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(parent: ID, name: string, key?: string): Promise<Payload> {
    const res = await sendRequired(
      this.client,
      Writer.ENDPOINT,
      { parent, name, key },
      createReqZ,
      resZ,
    );
    return res.group;
  }

  async rename(key: string, name: string): Promise<void> {
    await sendRequired(
      this.client,
      Writer.ENDPOINT_RENAME,
      { key, name },
      renameReqZ,
      z.object({}),
    );
  }

  async delete(keys: string[]): Promise<void> {
    await sendRequired(
      this.client,
      Writer.ENDPOINT_DELETE,
      { keys },
      deleteReqZ,
      z.object({}),
    );
  }
}
