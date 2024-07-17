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

import { nullableArrayZ } from "@/util/zod";
import {
  OntologyID,
  OntologyIDType,
  Policy,
  policyZ
} from "@/access/payload";
import {NotFoundError} from "@/errors";

const reqZ = z.object({subject: OntologyID})

type Request = z.infer<typeof reqZ>;

const resZ = z.object({
  policies: policyZ.array().optional().default([]),
});

export class Retriever {
  private readonly ENDPOINT = "/access/policy/retrieve";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(id: OntologyIDType): Promise<Policy[]> {
    const res = await this.execute({subject: id});
    if(res.length == 0){
      throw new NotFoundError(`Policy with subject ${id} not found`)
    }
    return res
  }

  private async execute(request: Request): Promise<Policy[]> {
    const [res, err] = await this.client.send(
      this.ENDPOINT,
      request,
      reqZ,
      resZ,
    );
    if (err != null) throw err;
    return res.policies;
  }
}
