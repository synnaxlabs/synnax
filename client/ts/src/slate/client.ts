// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { toArray } from "@synnaxlabs/x";
import { z } from "zod";

import { ontology } from "@/ontology";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  ONTOLOGY_TYPE,
  type Params,
  type Slate,
  slateZ,
} from "@/slate/payload";
import { nullableArrayZ } from "@/util/zod";

const RETRIEVE_ENDPOINT = "/workspace/slate/retrieve";
const CREATE_ENDPOINT = "/workspace/slate/create";
const DELETE_ENDPOINT = "/workspace/slate/delete";

const retrieveReqZ = z.object({ keys: keyZ.array() });
const createReqZ = z.object({ slates: newZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveResZ = z.object({ slates: nullableArrayZ(slateZ) });
const createResZ = z.object({ slates: slateZ.array() });
const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(slate: New): Promise<Slate>;
  async create(slates: New[]): Promise<Slate[]>;
  async create(slates: New | New[]): Promise<Slate | Slate[]> {
    const isMany = Array.isArray(slates);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { slates: toArray(slates) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.slates : res.slates[0];
  }

  async retrieve(key: Key): Promise<Slate>;
  async retrieve(keys: Key[]): Promise<Slate[]>;
  async retrieve(keys: Params): Promise<Slate | Slate[]> {
    const isMany = Array.isArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: toArray(keys) },
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.slates : res.slates[0];
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
  async delete(keys: Params): Promise<void> {
    await sendRequired(
      this.client,
      DELETE_ENDPOINT,
      { keys: toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });
