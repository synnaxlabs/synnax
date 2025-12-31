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
import { z } from "zod";

import { checkForMultipleOrNoResults } from "@/util/retrieve";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  type View,
  viewZ,
} from "@/view/types.gen";

export const SET_CHANNEL_NAME = "sy_view_set";
export const DELETE_CHANNEL_NAME = "sy_view_delete";

const createReqZ = z.object({ views: newZ.array() });
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

const singleRetrieveArgsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const retrieveArgsZ = z.union([singleRetrieveArgsZ, retrieveRequestZ]);

export interface RetrieveSingleParams extends z.input<typeof singleRetrieveArgsZ> {}
export interface RetrieveMultipleParams extends z.input<typeof retrieveRequestZ> {}

const retrieveResponseZ = z.object({ views: array.nullishToEmpty(viewZ) });

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(args: RetrieveSingleParams): Promise<View>;
  async retrieve(args: RetrieveMultipleParams): Promise<View[]>;
  async retrieve(
    args: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<View | View[]> {
    const isSingle = "key" in args;
    const res = await sendRequired(
      this.client,
      "/view/retrieve",
      args,
      retrieveArgsZ,
      retrieveResponseZ,
    );
    checkForMultipleOrNoResults("View", args, res.views, isSingle);
    return isSingle ? res.views[0] : res.views;
  }

  async create(view: New): Promise<View>;
  async create(views: New[]): Promise<View[]>;
  async create(views: New | New[]): Promise<View | View[]> {
    const isMany = Array.isArray(views);
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      "/view/create",
      { views: array.toArray(views) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.views : res.views[0];
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof emptyResZ>(
      this.client,
      "/view/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}

