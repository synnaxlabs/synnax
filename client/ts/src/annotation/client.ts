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
import { z } from "zod/v4";

import {
  type Annotation,
  annotationZ,
  type Key,
  keyZ,
  type New,
  newZ,
  type Params,
} from "@/annotation/payload";
import { ontology } from "@/ontology";

const CREATE_ENDPOINT = "/annotation/create";
const DELETE_ENDPOINT = "/annotation/delete";
const RETRIEVE_ENDPOINT = "/annotation/retrieve";

export const SET_CHANNEL_NAME = "sy_annotation_set";
export const DELETE_CHANNEL_NAME = "sy_annotation_delete";

const createReqZ = z.object({
  parent: ontology.idZ,
  annotations: z.array(newZ),
});
const createResZ = z.object({ annotations: z.array(annotationZ) });
const deleteReqZ = z.object({ keys: z.array(keyZ) });
const retrieveResZ = z.object({ annotations: z.array(annotationZ) });
const emptyResZ = z.object({});

const retrieveReqZ = z.object({
  parent: ontology.idZ.optional(),
  keys: z.array(keyZ).optional(),
  term: z.string().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  includeCreator: z.boolean().optional(),
});
export type RetrieveRequest = z.infer<typeof retrieveReqZ>;

const keyRetrieveReqZ = z
  .object({ key: keyZ, includeCreator: z.boolean().optional() })
  .transform(({ key, includeCreator }) => ({ keys: [key], includeCreator }));

type KeyRetrieveRequest = z.input<typeof keyRetrieveReqZ>;

const retrieveArgsZ = z.union([keyRetrieveReqZ, retrieveReqZ]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(annotation: New, parent: ontology.ID): Promise<Annotation>;
  async create(annotations: New[], parent: ontology.ID): Promise<Annotation[]>;
  async create(
    annotations: New | New[],
    parent: ontology.ID,
  ): Promise<Annotation | Annotation[]> {
    const isMany = Array.isArray(annotations);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { parent, annotations: array.toArray(annotations) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.annotations : res.annotations[0];
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired(
      this.client,
      DELETE_ENDPOINT,
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }

  async retrieve(args: KeyRetrieveRequest): Promise<Annotation>;
  async retrieve(args: RetrieveRequest): Promise<Annotation[]>;
  async retrieve(args: RetrieveArgs): Promise<Annotation | Annotation[]> {
    const isSingle = "key" in args;
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    return isSingle ? res.annotations[0] : res.annotations;
  }
}
