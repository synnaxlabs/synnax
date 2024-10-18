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

import {
  type Key,
  keyZ,
  type Log,
  logRemoteZ,
  logZ,
  type Params,
} from "@/workspace/log/payload";
import { keyZ as workspaceKeyZ } from "@/workspace/payload";

export const newLogZ = logZ.partial({ key: true }).transform((p) => ({
  ...p,
  data: JSON.stringify(p.data),
}));

export type NewLog = z.input<typeof newLogZ>;

const createReqZ = z.object({
  workspace: workspaceKeyZ,
  logs: newLogZ.array(),
});

const createResZ = z.object({
  logs: logRemoteZ.array(),
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

const CREATE_ENDPOINT = "/workspace/log/create";
const DELETE_ENDPOINT = "/workspace/log/delete";
const RENAME_ENDPOINT = "/workspace/log/rename";
const SET_DATA_ENDPOINT = "/workspace/log/set-data";

export class Writer {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: string, log: NewLog): Promise<Log> {
    const log_ = { ...log, data: JSON.stringify(log.data) };
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { workspace, logs: [log_] },
      createReqZ,
      createResZ,
    );

    return res.logs[0];
  }

  async delete(logs: Params): Promise<void> {
    const normalized = toArray(logs);
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: normalized },
      deleteReqZ,
      deleteResZ,
    );
  }

  async rename(log: Key, name: string): Promise<void> {
    await sendRequired<typeof renameReqZ, typeof renameResZ>(
      this.client,
      RENAME_ENDPOINT,
      { key: log, name },
      renameReqZ,
      renameResZ,
    );
  }

  async setData(log: Key, data: UnknownRecord): Promise<void> {
    await sendRequired<typeof setDataReqZ, typeof setDataResZ>(
      this.client,
      SET_DATA_ENDPOINT,
      { key: log, data: JSON.stringify(data) },
      setDataReqZ,
      setDataResZ,
    );
  }
}
