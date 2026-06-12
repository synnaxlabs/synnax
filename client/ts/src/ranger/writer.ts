// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { nameZ } from "@/ranger/payload";
import { keyZ, newZ, type Payload, payloadZ } from "@/ranger/types.gen";

const parentRefZ = z.object({ key: keyZ });
const createPayloadZ = newZ.extend({ parent: parentRefZ.optional() });
export type CreatePayload = z.input<typeof createPayloadZ>;

const createReqZ = z.object({ ranges: createPayloadZ.array() });
const createResZ = z.object({ ranges: payloadZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

const renameReqZ = z.object({ key: keyZ, name: nameZ });
const renameResZ = z.object({});

export class Writer {
  client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async rename(key: string, name: string): Promise<void> {
    await this.client.send("/range/rename", { key, name }, renameReqZ, renameResZ);
  }

  async create(ranges: CreatePayload[]): Promise<Payload[]> {
    const res = await this.client.send(
      "/range/create",
      { ranges },
      createReqZ,
      createResZ,
    );
    return res.ranges;
  }

  async delete(keys: string[]): Promise<void> {
    await this.client.send("/range/delete", { keys }, deleteReqZ, deleteResZ);
  }
}
