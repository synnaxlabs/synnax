// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { toArray } from "@synnaxlabs/x/toArray";
import { z } from "zod";

import { keyZ, type Params, type Policy, policyZ } from "@/access/payload";
import { ontology } from "@/ontology";
import { nullableArrayZ } from "@/util/zod";

const reqZ = z.object({
  keys: keyZ.array().optional(),
  subjects: ontology.idZ.array().optional(),
});

type Request = z.infer<typeof reqZ>;

const resZ = z.object({
  policies: nullableArrayZ(policyZ),
});

export class Retriever {
  private static readonly ENDPOINT = "/access/policy/retrieve";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(params: Params): Promise<Policy[]> {
    return await this.execute({ keys: toArray(params) });
  }

  async retrieveFor(ids: ontology.IDPayload[]): Promise<Policy[]> {
    return await this.execute({ subjects: ids });
  }

  private async execute(req: Request): Promise<Policy[]> {
    const [res, err] = await this.client.send<typeof reqZ, typeof resZ>(
      Retriever.ENDPOINT,
      req,
      reqZ,
      resZ,
    );
    if (err != null) throw err;
    return res.policies;
  }
}
