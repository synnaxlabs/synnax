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

import { type NewPayload, type Payload, payloadZ } from "@/ranger/payload";

const resZ = z.object({
  ranges: z.array(payloadZ),
});

export class Creator {
  private static readonly ENDPOINT = "/range/create";
  client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(ranges: NewPayload[]): Promise<Payload[]> {
    const [res, err] = await this.client.send(Creator.ENDPOINT, { ranges }, resZ);
    if (err != null) throw err;
    return res.ranges;
  }
}
