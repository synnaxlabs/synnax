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

import {
  Key,
  keyZ,
  NewPolicyPayload,
  newPolicyPayloadZ,
  Policy,
  policyZ,
} from "@/access/payload";
import { NotFoundError } from "@/errors";
import { IDPayload, idZ } from "@/ontology/payload";

const CREATE_ENDPOINT = "/access/policy/create";
const DELETE_ENDPOINT = "/access/policy/delete";
const RETRIEVE_ENDPOINT = "/access/policy/retrieve";

const createReqZ = z.object({ policies: newPolicyPayloadZ.array() });
const createResZ = z.object({ policies: policyZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

const retrieveReqZ = z.object({ subject: idZ });
const retrieveResZ = z.object({
  policies: policyZ.array().optional().default([]),
});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(policies: NewPolicyPayload): Promise<Policy>;

  async create(policies: NewPolicyPayload[]): Promise<Policy[]>;

  async create(
    policies: NewPolicyPayload | NewPolicyPayload[],
  ): Promise<Policy | Policy[]> {
    const single = !Array.isArray(policies);

    const { policies: created } = await sendRequired<
      typeof createReqZ,
      typeof createResZ
    >(
      this.client,
      CREATE_ENDPOINT,
      { policies: toArray(policies) },
      createReqZ,
      createResZ,
    );

    return single ? created[0] : created;
  }

  async retrieve(subject: IDPayload): Promise<Policy | Policy[]> {
    const { policies: retrieved } = await sendRequired<
      typeof retrieveReqZ,
      typeof retrieveResZ
    >(this.client, RETRIEVE_ENDPOINT, { subject: subject }, retrieveReqZ, retrieveResZ);
    if (retrieved.length == 0) {
      throw new NotFoundError(`Policy with subject ${subject} not found`);
    }
    return retrieved.length == 1 ? retrieved[0] : retrieved;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: toArray(keys) },
      deleteReqZ,
      deleteResZ,
    );
  }
}
