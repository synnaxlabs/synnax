// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { ontology } from "@/ontology";
import { keyZ, nameZ, type New, newZ, type Payload, payloadZ } from "@/range/payload";

const createResZ = z.object({ ranges: payloadZ.array() });

const createReqZ = z.object({ parent: ontology.idZ.optional(), ranges: newZ.array() });

interface CreateRequest extends z.infer<typeof createReqZ> {}
export interface CreateOptions extends Pick<CreateRequest, "parent"> {}

const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

const renameReqZ = z.object({ key: keyZ, name: nameZ });
const renameResZ = z.object({});

export class Writer {
  client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async rename(key: string, name: string): Promise<void> {
    await sendRequired<typeof renameReqZ, typeof renameResZ>(
      this.client,
      "/range/rename",
      { key, name },
      renameReqZ,
      renameResZ,
    );
  }

  async create(ranges: New[], options?: CreateOptions): Promise<Payload[]> {
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      "/range/create",
      { ranges, ...options },
      createReqZ,
      createResZ,
    );
    return res.ranges;
  }

  async delete(keys: string[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      "/range/delete",
      { keys },
      deleteReqZ,
      deleteResZ,
    );
  }
}
