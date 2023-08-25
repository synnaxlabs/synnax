// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { Payload, payloadZ } from "@/ontology/group/payload";
import { ID } from "@/ontology/payload";

const resZ = z.object({
  group: payloadZ,
});

export class Creator {
  private static readonly ENDPOINT = "/ontology/create-group";
  private static readonly ENDPOINT_RENAME = "/ontology/rename-group";
  client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(parent: ID, name: string): Promise<Payload> {
    const req = { parent, name };
    const [res, err] = await this.client.send(Creator.ENDPOINT, req, resZ);
    if (err != null) throw err;
    return res.group;
  }

  async rename(key: string, name: string): Promise<void> {
    const req = { key, name };
    const [, err] = await this.client.send(Creator.ENDPOINT_RENAME, req, z.object({}));
    if (err != null) throw err;
  }
}
