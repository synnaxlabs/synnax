// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  sendRequired,
  type Stream,
  type StreamClient,
  type UnaryClient,
} from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { type Arc, arcZ, keyZ, type New, newZ, type Params } from "@/arc/payload";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_arc_set";
export const DELETE_CHANNEL_NAME = "sy_arc_delete";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  searchTerm: z.string().optional(),
  limit: z.int().optional(),
  offset: z.int().optional(),
  includeStatus: z.boolean().optional(),
});
const createReqZ = z.object({ arcs: newZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveResZ = z.object({ arcs: array.nullableZ(arcZ) });
const createResZ = z.object({ arcs: arcZ.array() });
const emptyResZ = z.object({});

export const lspMessageZ = z.object({ content: z.string() });
export type LSPMessage = z.infer<typeof lspMessageZ>;

export type RetrieveRequest = z.input<typeof retrieveReqZ>;

const keyRetrieveRequestZ = z
  .object({
    key: keyZ,
    includeStatus: z.boolean().optional(),
  })
  .transform(({ key, includeStatus }) => ({ keys: [key], includeStatus }));

const nameRetrieveRequestZ = z
  .object({
    name: z.string(),
    includeStatus: z.boolean().optional(),
  })
  .transform(({ name, includeStatus }) => ({ names: [name], includeStatus }));

export const singleRetrieveArgsZ = z.union([keyRetrieveRequestZ, nameRetrieveRequestZ]);

export type SingleRetrieveArgs = z.input<typeof singleRetrieveArgsZ>;

const retrieveArgsZ = z.union([singleRetrieveArgsZ, retrieveReqZ]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

export class Client {
  private readonly client: UnaryClient;
  private readonly streamClient: StreamClient;

  constructor(client: UnaryClient, streamClient: StreamClient) {
    this.client = client;
    this.streamClient = streamClient;
  }

  async create(arc: New): Promise<Arc>;
  async create(arcs: New[]): Promise<Arc[]>;
  async create(arcs: New | New[]): Promise<Arc | Arc[]> {
    const isMany = Array.isArray(arcs);
    const res = await sendRequired(
      this.client,
      "/arc/create",
      { arcs: array.toArray(arcs) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.arcs : res.arcs[0];
  }

  async retrieve(args: SingleRetrieveArgs): Promise<Arc>;
  async retrieve(args: RetrieveArgs): Promise<Arc[]>;
  async retrieve(args: RetrieveArgs): Promise<Arc | Arc[]> {
    const isSingle = "key" in args || "name" in args;
    const res = await sendRequired(
      this.client,
      "/arc/retrieve",
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Arc", args, res.arcs, isSingle);
    return isSingle ? res.arcs[0] : res.arcs;
  }

  async delete(keys: Params): Promise<void> {
    await sendRequired(
      this.client,
      "/arc/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }

  async openLSP(): Promise<Stream<typeof lspMessageZ, typeof lspMessageZ>> {
    return await this.streamClient.stream("/arc/lsp", lspMessageZ, lspMessageZ);
  }
}
