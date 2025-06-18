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

const CREATE_ENDPOINT = "/annotation/create";
const DELETE_ENDPOINT = "/annotation/delete";
const RETRIEVE_ENDPOINT = "/annotation/retrieve";

const createReqZ = z.object({ annotations: z.array(newZ) });
const createResZ = z.object({ annotations: z.array(annotationZ) });
const deleteReqZ = z.object({ keys: z.array(keyZ) });
const retrieveReqZ = z.object({ keys: z.array(keyZ) });
const retrieveResZ = z.object({ annotations: z.array(annotationZ) });
const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(annotation: New): Promise<Annotation>;
  async create(annotations: New[]): Promise<Annotation[]>;
  async create(annotations: New | New[]): Promise<Annotation | Annotation[]> {
    const isMany = Array.isArray(annotations);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { annotations: array.toArray(annotations) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.annotations : res.annotations[0];
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
  async delete(keys: Params): Promise<void> {
    await sendRequired(
      this.client,
      DELETE_ENDPOINT,
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }

  async retrieve(key: Key): Promise<Annotation>;
  async retrieve(keys: Key[]): Promise<Annotation[]>;
  async retrieve(keys: Params): Promise<Annotation | Annotation[]> {
    const isMany = Array.isArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: array.toArray(keys) },
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.annotations : res.annotations[0];
  }
}
