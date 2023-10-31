// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { type AsyncTermSearcher } from "@synnaxlabs/x";

import { type Client as FrameClient } from "@/framer/client";
import { ChangeTracker } from "@/ontology/cdc";
import { group } from "@/ontology/group";
import { type ID, type Resource } from "@/ontology/payload";
import { Retriever } from "@/ontology/retriever";
import { Writer } from "@/ontology/writer";

import { QueryError } from "..";

/** The core client class for executing queries against a Synnax cluster ontology */
export class Client implements AsyncTermSearcher<string, string, Resource> {
  groups: group.Client;
  retriever: Retriever;
  private readonly writer: Writer;
  private readonly framer: FrameClient;

  constructor(unary: UnaryClient, framer: FrameClient) {
    this.retriever = new Retriever(unary);
    this.writer = new Writer(unary);
    this.groups = new group.Client(unary);
    this.framer = framer;
  }

  async search(term: string): Promise<Resource[]> {
    return await this.retriever.search(term);
  }

  async retrieve(
    id: ID | string,
    includeSchema?: boolean,
    includeFieldData?: boolean,
  ): Promise<Resource>;

  async retrieve(
    ids: ID[] | string[],
    includeSchema?: boolean,
    includeFieldData?: boolean,
  ): Promise<Resource[]>;

  async retrieve(
    ids: ID | ID[] | string | string[],
    includeSchema?: boolean,
    includeFieldData?: boolean,
  ): Promise<Resource | Resource[]> {
    const resources = await this.retriever.retrieve(
      ids,
      includeSchema,
      includeFieldData,
    );
    if (Array.isArray(ids)) return resources;
    if (resources.length === 0)
      throw new QueryError(`No resource found with ID ${ids.toString()}`);
    return resources[0];
  }

  async page(offset: number, limit: number): Promise<Resource[]> {
    return [];
  }

  async retrieveChildren(
    IDs: ID | ID[],
    includeSchema?: boolean,
    includeFieldData?: boolean,
  ): Promise<Resource[]> {
    return await this.retriever.retrieveChildren(IDs, includeSchema, includeFieldData);
  }

  async retrieveParents(
    IDs: ID | ID[],
    includeSchema?: boolean,
    includeFieldData?: boolean,
  ): Promise<Resource[]> {
    return await this.retriever.retrieveParents(IDs, includeSchema, includeFieldData);
  }

  async addChildren(id: ID, ...children: ID[]): Promise<void> {
    return await this.writer.addChildren(id, ...children);
  }

  async removeChildren(id: ID, ...children: ID[]): Promise<void> {
    return await this.writer.removeChildren(id, ...children);
  }

  async moveChildren(from: ID, to: ID, ...children: ID[]): Promise<void> {
    return await this.writer.moveChildren(from, to, ...children);
  }

  async openChangeTracker(): Promise<ChangeTracker> {
    return await ChangeTracker.open(this.framer, this.retriever);
  }
}
