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
import { z } from "zod/v4";

import {
  type Key,
  keyZ,
  type New,
  newZ,
  type Policy,
  policyZ,
} from "@/access/policy/payload";

const createReqZ = z.object({ policies: policyZ.partial({ key: true }).array() });
const createResZ = z.object({ policies: policyZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

const CREATE_ENDPOINT = "/access/policy/create";
const DELETE_ENDPOINT = "/access/policy/delete";

export class Writer {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(policies: New | New[]): Promise<Policy[]> {
    const parsedPolicies = newZ.array().parse(array.toArray(policies));
    const req = parsedPolicies.map((policy) => ({
      objects: array.toArray(policy.objects),
      actions: array.toArray(policy.actions),
      subjects: array.toArray(policy.subjects),
    }));
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { policies: req },
      createReqZ,
      createResZ,
    );
    return res.policies;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: array.toArray(keys) },
      deleteReqZ,
      deleteResZ,
    );
  }
}
