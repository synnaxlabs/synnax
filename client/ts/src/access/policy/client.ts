// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { toArray } from "@synnaxlabs/x/toArray";

import { type Key, type NewPolicy, type Policy } from "@/access/policy/payload";
import { Retriever } from "@/access/policy/retriever";
import { Writer } from "@/access/policy/writer";
import { ontology } from "@/ontology";

export class Client {
  private readonly retriever: Retriever;
  private readonly writer: Writer;

  constructor(client: UnaryClient) {
    this.retriever = new Retriever(client);
    this.writer = new Writer(client);
  }

  async create(policy: NewPolicy): Promise<Policy>;

  async create(policies: NewPolicy[]): Promise<Policy[]>;

  async create(policies: NewPolicy | NewPolicy[]): Promise<Policy | Policy[]> {
    const isMany = Array.isArray(policies);
    const createdPolicies = await this.writer.create(policies);
    return isMany ? createdPolicies : createdPolicies[0];
  }

  async retrieve(key: Key): Promise<Policy>;

  async retrieve(keys: Key[]): Promise<Policy[]>;

  async retrieve(keys: Key | Key[]): Promise<Policy | Policy[]> {
    const isMany = Array.isArray(keys);
    const res = await this.retriever.retrieve({ keys: toArray(keys) });
    return isMany ? res : res[0];
  }

  async retrieveFor(subject: ontology.CrudeID): Promise<Policy[]>;

  async retrieveFor(subjects: ontology.CrudeID[]): Promise<Policy[]>;

  async retrieveFor(
    subjects: ontology.CrudeID | ontology.CrudeID[],
  ): Promise<Policy[]> {
    const newIds = toArray(subjects).map((id) => new ontology.ID(id).payload);
    return await this.retriever.retrieve({ subjects: newIds });
  }

  async delete(key: Key): Promise<void>;

  async delete(keys: Key[]): Promise<void>;

  async delete(keys: Key | Key[]): Promise<void> {
    await this.writer.delete(keys);
  }
}
