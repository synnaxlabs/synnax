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
  type Arc,
  arcZ,
  type Key,
  keyZ,
  type New,
  newZ,
  ONTOLOGY_TYPE,
  type Params,
} from "@/arc/payload";
import { type ontology } from "@/ontology";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";

const RETRIEVE_ENDPOINT = "/arc/retrieve";
const CREATE_ENDPOINT = "/arc/create";
const DELETE_ENDPOINT = "/arc/delete";

const retrieveReqZ = z.object({ keys: keyZ.array() });
const createReqZ = z.object({ arcs: newZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveResZ = z.object({ arcs: nullableArrayZ(arcZ) });
const createResZ = z.object({ arcs: arcZ.array() });
const emptyResZ = z.object({});

export type RetrieveRequest = z.input<typeof retrieveReqZ>;

const keyRetrieveRequestZ = z
  .object({
    key: keyZ,
  })
  .transform(({ key }) => ({ keys: [key] }));

export type KeyRetrieveRequest = z.input<typeof keyRetrieveRequestZ>;

const retrieveArgsZ = z.union([keyRetrieveRequestZ, retrieveReqZ]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(arc: New): Promise<Arc>;
  async create(arcs: New[]): Promise<Arc[]>;
  async create(arcs: New | New[]): Promise<Arc | Arc[]> {
    const isMany = Array.isArray(arcs);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { arcs: array.toArray(arcs) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.arcs : res.arcs[0];
  }

  async retrieve(args: KeyRetrieveRequest): Promise<Arc>;
  async retrieve(args: RetrieveArgs): Promise<Arc[]>;
  async retrieve(args: RetrieveArgs): Promise<Arc | Arc[]> {
    const isSingle = "key" in args;
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Arc", args, res.arcs, isSingle);
    return isSingle ? res.arcs[0] : res.arcs;
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
}

export const ontologyID = (key: Key): ontology.ID => ({ type: ONTOLOGY_TYPE, key });
