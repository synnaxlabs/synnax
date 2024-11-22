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

import { type Params, type Table, tableRemoteZ } from "@/workspace/table/payload";

const reqZ = z.object({ keys: z.string().array() });
const resZ = z.object({ tables: tableRemoteZ.array() });

export class Retriever {
  private readonly ENDPOINT = "/workspace/table/retrieve";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(tables: Params): Promise<Table[]> {
    return (
      await sendRequired(
        this.client,
        this.ENDPOINT,
        { keys: toArray(tables) },
        reqZ,
        resZ,
      )
    ).tables;
  }
}
