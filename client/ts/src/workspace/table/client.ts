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
  type Table,
} from "@/workspace/table/payload";

const renameReqZ = z.object({ key: keyZ, name: z.string() });

const setDataReqZ = z.object({ key: keyZ, data: z.string() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveReqZ = z.object({ keys: keyZ.array() });
const singleRetrieveArgsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

export const retrieveArgsZ = z.union([singleRetrieveArgsZ, retrieveReqZ]);
export type RetrieveArgs = z.input<typeof retrieveArgsZ>;
export type RetrieveSingleParams = z.input<typeof singleRetrieveArgsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveReqZ>;

const retrieveResZ = z.object({ tables: array.nullableZ(remoteZ) });

const createReqZ = z.object({ workspace: workspaceKeyZ, tables: newZ.array() });
const createResZ = z.object({ tables: remoteZ.array() });

const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: WorkspaceKey, table: New): Promise<Table>;
  async create(workspace: WorkspaceKey, tables: New[]): Promise<Table[]>;
  async create(workspace: WorkspaceKey, tables: New | New[]): Promise<Table | Table[]> {
    const isMany = Array.isArray(tables);
    const res = await sendRequired(
      this.client,
      "/workspace/table/create",
      { workspace, tables: array.toArray(tables) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.tables : res.tables[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await sendRequired(
      this.client,
      "/workspace/table/rename",
      { key, name },
      renameReqZ,
      emptyResZ,
    );
  }

  async setData(key: Key, data: record.Unknown): Promise<void> {
    await sendRequired(
      this.client,
      "/workspace/table/set-data",
      { key, data: JSON.stringify(data) },
      setDataReqZ,
      emptyResZ,
    );
  }

  async retrieve(args: RetrieveSingleParams): Promise<Table>;
  async retrieve(args: RetrieveMultipleParams): Promise<Table[]>;
  async retrieve(
    args: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<Table | Table[]> {
    const isSingle = singleRetrieveArgsZ.safeParse(args).success;
    const res = await sendRequired(
      this.client,
      "/workspace/table/retrieve",
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Table", args, res.tables, isSingle);
    return isSingle ? res.tables[0] : res.tables;
  }

  async delete(keys: Params): Promise<void> {
    await sendRequired(
      this.client,
      "/workspace/table/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}

export const ontologyID = (key: Key): ontology.ID => ({ type: "table", key });
