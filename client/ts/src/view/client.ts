// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x/array";
import z from "zod";

import { ontology } from "@/ontology";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { type Key, keyZ, type View, viewZ } from "@/view/payload";

export const newZ = viewZ.extend({ key: keyZ.optional() });
export interface New extends z.infer<typeof newZ> {}

const createReqZ = z.object({
  parent: ontology.idZ.optional(),
  views: newZ.array(),
});
const createResZ = z.object({ views: viewZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const CREATE_ENDPOINT = "/view/create";
const DELETE_ENDPOINT = "/view/delete";
const RETRIEVE_ENDPOINT = "/view/retrieve";

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

const singleRetrieveArgsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const retrieveArgsZ = z.union([singleRetrieveArgsZ, retrieveRequestZ]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;
export type RetrieveSingleParams = z.input<typeof singleRetrieveArgsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;

const retrieveResponseZ = z.object({ views: array.nullableZ(viewZ) });

export interface CreateOptions {
  parent?: ontology.ID;
}

export class Client {
  readonly type: string = "view";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(args: RetrieveSingleParams): Promise<View>;
  async retrieve(args: RetrieveMultipleParams): Promise<View[]>;
  async retrieve(args: RetrieveArgs): Promise<View | View[]> {
    const isSingle = "key" in args;
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      args,
      retrieveArgsZ,
      retrieveResponseZ,
    );
    checkForMultipleOrNoResults("View", args, res.views, isSingle);
    return isSingle ? res.views[0] : res.views;
  }

  async create(view: New, opts?: CreateOptions): Promise<View>;
  async create(views: New[], opts?: CreateOptions): Promise<View[]>;
  async create(views: New | New[], opts: CreateOptions = {}): Promise<View | View[]> {
    const isMany = Array.isArray(views);
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { parent: opts.parent, views: array.toArray(views) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.views : res.views[0];
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof emptyResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}

export const ontologyID = (key: Key): ontology.ID => ({ type: "view", key });