// Copyright 2025 Synnax Labs, Inc.
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

import { keyZ, type Label, labelZ, type Params } from "@/label/payload";
import { ontology } from "@/ontology";
import { nullableArrayZ } from "@/util/zod";

const reqZ = z.object({
  keys: keyZ.array().optional(),
  for: ontology.idZ.optional(),
  search: z.string().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

type Request = z.infer<typeof reqZ>;

const resZ = z.object({
  labels: nullableArrayZ(labelZ),
});

export class Retriever {
  private static readonly ENDPOINT = "/label/retrieve";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(labels: Params): Promise<Label[]> {
    const normalized = toArray(labels);
    return await this.execute({ keys: normalized });
  }

  async retrieveFor(id: ontology.ID): Promise<Label[]> {
    return await this.execute({ for: id });
  }

  async search(term: string): Promise<Label[]> {
    return await this.execute({ search: term });
  }

  async page(offset: number, limit: number): Promise<Label[]> {
    return await this.execute({ offset, limit });
  }

  private async execute(req: Request): Promise<Label[]> {
    const [res, err] = await this.client.send<typeof reqZ, typeof resZ>(
      Retriever.ENDPOINT,
      req,
      reqZ,
      resZ,
    );
    if (err != null) throw err;
    return res.labels;
  }
}
