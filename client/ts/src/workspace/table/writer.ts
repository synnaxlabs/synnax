// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { type UnknownRecord } from "@synnaxlabs/x/record";
import { toArray } from "@synnaxlabs/x/toArray";
import { z } from "zod";

import { keyZ as workspaceKeyZ } from "@/workspace/payload";
import {
  type Key,
  keyZ,
  type Params,
  type Table,
  tableRemoteZ,
  tableZ,
} from "@/workspace/table/payload";

export const newTableZ = tableZ
  .partial({ key: true })
  .transform((p) => ({ ...p, data: JSON.stringify(p.data) }));

export type NewTable = z.input<typeof newTableZ>;

const createReqZ = z.object({ workspace: workspaceKeyZ, tables: newTableZ.array() });

const createResZ = z.object({ tables: tableRemoteZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });

const deleteResZ = z.object({});

const renameReqZ = z.object({ key: keyZ, name: z.string() });

const renameResZ = z.object({});

const setDataReqZ = z.object({ key: keyZ, data: z.string() });

const setDataResZ = z.object({});

const CREATE_ENDPOINT = "/workspace/table/create";
const DELETE_ENDPOINT = "/workspace/table/delete";
const RENAME_ENDPOINT = "/workspace/table/rename";
const SET_DATA_ENDPOINT = "/workspace/table/set-data";

export class Writer {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: string, table: NewTable): Promise<Table> {
    const table_ = { ...table, data: JSON.stringify(table.data) };
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { workspace, tables: [table_] },
      createReqZ,
      createResZ,
    );

    return res.tables[0];
  }

  async delete(tables: Params): Promise<void> {
    const normalized = toArray(tables);
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: normalized },
      deleteReqZ,
      deleteResZ,
    );
  }

  async rename(table: Key, name: string): Promise<void> {
    await sendRequired<typeof renameReqZ, typeof renameResZ>(
      this.client,
      RENAME_ENDPOINT,
      { key: table, name },
      renameReqZ,
      renameResZ,
    );
  }

  async setData(table: Key, data: UnknownRecord): Promise<void> {
    await sendRequired<typeof setDataReqZ, typeof setDataResZ>(
      this.client,
      SET_DATA_ENDPOINT,
      { key: table, data: JSON.stringify(data) },
      setDataReqZ,
      setDataResZ,
    );
  }
}
