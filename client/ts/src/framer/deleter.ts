// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, UnaryClient } from "@synnaxlabs/freighter";
import { TimeRange } from "@synnaxlabs/x";
import { z } from "zod";

import { keyZ } from "@/channel/payload";

const reqZ = z.object({
  keys: keyZ.array().optional(),
  bounds: TimeRange.z,
  names: z.string().array().optional(),
});

type RequestProps = z.infer<typeof reqZ>;

const resZ = z.object({});

const ENDPOINT = "/frame/delete";

export class Deleter {
  /*
  Deleter is used to delete a time range of telemetry from the data engine.
   */
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async delete(props: RequestProps): Promise<void> {
    await sendRequired<typeof reqZ, typeof resZ>(
      this.client,
      ENDPOINT,
      props,
      reqZ,
      resZ,
    );
  }
}
