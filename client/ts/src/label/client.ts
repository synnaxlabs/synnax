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

import { type Key, type Label } from "@/label/payload";
import { Retriever } from "@/label/retriever";
import { Writer, type NewLabelPayload } from "@/label/writer";
import { type ontology } from "@/ontology";

export class Client implements AsyncTermSearcher<string, Key, Label> {
  private readonly retriever: Retriever;
  private readonly writer: Writer;

  constructor(client: UnaryClient) {
    this.writer = new Writer(client);
    this.retriever = new Retriever(client);
  }

  async search(term: string): Promise<Label[]> {
    return await this.retriever.search(term);
  }

  async retrieve(key: Key): Promise<Label>;

  async retrieve(keys: Key[]): Promise<Label[]>;

  async retrieve(keys: Key | Key[]): Promise<Label | Label[]> {
    const isMany = Array.isArray(keys);
    const res = await this.retriever.retrieve(keys);
    return isMany ? res : res[0];
  }

  async retrieveFor(id: ontology.ID): Promise<Label[]> {
    return await this.retriever.retrieveFor(id);
  }

  async label(id: ontology.ID, labels: Key[]): Promise<void> {
    await this.writer.set(id, labels);
  }

  async removeLabels(id: ontology.ID, labels: Key[]): Promise<void> {
    await this.writer.remove(id, labels);
  }

  async page(offset: number, limit: number): Promise<Label[]> {
    return await this.retriever.page(offset, limit);
  }

  async create(label: NewLabelPayload): Promise<Label>;

  async create(labels: NewLabelPayload[]): Promise<Label[]>;

  async create(labels: NewLabelPayload | NewLabelPayload[]): Promise<Label | Label[]> {
    const isMany = Array.isArray(labels);
    const res = await this.writer.create(labels);
    return isMany ? res : res[0];
  }

  async delete(key: Key): Promise<void>;

  async delete(keys: Key[]): Promise<void>;

  async delete(keys: Key | Key[]): Promise<void> {
    await this.writer.delete(keys);
  }
}
