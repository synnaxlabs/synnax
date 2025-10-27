// Copyright 2025 Synnax Labs, Inc.
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

const RETRIEVE_ENDPOINT = "/arc/retrieve";
const CREATE_ENDPOINT = "/arc/create";
const DELETE_ENDPOINT = "/arc/delete";
const LSP_ENDPOINT = "/arc/lsp";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  searchTerm: z.string().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
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
      CREATE_ENDPOINT,
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
      RETRIEVE_ENDPOINT,
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
      DELETE_ENDPOINT,
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }

  /**
   * Opens a new LSP stream to the server for Language Server Protocol communication.
   * This allows editor integrations to communicate with the Arc LSP server using
   * JSON-RPC messages over a WebSocket transport.
   *
   * @returns A bidirectional stream for sending and receiving JSON-RPC messages
   */
  async openLSP(): Promise<Stream<typeof lspMessageZ, typeof lspMessageZ>> {
    return await this.streamClient.stream(LSP_ENDPOINT, lspMessageZ, lspMessageZ);
  }
}

export const ontologyID = (key: Key): ontology.ID => ({ type: ONTOLOGY_TYPE, key });
