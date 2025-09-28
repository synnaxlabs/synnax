// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array, type record } from "@synnaxlabs/x";
import { z } from "zod";

import { type ontology } from "@/ontology";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { type Key as WorkspaceKey, keyZ as workspaceKeyZ } from "@/workspace/payload";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  type Params,
  remoteZ,
  type Schematic,
  schematicZ,
} from "@/workspace/schematic/payload";
import { symbol } from "@/workspace/schematic/symbol";

const RETRIEVE_ENDPOINT = "/workspace/schematic/retrieve";
const CREATE_ENDPOINT = "/workspace/schematic/create";
const RENAME_ENDPOINT = "/workspace/schematic/rename";
const SET_DATA_ENDPOINT = "/workspace/schematic/set-data";
const DELETE_ENDPOINT = "/workspace/schematic/delete";
const COPY_ENDPOINT = "/workspace/schematic/copy";

const renameReqZ = z.object({ key: keyZ, name: z.string() });

const setDataReqZ = z.object({ key: keyZ, data: z.string() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const copyReqZ = z.object({
  key: keyZ,
  name: z.string(),
  snapshot: z.boolean(),
});

const retrieveReqZ = z.object({ keys: keyZ.array() });
const singleRetrieveArgsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

export const retrieveArgsZ = z.union([singleRetrieveArgsZ, retrieveReqZ]);
export type RetrieveArgs = z.input<typeof retrieveArgsZ>;
export type RetrieveSingleParams = z.input<typeof singleRetrieveArgsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveReqZ>;
export type CopyArgs = z.input<typeof copyReqZ>;

const retrieveResZ = z.object({ schematics: array.nullableZ(remoteZ) });

const createReqZ = z.object({
  workspace: workspaceKeyZ,
  schematics: newZ.array(),
});
const createResZ = z.object({ schematics: remoteZ.array() });

const copyResZ = z.object({ schematic: schematicZ });
const emptyResZ = z.object({});

export class Client {
  readonly symbols: symbol.Client;
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
    this.symbols = new symbol.Client(client);
  }

  async create(workspace: WorkspaceKey, schematic: New): Promise<Schematic>;
  async create(workspace: WorkspaceKey, schematics: New[]): Promise<Schematic[]>;
  async create(
    workspace: WorkspaceKey,
    schematics: New | New[],
  ): Promise<Schematic | Schematic[]> {
    const isMany = Array.isArray(schematics);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { workspace, schematics: array.toArray(schematics) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.schematics : res.schematics[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await sendRequired(
      this.client,
      RENAME_ENDPOINT,
      { key, name },
      renameReqZ,
      emptyResZ,
    );
  }

  async setData(key: Key, data: record.Unknown): Promise<void> {
    await sendRequired(
      this.client,
      SET_DATA_ENDPOINT,
      { key, data: JSON.stringify(data) },
      setDataReqZ,
      emptyResZ,
    );
  }

  async retrieve(args: RetrieveSingleParams): Promise<Schematic>;
  async retrieve(args: RetrieveMultipleParams): Promise<Schematic[]>;
  async retrieve(
    args: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<Schematic | Schematic[]> {
    const isSingle = singleRetrieveArgsZ.safeParse(args).success;
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Schematic", args, res.schematics, isSingle);
    return isSingle ? res.schematics[0] : res.schematics;
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

  async copy(args: CopyArgs): Promise<Schematic> {
    const res = await sendRequired(
      this.client,
      COPY_ENDPOINT,
      args,
      copyReqZ,
      copyResZ,
    );
    return res.schematic;
  }
}

export const ontologyID = (key: Key): ontology.ID => ({
  type: "schematic",
  key,
});
