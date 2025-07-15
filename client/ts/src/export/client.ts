// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { TimeRange } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { channel } from "@/channel";

const csvRequestZ = z.object({
  keys: channel.keyZ.array(),
  timeRange: TimeRange.z.default(TimeRange.MAX),
  channelNames: z.record(channel.keyStringZ, z.string()).optional(),
});

interface CSVRequest extends z.input<typeof csvRequestZ> {}

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async csv(request: CSVRequest): Promise<Response> {
    const [res, err] = await this.client.send("/export/csv", request, csvRequestZ);
    if (err != null) throw err;
    return res;
  }
}
