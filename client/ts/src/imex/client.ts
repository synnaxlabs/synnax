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

export const envelopeZ = z.looseObject({
  version: z.union([z.number(), z.string()]),
  type: z.string(),
  key: z.string(),
  name: z.string().default(""),
});
export type Envelope = z.input<typeof envelopeZ>;

const importReqZ = z.object({
  parent: ontology.idZ.optional(),
  resources: envelopeZ.array(),
});
const importResZ = z.object({});

const exportReqZ = z.object({ resources: ontology.idZ.array() });
const exportResZ = z.object({ resources: envelopeZ.array() });

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async import_(
    parent: ontology.ID | null,
    resources: Envelope[],
  ): Promise<void> {
    await sendRequired(
      this.client,
      "/import",
      { parent: parent ?? undefined, resources },
      importReqZ,
      importResZ,
    );
  }

  async export_(resources: ontology.ID[]): Promise<Envelope[]> {
    const res = await sendRequired(
      this.client,
      "/export",
      { resources },
      exportReqZ,
      exportResZ,
    );
    return res.resources;
  }
}
