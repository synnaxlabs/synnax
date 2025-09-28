// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { keyZ, type User, userZ } from "@/user/payload";

const reqZ = z.object({
  keys: keyZ.array().optional(),
  usernames: z.string().array().optional(),
});
interface Request extends z.infer<typeof reqZ> {}
const resZ = z.object({ users: array.nullableZ(userZ) });
const ENDPOINT = "/user/retrieve";

export class Retriever {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(req: Request): Promise<User[]> {
    const res = await sendRequired<typeof reqZ, typeof resZ>(
      this.client,
      ENDPOINT,
      req,
      reqZ,
      resZ,
    );
    return res.users;
  }
}
