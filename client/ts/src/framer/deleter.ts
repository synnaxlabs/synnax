// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { TimeRange } from "@synnaxlabs/x";
import { z } from "zod";

import { channel } from "@/channel";

const reqZ = z.object({
  keys: channel.keyZ.array().optional(),
  bounds: TimeRange.z,
  names: channel.nameZ.array().optional(),
});
interface Request extends z.infer<typeof reqZ> {}

const resZ = z.object({});

export class Deleter {
  /*
  Deleter is used to delete a time range of telemetry from the data engine.
   */
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async delete(props: Request): Promise<void> {
    await sendRequired<typeof reqZ, typeof resZ>(
      this.client,
      "/frame/delete",
      props,
      reqZ,
      resZ,
    );
  }
}
