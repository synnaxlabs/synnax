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
import { binary } from "@synnaxlabs/x/binary";
import { z } from "zod";

import { channel } from "@/channel";
import { type Transport } from "@/transport";

const responseTypeZ = z.enum(["csv"]);

const readRequestZ = z.object({
  keys: channel.keyZ.array(),
  timeRange: TimeRange.z,
  channelNames: z.record(channel.keyZ, z.string()).optional(),
  responseType: responseTypeZ,
});
export interface ReadRequest extends z.input<typeof readRequestZ> {}

export class Reader {
  private readonly csvClient: UnaryClient;

  constructor(transport: Transport) {
    this.csvClient = transport.withDecoder(binary.CSV_CODEC);
  }

  async read(req: ReadRequest): Promise<Response> {
    const [res, err] = await this.csvClient.send("/frame/read", req, readRequestZ);
    if (err != null) throw err;
    return res;
  }
}
