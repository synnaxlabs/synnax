// Copyright 2023 Synnax Labs, Inc.
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

import { type Schematic, type Params, schematicRemoteZ } from "@/workspace/schematic/payload";

const reqZ = z.object({
  keys: z.string().array(),
});

type Request = z.infer<typeof reqZ>;

const resZ = z.object({
  schematics: schematicRemoteZ.array(),
});

export class Retriever {
  private readonly ENDPOINT = "/workspace/schematic/retrieve";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(params: Params): Promise<Schematic[]> {
    return (await sendRequired(
      this.client,
      this.ENDPOINT,
      { keys: toArray(params) },
      reqZ,
      resZ,
    )).schematics;
  }
}
