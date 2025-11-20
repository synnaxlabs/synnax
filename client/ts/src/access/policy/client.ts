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

export type RetrieveSingleParams = z.input<typeof keyRetrieveRequestZ>;
export type RetrieveMultipleParams = z.input<typeof listRetrieveArgsZ>;

const retrieveArgsZ = z.union([keyRetrieveRequestZ, listRetrieveArgsZ]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

const retrieveResZ = z.object({ policies: array.nullableZ(policyZ) });

const singleCreateArgsZ = newZ.transform((p) => ({ policies: [p] }));
export type SingleCreateArgs = z.input<typeof singleCreateArgsZ>;

export const multipleCreateArgsZ = newZ.array().transform((policies) => ({ policies }));

export const createArgsZ = z.union([singleCreateArgsZ, multipleCreateArgsZ]);
export type CreateArgs = z.input<typeof createArgsZ>;

const createResZ = z.object({ policies: policyZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(policy: New): Promise<Policy>;
  async create(policies: New[]): Promise<Policy[]>;
  async create(policies: CreateArgs): Promise<Policy | Policy[]> {
    const isMany = Array.isArray(policies);
    const res = await sendRequired<typeof createArgsZ, typeof createResZ>(
      this.client,
      "/access/policy/create",
      policies,
      createArgsZ,
      createResZ,
    );
    return isMany ? res.policies : res.policies[0];
  }

  async retrieve(args: RetrieveSingleParams): Promise<Policy>;
  async retrieve(args: RetrieveMultipleParams): Promise<Policy[]>;
  async retrieve(args: RetrieveArgs): Promise<Policy | Policy[]> {
    const isSingle = "key" in args;
    const res = await sendRequired<typeof retrieveArgsZ, typeof retrieveResZ>(
      this.client,
      "/access/policy/retrieve",
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
      "/access/policy/delete",
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
