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

import {
  type Params,
  type Workspace,
  keyZ,
  workspaceRemoteZ,
} from "@/workspace/payload";

const reqZ = z.object({
  keys: keyZ.array().optional(),
  search: z.string().optional(),
  author: z.string().uuid().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

type Request = z.infer<typeof reqZ>;

const resZ = z.object({
  workspaces: workspaceRemoteZ.array(),
});

export class Retriever {
  private static readonly ENDPOINT = "/workspace/retrieve";

  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(params: Params): Promise<Workspace[]> {
    const normalized = toArray(params);
    const res = await this.execute({ keys: normalized });
    return res;
  }

  async retrieveByAuthor(author: string): Promise<Workspace[]> {
    return await this.execute({ author });
  }

  async search(term: string): Promise<Workspace[]> {
    return await this.execute({ search: term });
  }

  async page(offset: number, limit: number): Promise<Workspace[]> {
    return await this.execute({ offset, limit });
  }

  private async execute(request: Request): Promise<Workspace[]> {
    const [res, err] = await this.client.send(Retriever.ENDPOINT, request, resZ);
    if (err != null) throw err;
    return res.workspaces;
  }
}
