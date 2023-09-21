// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { toArray } from "@synnaxlabs/x";
import { z } from "zod";

import { type Params, keyZ } from "@/workspace/payload";

const reqZ = z.object({
  keys: keyZ.array(),
});

type Request = z.infer<typeof reqZ>;

const resZ = z.object({});

export class Deleter {
  private static readonly ENDPOINT = "/workspace/delete";

  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async delete(params: Params): Promise<void> {
    const normalized = toArray(params);
    await this.execute({ keys: normalized });
  }

  private async execute(request: Request): Promise<void> {
    const [, err] = await this.client.send(Deleter.ENDPOINT, request, resZ);
    if (err != null) throw err;
  }
}
