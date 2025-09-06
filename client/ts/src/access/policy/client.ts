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

import {
  type Key,
  keyZ,
  type New,
  newZ,
  type Policy,
  policyZ,
} from "@/access/policy/payload";
import { ontology } from "@/ontology";
import { nullableArrayZ } from "@/util/zod";

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  subjects: ontology.idZ.array().optional(),
});

const keyRetrieveRequestZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const listRetrieveArgsZ = z.union([
  z
    .object({ for: ontology.idZ })
    .transform(({ for: forId }) => ({ subjects: [forId] })),
  z
    .object({ for: ontology.idZ.array() })
    .transform(({ for: forIds }) => ({ subjects: forIds })),
  retrieveRequestZ,
]);

export type SingleRetrieveArgs = z.input<typeof keyRetrieveRequestZ>;
export type ListRetrieveArgs = z.input<typeof listRetrieveArgsZ>;

const retrieveArgsZ = z.union([keyRetrieveRequestZ, listRetrieveArgsZ]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

const retrieveResZ = z.object({ policies: nullableArrayZ(policyZ) });

const createReqZ = z.object({ policies: policyZ.partial({ key: true }).array() });
const createResZ = z.object({ policies: policyZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

const RETRIEVE_ENDPOINT = "/access/policy/retrieve";
const CREATE_ENDPOINT = "/access/policy/create";
const DELETE_ENDPOINT = "/access/policy/delete";

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(policy: New): Promise<Policy>;
  async create(policies: New[]): Promise<Policy[]>;
  async create(policies: New | New[]): Promise<Policy | Policy[]> {
    const isMany = Array.isArray(policies);
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
    return isMany ? res.policies : res.policies[0];
  }

  async retrieve(args: SingleRetrieveArgs): Promise<Policy>;
  async retrieve(args: ListRetrieveArgs): Promise<Policy[]>;
  async retrieve(args: RetrieveArgs): Promise<Policy | Policy[]> {
    const isSingle = "key" in args;
    const res = await sendRequired<typeof retrieveArgsZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    return isSingle ? res.policies[0] : res.policies;
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
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

export const ontologyID = (key: Key): ontology.ID => ({ type: "policy", key });

export const ALLOW_ALL_ONTOLOGY_ID: ontology.ID = {
  type: "allow_all",
  key: "",
};
