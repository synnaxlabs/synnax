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

import {
  keyZ,
  type NewPayload,
  newPayloadZ,
  type Payload,
  payloadZ,
} from "@/ranger/payload";

const createResZ = z.object({
  ranges: payloadZ.array(),
});

const createReqZ = z.object({
  ranges: newPayloadZ.array(),
});

const deleteReqZ = z.object({
  keys: keyZ.array(),
});

const deleteResZ = z.object({});

const renameReqZ = z.object({
  key: keyZ,
  name: z.string(),
});

const renameResZ = z.object({});

const CREATE_ENDPOINT = "/range/create";
const DELETE_ENDPOINT = "/range/delete";
const RENAME_ENDPOINT = "/range/rename";

export class Writer {
  client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async rename(key: string, name: string): Promise<void> {
    await sendRequired<typeof renameReqZ, typeof renameResZ>(
      this.client,
      RENAME_ENDPOINT,
      { key, name },
      renameReqZ,
      renameResZ,
    );
  }

  async create(ranges: NewPayload[]): Promise<Payload[]> {
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { ranges },
      createReqZ,
      createResZ,
    );
    return res.ranges;
  }

  async delete(keys: string[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys },
      deleteReqZ,
      deleteResZ,
    );
  }
}
