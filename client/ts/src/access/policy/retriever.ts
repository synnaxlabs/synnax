// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { keyZ, type Policy, policyZ } from "@/access/policy/payload";
import { ontology } from "@/ontology";
import { nullableArrayZ } from "@/util/zod";

const reqZ = z.object({
  keys: keyZ.array().optional(),
  subjects: ontology.idZ.array().optional(),
});
type Request = z.infer<typeof reqZ>;
const resZ = z.object({ policies: nullableArrayZ(policyZ) });

const ENDPOINT = "/access/policy/retrieve";

export class Retriever {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(req: Request): Promise<Policy[]> {
    const res = await sendRequired<typeof reqZ, typeof resZ>(
      this.client,
      ENDPOINT,
      req,
      reqZ,
      resZ,
    );
    return res.policies;
  }
}
