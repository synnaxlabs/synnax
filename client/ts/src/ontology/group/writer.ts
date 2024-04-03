// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { type Payload, payloadZ } from "@/ontology/group/payload";
import { idZ, type ID } from "@/ontology/payload";

const resZ = z.object({
  group: payloadZ,
});

const createReqZ = z.object({
  parent: idZ,
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

  async create(parent: ID, name: string): Promise<Payload> {
    const [res, err] = await this.client.send(Writer.ENDPOINT, {parent, name},createReqZ, resZ);
    if (err != null) throw err;
    return res.group;
  }

  async rename(key: string, name: string): Promise<void> {
    const req = { key, name };
    const [, err] = await this.client.send(Writer.ENDPOINT_RENAME, req, renameReqZ, z.object({}));
    if (err != null) throw err;
  }

  async delete(keys: string[]): Promise<void> {
    const req = { keys };
    const [, err] = await this.client.send(Writer.ENDPOINT_DELETE, req, deleteReqZ, z.object({}));
    if (err != null) throw err;
  }
}
