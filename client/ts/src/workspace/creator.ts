// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { type Workspace, workspaceZ } from "./payload";

const crudeWorkspaceZ = workspaceZ.partial({ key: true });

export type CrudeWorkspace = z.infer<typeof crudeWorkspaceZ>;

export const reqZ = z.object({
  workspaces: crudeWorkspaceZ.partial({ key: true }).array(),
});

export type Request = z.infer<typeof reqZ>;

export const resZ = z.object({
  workspaces: workspaceZ.array(),
});

export type Response = z.infer<typeof resZ>;

export class Creator {
  private static readonly ENDPOINT = "/workspace/create";

  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspaces: CrudeWorkspace[]): Promise<Workspace[]> {
    return await this.execute({ workspaces });
  }

  private async execute(request: Request): Promise<Workspace[]> {
    const [res, err] = await this.client.send(Creator.ENDPOINT, request, resZ);
    if (err != null) throw err;
    return res.workspaces;
  }
}
