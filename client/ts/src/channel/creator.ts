// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { type NewPayload,newPayload, type Payload, payload } from "@/channel/payload";

const reqZ = z.object({ channels: newPayload.array() });

const resZ = z.object({ channels: payload.array() });

export class Creator {
  private static readonly ENDPOINT = "/channel/create";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(channels: NewPayload[]): Promise<Payload[]> {
    const [res, err] = await this.client.send<typeof reqZ, typeof resZ>(
      Creator.ENDPOINT,
      { channels },
      reqZ,
      resZ,
    );
    if (err != null) throw err;
    return res.channels;
  }
}
