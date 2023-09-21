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

import { pidZ, type PID } from "@/workspace/pid/payload";

export const crudePIDz = pidZ.partial({ key: true });

export type CrudePID = z.infer<typeof crudePIDz>;

const reqZ = z.object({
  pids: crudePIDz.array(),
});

type Request = z.infer<typeof reqZ>;

const resZ = z.object({
  pids: pidZ.array(),
});

export class Creator {
  private readonly ENDPOINT = "/workspace/pid/create";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(pid: CrudePID): Promise<PID> {
    return await this.execute({ pids: toArray(pid) });
  }

  private async execute(request: Request): Promise<PID> {
    const [res, err] = await this.client.send(this.ENDPOINT, request, resZ);
    if (err != null) throw err;
    return res.pids[0];
  }
}
