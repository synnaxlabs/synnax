// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import {
  Key,
  keyZ,
  Policy,
  policyZ,
} from "@/access/payload";
import {toArray} from "@synnaxlabs/x";

export const newPolicyPayloadZ = policyZ.extend({key: keyZ.optional()})

export type NewPolicyPayload = z.infer<typeof newPolicyPayloadZ>

const createReqZ = z.object({ policies: newPolicyPayloadZ.array() });
const createResZ = z.object({ policies: policyZ.array() });

const deleteReqZ = z.object({
  keys: keyZ.array()
});
const deleteResZ = z.object({});

const CREATE_ENDPOINT = "/access/policy/create";
const DELETE_ENDPOINT = "/access/policy/delete";

export class Writer {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(policies: NewPolicyPayload | NewPolicyPayload[]): Promise<Policy[]> {
    const { policies: created } = await sendRequired<
      typeof createReqZ,
      typeof createResZ
    >(this.client, CREATE_ENDPOINT,
      { policies: toArray(policies) },
      createReqZ,
      createResZ
    );
    return created;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      {keys: toArray(keys)},
      deleteReqZ,
      deleteResZ,
    );
  }
}
