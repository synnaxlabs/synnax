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
  type Schematic,
  schematicRemoteZ,
  schematicZ,
} from "@/workspace/schematic/payload";

export const newSchematicZ = schematicZ
  .partial({ key: true, snapshot: true })
  .transform((p) => ({
    ...p,
    data: JSON.stringify(p.data),
  }));

export type NewSchematic = z.input<typeof newSchematicZ>;

const createReqZ = z.object({
  workspace: workspaceKeyZ,
  schematics: newSchematicZ.array(),
});

const createResZ = z.object({
  schematics: schematicRemoteZ.array(),
});

const deleteReqZ = z.object({
  keys: keyZ.array(),
});

const deleteResZ = z.object({});

const renameReqZ = z.object({
  key: keyZ,
  name: z.string(),
});

const renameResZ = z.object({});

const setDataReqZ = z.object({
  key: keyZ,
  data: z.string(),
});

const setDataResZ = z.object({});

const copyReqZ = z.object({
  key: keyZ,
  name: z.string(),
  snapshot: z.boolean(),
});

const copyResZ = z.object({
  schematic: schematicRemoteZ,
});

const CREATE_ENDPOINT = "/workspace/schematic/create";
const DELETE_ENDPOINT = "/workspace/schematic/delete";
const RENAME_ENDPOINT = "/workspace/schematic/rename";
const SET_DATA_ENDPOINT = "/workspace/schematic/set-data";
const COPY_ENDPOINT = "/workspace/schematic/copy";

export class Writer {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: string, schematic: NewSchematic): Promise<Schematic> {
    const schematic_ = { ...schematic, data: JSON.stringify(schematic.data) };
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { workspace, schematics: [schematic_] },
      createReqZ,
      createResZ,
    );

    return res.schematics[0];
  }

  async copy(key: Key, name: string, snapshot: boolean): Promise<Schematic> {
    const res = await sendRequired<typeof copyReqZ, typeof copyResZ>(
      this.client,
      COPY_ENDPOINT,
      { key, name, snapshot },
      copyReqZ,
      copyResZ,
    );
    return res.schematic;
  }

  async delete(schematics: Params): Promise<void> {
    const normalized = toArray(schematics);
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: normalized },
      deleteReqZ,
      deleteResZ,
    );
  }

  async rename(schematic: Key, name: string): Promise<void> {
    await sendRequired<typeof renameReqZ, typeof renameResZ>(
      this.client,
      RENAME_ENDPOINT,
      { key: schematic, name },
      renameReqZ,
      renameResZ,
    );
  }

  async setData(schematic: Key, data: UnknownRecord): Promise<void> {
    await sendRequired<typeof setDataReqZ, typeof setDataResZ>(
      this.client,
      SET_DATA_ENDPOINT,
      { key: schematic, data: JSON.stringify(data) },
      setDataReqZ,
      renameResZ,
    );
  }
}
