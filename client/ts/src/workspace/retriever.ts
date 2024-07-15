// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { toArray } from "@synnaxlabs/x/toArray";
import { z } from "zod";

import { nullableArrayZ } from "@/util/zod";
import {
  keyZ,
  type Params,
  type Workspace,
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

const resZ = z.object({ workspaces: nullableArrayZ(workspaceRemoteZ) });

export class Retriever {
  private static readonly ENDPOINT = "/workspace/retrieve";

  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(params: Params): Promise<Workspace[]> {
    const normalized = toArray(params);
    return await this.execute({ keys: normalized });
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
    const res = await sendRequired(
      this.client,
      Retriever.ENDPOINT,
      request,
      reqZ,
      resZ,
    );
    return res.workspaces;
  }
}
