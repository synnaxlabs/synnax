// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import {
  RangeParams,
  RangePayload,
  analyzeRangeParams,
  rangeKey,
  rangePayload,
} from "./payload";

const reqZ = z.object({
  keys: z.array(rangeKey).optional(),
  names: z.array(z.string()).optional(),
  term: z.string().optional(),
});

type Request = z.infer<typeof reqZ>;

const resZ = z.object({
  ranges: z.array(rangePayload),
});

export class RangeRetriever {
  private readonly ENDPOINT = "/range/retrieve";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(params: RangeParams): Promise<RangePayload[]> {
    const { normalized, variant } = analyzeRangeParams(params);
    const res = await this.execute({ [variant]: normalized });
    return res;
  }

  async search(term: string): Promise<RangePayload[]> {
    return await this.execute({ term });
  }

  private async execute(request: Request): Promise<RangePayload[]> {
    const [res, err] = await this.client.send(this.ENDPOINT, request, resZ);
    if (err != null) throw err;
    return res.ranges;
  }
}
