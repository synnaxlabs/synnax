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
  type Key,
  keyZ,
  type NewPolicy,
  newPolicyZ,
  type Policy,
  policyZ,
} from "@/access/payload";
import { Retriever } from "@/access/retriever";
import { ontology } from "@/ontology";

const CREATE_ENDPOINT = "/access/policy/create";
const DELETE_ENDPOINT = "/access/policy/delete";

const createReqZ = z.object({ policies: newPolicyZ.array() });
const createResZ = z.object({ policies: policyZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;
  private readonly retriever: Retriever;

  constructor(client: UnaryClient) {
    this.client = client;
    this.retriever = new Retriever(client);
  }

  async create(policies: NewPolicy): Promise<Policy>;

  async create(policies: NewPolicy[]): Promise<Policy[]>;

  async create(policies: NewPolicy | NewPolicy[]): Promise<Policy | Policy[]> {
    const isMany = Array.isArray(policies);
    const { policies: created } = await sendRequired<
      typeof createReqZ,
      typeof createResZ
    >(
      this.client,
      CREATE_ENDPOINT,
      { policies: toArray(policies).map((policy) => newPolicyZ.parse(policy)) },
      createReqZ,
      createResZ,
    );

    return isMany ? created : created[0];
  }

  async retrieve(key: Key): Promise<Policy>;

  async retrieve(keys: Key[]): Promise<Policy[]>;

  async retrieve(keys: Key | Key[]): Promise<Policy | Policy[]> {
    const isMany = Array.isArray(keys);
    const res = await this.retriever.retrieve(keys);
    return isMany ? res : res[0];
  }

  async retrieveFor(id: ontology.CrudeID): Promise<Policy[]>;

  async retrieveFor(ids: ontology.CrudeID[]): Promise<Policy[]>;

  async retrieveFor(ids: ontology.CrudeID | ontology.CrudeID[]): Promise<Policy[]> {
    const newIds = toArray(ids).map((id) => new ontology.ID(id).payload);
    return await this.retriever.retrieveFor(newIds);
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
