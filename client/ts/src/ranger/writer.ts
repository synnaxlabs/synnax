// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { type ontology } from "@/ontology";
import { nameZ } from "@/ranger/payload";
import { keyZ, type New, newZ, type Payload, payloadZ } from "@/ranger/types.gen";

const createResZ = z.object({ ranges: payloadZ.array() });

const parentRefZ = z.object({ key: keyZ });
const createNewZ = newZ.extend({ parent: parentRefZ.optional() });
const createReqZ = z.object({ ranges: createNewZ.array() });

export interface CreateOptions {
  parent?: ontology.ID;
}

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
    await this.client.send("/range/rename", { key, name }, renameReqZ, renameResZ);
  }

  async create(ranges: New[], options?: CreateOptions): Promise<Payload[]> {
    const parent = options?.parent != null ? { key: options.parent.key } : undefined;
    const stripped = ranges.map(({ parent: _ignored, ...r }: New & { parent?: unknown }) => r);
    const withParent =
      parent == null ? stripped : stripped.map((r) => ({ ...r, parent }));
    const res = await this.client.send(
      "/range/create",
      { ranges: withParent },
      createReqZ,
      createResZ,
    );
    return res.ranges;
  }

  async delete(keys: string[]): Promise<void> {
    await this.client.send("/range/delete", { keys }, deleteReqZ, deleteResZ);
  }
}
