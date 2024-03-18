// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient, sendRequired } from "@synnaxlabs/freighter";
import { z } from "zod";

import { rackKeyZ } from "@/hardware/rack/payload";
import { type Task, taskZ } from "@/hardware/task/payload";

const retrieveReqZ = z.object({
  rack: rackKeyZ.optional(),
  keys: z.string().array().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

const rerieveResS = z.object({
  tasks: z.union([taskZ.array(), z.null().transform(() => [])]),
});

export type RetrieveRequest = z.infer<typeof retrieveReqZ>;

const RETRIEVE_ENDPOINT = "/hardware/task/retrieve";

export class Retriever {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(params: RetrieveRequest): Promise<Task[]> {
    const res = await sendRequired<typeof retrieveReqZ, typeof rerieveResS>(
      this.client,
      RETRIEVE_ENDPOINT,
      params,
      rerieveResS,
    );
    return res.tasks;
  }

  async search(term: string): Promise<Task[]> {
    const res = await sendRequired<typeof retrieveReqZ, typeof rerieveResS>(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: [term] },
      rerieveResS,
    );
    return res.tasks;
  }

  async page(offset: number, limit: number): Promise<Task[]> {
    const res = await sendRequired<typeof retrieveReqZ, typeof rerieveResS>(
      this.client,
      RETRIEVE_ENDPOINT,
      { offset, limit },
      rerieveResS,
    );
    return res.tasks;
  }
}
