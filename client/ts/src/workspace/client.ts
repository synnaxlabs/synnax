// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";

import { type CrudeWorkspace, Creator } from "@/workspace/creator";
import { Deleter } from "@/workspace/deleter";
import { pid } from "@/workspace/pid";
import { Retriever } from "@/workspace/retriever";

import { type Key, type Workspace } from "./payload";

export class Client {
  readonly pid: pid.Client;
  private readonly retriever: Retriever;
  private readonly creator: Creator;
  private readonly deleter: Deleter;

  constructor(client: UnaryClient) {
    this.pid = new pid.Client(client);
    this.retriever = new Retriever(client);
    this.creator = new Creator(client);
    this.deleter = new Deleter(client);
  }

  async retrieve(key: Key): Promise<Workspace>;

  async retrieve(keys: Key[]): Promise<Workspace[]>;

  async retrieve(keys: Key | Key[]): Promise<Workspace | Workspace[]> {
    const isMany = Array.isArray(keys);
    const res = await this.retriever.retrieve(keys);
    return isMany ? res : res[0];
  }

  async create(...workspaces: CrudeWorkspace[]): Promise<void> {
    await this.creator.create(workspaces);
  }

  async delete(...keys: Key[]): Promise<void> {
    await this.deleter.delete(keys);
  }
}
