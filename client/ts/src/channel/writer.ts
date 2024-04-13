// Copyright 2023 Synnax Labs, Inc.
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
  type Payload,
  payload,
  newPayload,
  type NewPayload,
  keyZ,
} from "@/channel/payload";

const createReqZ = z.object({ channels: newPayload.array() });
const createResZ = z.object({ channels: payload.array() });

const deleteReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
});
const deleteResZ = z.object({});

const CREATE_ENDPOINT = "/channel/create";
const DELETE_ENDPOINT = "/channel/delete";

export type DeleteProps = z.input<typeof deleteReqZ>;

export class Writer {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(channels: NewPayload[]): Promise<Payload[]> {
    return (
      await sendRequired<typeof createReqZ, typeof createResZ>(
        this.client,
        CREATE_ENDPOINT,
        { channels },
        createReqZ,
        createResZ,
      )
    ).channels;
  }

  async delete(props: DeleteProps): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      props,
      deleteReqZ,
      deleteResZ,
    );
  }
}
