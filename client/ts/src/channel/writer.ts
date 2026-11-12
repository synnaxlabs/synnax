// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { type DataType } from "@synnaxlabs/x";
import { z } from "zod";

import { keyZ, nameZ, type New, type Payload, payloadZ } from "@/channel/types.gen";

const createReqZ = z.object({
  channels: payloadZ.array(),
  retrieveIfNameExists: z.boolean(),
});
const createResZ = z.object({ channels: payloadZ.array() });

const deleteReqZ = z.object({
  keys: keyZ.array().optional(),
  names: nameZ.array().optional(),
});
const deleteResZ = z.object({});

export interface DeleteProps extends z.input<typeof deleteReqZ> {}

export class Writer {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(channels: New[], retrieveIfNameExists: boolean): Promise<Payload[]> {
    const { channels: created } = await this.client.send(
      "/channel/create",
      {
        channels: channels.map((c) => ({
          ...c,
          dataType: c.dataType as DataType,
        })),
        retrieveIfNameExists,
      },
      createReqZ,
      createResZ,
    );
    return created;
  }

  async delete(props: DeleteProps): Promise<void> {
    await this.client.send("/channel/delete", props, deleteReqZ, deleteResZ);
  }
}
