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
import z from "zod";

import { type Key, keyZ, type Label, labelZ } from "@/label/payload";
import { ontology } from "@/ontology";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_label_set";
export const DELETE_CHANNEL_NAME = "sy_label_delete";

export const newZ = labelZ.extend({ key: keyZ.optional() });
export interface New extends z.infer<typeof newZ> {}

const createReqZ = z.object({ labels: newZ.array() });
const createResZ = z.object({ labels: labelZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const setReqZ = z.object({
  id: ontology.idZ,
  labels: keyZ.array(),
  replace: z.boolean().optional(),
});

interface SetReq extends z.infer<typeof setReqZ> {}
export interface SetOptions extends Pick<SetReq, "replace"> {}

const removeReqZ = setReqZ.omit({ replace: true });
const emptyResZ = z.object({});

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  for: ontology.idZ.optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
});

const singleRetrieveArgsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const retrieveArgsZ = z.union([singleRetrieveArgsZ, retrieveRequestZ]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;
export type RetrieveSingleParams = z.input<typeof singleRetrieveArgsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;

const retrieveResponseZ = z.object({ labels: array.nullableZ(labelZ) });

export class Client {
  readonly type: string = "label";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(args: RetrieveSingleParams): Promise<Label>;
  async retrieve(args: RetrieveMultipleParams): Promise<Label[]>;
  async retrieve(args: RetrieveArgs): Promise<Label | Label[]> {
    const isSingle = "key" in args;
    const res = await sendRequired(
      this.client,
      "/label/retrieve",
      args,
      retrieveArgsZ,
      retrieveResponseZ,
    );
    checkForMultipleOrNoResults("Label", args, res.labels, isSingle);
    return isSingle ? res.labels[0] : res.labels;
  }

  async label(id: ontology.ID, labels: Key[], opts: SetOptions = {}): Promise<void> {
    await sendRequired<typeof setReqZ, typeof emptyResZ>(
      this.client,
      "/label/set",
      { id, labels, replace: opts.replace },
      setReqZ,
      emptyResZ,
    );
  }

  async remove(id: ontology.ID, labels: Key[]): Promise<void> {
    await sendRequired<typeof removeReqZ, typeof emptyResZ>(
      this.client,
      "/label/remove",
      { id, labels },
      removeReqZ,
      emptyResZ,
    );
  }

  async create(label: New): Promise<Label>;
  async create(labels: New[]): Promise<Label[]>;
  async create(labels: New | New[]): Promise<Label | Label[]> {
    const isMany = Array.isArray(labels);
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      "/label/create",
      { labels: array.toArray(labels) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.labels : res.labels[0];
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof emptyResZ>(
      this.client,
      "/label/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}

export const ontologyID = (key: Key): ontology.ID => ({ type: "label", key });
