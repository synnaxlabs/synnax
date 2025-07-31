// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";

import { ALLOW_ALL_ONTOLOGY_TYPE, ONTOLOGY_TYPE } from "@/access/policy/ontology";
import { type Key, type New, type Policy } from "@/access/policy/payload";
import { Retriever } from "@/access/policy/retriever";
import { Writer } from "@/access/policy/writer";
import { type ontology } from "@/ontology";

export class Client {
  private readonly retriever: Retriever;
  private readonly writer: Writer;

  constructor(client: UnaryClient) {
    this.retriever = new Retriever(client);
    this.writer = new Writer(client);
  }

  async create(policy: New): Promise<Policy>;
  async create(policies: New[]): Promise<Policy[]>;
  async create(policies: New | New[]): Promise<Policy | Policy[]> {
    const isMany = Array.isArray(policies);
    const createdPolicies = await this.writer.create(policies);
    return isMany ? createdPolicies : createdPolicies[0];
  }

  async retrieve(key: Key): Promise<Policy>;
  async retrieve(keys: Key[]): Promise<Policy[]>;
  async retrieve(keys: Key | Key[]): Promise<Policy | Policy[]> {
    const isMany = Array.isArray(keys);
    const res = await this.retriever.retrieve({ keys: array.toArray(keys) });
    return isMany ? res : res[0];
  }

  async retrieveFor(subject: ontology.ID): Promise<Policy[]>;
  async retrieveFor(subjects: ontology.ID[]): Promise<Policy[]>;
  async retrieveFor(subjects: ontology.ID | ontology.ID[]): Promise<Policy[]> {
    return await this.retriever.retrieve({ subjects: array.toArray(subjects) });
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
  async delete(keys: Key | Key[]): Promise<void> {
    await this.writer.delete(keys);
  }
}

export const ontologyID = (key: Key): ontology.ID => ({ type: ONTOLOGY_TYPE, key });

export const ALLOW_ALL_ONTOLOGY_ID: ontology.ID = {
  type: ALLOW_ALL_ONTOLOGY_TYPE,
  key: "",
};
