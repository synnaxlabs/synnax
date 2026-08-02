// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { type Key, keyZ, type New, type View, viewZ } from "@/view/types.gen";

export const SET_CHANNEL_NAME = "sy_view_set";
export const DELETE_CHANNEL_NAME = "sy_view_delete";

const createReqZ = z.object({ views: viewZ.array() });
const createResZ = z.object({ views: viewZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  types: z.string().array().optional(),
  searchTerm: z.string().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

const singleRetrieveParamsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveRequestZ]);

export interface RetrieveSingleParams extends z.input<typeof singleRetrieveParamsZ> {}
export interface RetrieveMultipleParams extends z.input<typeof retrieveRequestZ> {}

const retrieveResponseZ = z.object({ views: viewZ.array().default(() => []) });

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(key: Key): Promise<View>;
  async retrieve(params: RetrieveSingleParams): Promise<View>;
  async retrieve(params: RetrieveMultipleParams): Promise<View[]>;
  async retrieve(
    params: Key | RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<View | View[]> {
    if (typeof params !== "object") params = { key: params };
    const isSingle = "key" in params;
    const res = await this.client.send(
      "/view/retrieve",
      params,
      retrieveParamsZ,
      retrieveResponseZ,
    );
    checkForMultipleOrNoResults("View", params, res.views, isSingle);
    return isSingle ? res.views[0] : res.views;
  }

  async create(view: New): Promise<View>;
  async create(views: New[]): Promise<View[]>;
  async create(views: New | New[]): Promise<View | View[]> {
    const isMany = Array.isArray(views);
    const res = await this.client.send(
      "/view/create",
      { views: array.toArray(views) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.views : res.views[0];
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await this.client.send(
      "/view/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}
