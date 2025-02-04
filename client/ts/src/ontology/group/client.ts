// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";

import { Group } from "@/ontology/group/group";
import { type Payload } from "@/ontology/group/payload";
import { Writer } from "@/ontology/group/writer";
import { type ID } from "@/ontology/payload";

export class Client {
  private readonly creator: Writer;

  constructor(unary: UnaryClient) {
    this.creator = new Writer(unary);
  }

  async create(parent: ID, name: string, key?: string): Promise<Group> {
    return this.sugar(await this.creator.create(parent, name, key));
  }

  async rename(key: string, name: string): Promise<void> {
    return await this.creator.rename(key, name);
  }

  async delete(...keys: string[]): Promise<void> {
    return await this.creator.delete(keys);
  }

  private sugar(payload: Payload): Group {
    return new Group(payload.name, payload.key);
  }
}
