// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {type UnaryClient} from "@synnaxlabs/freighter";
import {toArray} from "@synnaxlabs/x/toArray";

import {Key, OntologyIDType, Policy, policyZ} from "@/access/payload";
import {Retriever} from "@/access/retriever";
import {NewPolicyPayload, Writer} from "@/access/writer";
import {NewLabelPayload} from "@/label/writer";

export class Client {
  private readonly client: UnaryClient;
  readonly retriever: Retriever;
  readonly writer: Writer;

  constructor(
    client: UnaryClient,
  ) {
    this.retriever = new Retriever(client);
    this.client = client;
    this.writer = new Writer(client);
  }

  async create(policies: NewPolicyPayload): Promise<Policy>;

  async create(policies: NewPolicyPayload[]): Promise<Policy[]>;

  async create(
    policies: NewPolicyPayload | NewPolicyPayload[],
  ): Promise<Policy | Policy[]> {
    const single = !Array.isArray(policies);
    let toCreate = toArray(policies);
    let created: Policy[] = [];
    created = created.concat(await this.writer.create(toCreate));
    return single ? created[0] : created;
  }

  async retrieve(
    subject: OntologyIDType,
  ): Promise<Policy[]> {
    return await this.retriever.retrieve(subject);
  }

  async delete(keys: Key | Key[]): Promise<void> {
    if (Array.isArray(keys)) {
      return await this.writer.delete(keys);
    }
    return await this.writer.delete([keys])
  }
}
