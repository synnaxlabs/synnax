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
import z from "zod";

import { type Key, keyZ, type Label, labelZ, ONTOLOGY_TYPE } from "@/label/payload";
import { type New, type SetOptions, Writer } from "@/label/writer";
import { ontology } from "@/ontology";
import { nullableArrayZ } from "@/util/zod";

export const SET_CHANNEL_NAME = "sy_label_set";
export const DELETE_CHANNEL_NAME = "sy_label_delete";

const RETRIEVE_ENDPOINT = "/label/retrieve";

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  for: ontology.idZ.optional(),
  search: z.string().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

export interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

const retrieveResponseZ = z.object({ labels: nullableArrayZ(labelZ) });

export class Client {
  readonly type: string = "label";
  private readonly client: UnaryClient;
  private readonly writer: Writer;

  constructor(client: UnaryClient) {
    this.client = client;
    this.writer = new Writer(client);
  }

  async retrieve(req: RetrieveRequest): Promise<Label[]>;
  async retrieve(key: Key): Promise<Label>;
  async retrieve(keys: Key[]): Promise<Label[]>;
  async retrieve(keys: Key | RetrieveRequest | Key[]): Promise<Label | Label[]> {
    const isSingle = typeof keys === "string";
    let req: RetrieveRequest;
    if (typeof keys === "object" && !Array.isArray(keys)) req = keys;
    else req = { keys: array.toArray(keys) };
    const [res, err] = await this.client.send<
      typeof retrieveRequestZ,
      typeof retrieveResponseZ
    >(RETRIEVE_ENDPOINT, req, retrieveRequestZ, retrieveResponseZ);
    if (err != null) throw err;
    if (isSingle) return res.labels[0];
    return res.labels;
  }

  async retrieveFor(id: ontology.ID): Promise<Label[]> {
    return await this.retrieve({ for: id });
  }

  async label(id: ontology.ID, labels: Key[], opts: SetOptions = {}): Promise<void> {
    await this.writer.set(id, labels, opts);
  }

  async removeLabels(id: ontology.ID, labels: Key[]): Promise<void> {
    await this.writer.remove(id, labels);
  }

  async create(label: New): Promise<Label>;
  async create(labels: New[]): Promise<Label[]>;
  async create(labels: New | New[]): Promise<Label | Label[]> {
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

export const ontologyID = (key: Key): ontology.ID => ({ type: ONTOLOGY_TYPE, key });
