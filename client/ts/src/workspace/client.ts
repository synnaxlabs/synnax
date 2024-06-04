// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { type UnknownRecord } from "@synnaxlabs/x/record";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";

import { linePlot } from "@/workspace/lineplot";
import { type Key, type Workspace } from "@/workspace/payload";
import { Retriever } from "@/workspace/retriever";
import { schematic } from "@/workspace/schematic";
import { type NewWorkspace, Writer } from "@/workspace/writer";

export class Client implements AsyncTermSearcher<string, Key, Workspace> {
  readonly type = "workspace";
  readonly schematic: schematic.Client;
  readonly linePlot: linePlot.Client;
  private readonly retriever: Retriever;
  private readonly writer: Writer;

  constructor(client: UnaryClient) {
    this.schematic = new schematic.Client(client);
    this.linePlot = new linePlot.Client(client);
    this.retriever = new Retriever(client);
    this.writer = new Writer(client);
  }

  async search(term: string): Promise<Workspace[]> {
    return await this.retriever.search(term);
  }

  async retrieveByAuthor(author: string): Promise<Workspace[]> {
    return await this.retriever.retrieveByAuthor(author);
  }

  async retrieve(key: Key): Promise<Workspace>;

  async retrieve(keys: Key[]): Promise<Workspace[]>;

  async retrieve(keys: Key | Key[]): Promise<Workspace | Workspace[]> {
    const isMany = Array.isArray(keys);
    const res = await this.retriever.retrieve(keys);
    return isMany ? res : res[0];
  }

  async page(offset: number, limit: number): Promise<Workspace[]> {
    return await this.retriever.page(offset, limit);
  }

  async create(workspace: NewWorkspace): Promise<Workspace>;

  async create(
    workspaces: NewWorkspace | NewWorkspace[],
  ): Promise<Workspace | Workspace[]> {
    const isMany = Array.isArray(workspaces);
    const res = await this.writer.create(workspaces);
    return isMany ? res : res[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.writer.rename(key, name);
  }

  async setLayout(key: Key, layout: UnknownRecord): Promise<void> {
    await this.writer.setLayout(key, layout);
  }

  async delete(...keys: Key[]): Promise<void> {
    await this.writer.delete(keys);
  }
}
