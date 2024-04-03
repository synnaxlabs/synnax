// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { toArray, type UnknownRecord } from "@synnaxlabs/x";
import { z } from "zod";

import { keyZ as workspaceKeyZ } from "@/workspace/payload";
import {
  pidZ,
  type PID,
  type Params,
  keyZ,
  type Key,
  pidRemoteZ,
} from "@/workspace/pid/payload";

export const newPIDZ = pidZ.partial({ key: true, snapshot: true }).transform((p) => ({
  ...p,
  data: JSON.stringify(p.data),
}));

export type NewPID = z.input<typeof newPIDZ>;

const createReqZ = z.object({
  workspace: workspaceKeyZ,
  pids: newPIDZ.array(),
});

const createResZ = z.object({
  pids: pidRemoteZ.array(),
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
  pid: pidRemoteZ,
});

const CREATE_ENDPOINT = "/workspace/pid/create";
const DELETE_ENDPOINT = "/workspace/pid/delete";
const RENAME_ENDPOINT = "/workspace/pid/rename";
const SET_DATA_ENDPOINT = "/workspace/pid/set-data";
const COPY_ENDPOINT = "/workspace/pid/copy";

export class Writer {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: string, pid: NewPID): Promise<PID> {
    const pid_ = { ...pid, data: JSON.stringify(pid.data) };
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { workspace, pids: [pid_] },
      createReqZ,
      createResZ,
    );

    return res.pids[0];
  }

  async copy(key: Key, name: string, snapshot: boolean): Promise<PID> {
    const res = await sendRequired<typeof copyReqZ, typeof copyResZ>(
      this.client,
      COPY_ENDPOINT,
      { key, name, snapshot },
      copyReqZ,
      copyResZ,
    );
    return res.pid;
  }

  async delete(params: Params): Promise<void> {
    const normalized = toArray(params);
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: normalized },
      deleteReqZ,
      deleteResZ,
    );
  }

  async rename(pid: Key, name: string): Promise<void> {
    await sendRequired<typeof renameReqZ, typeof renameResZ>(
      this.client,
      RENAME_ENDPOINT,
      { key: pid, name },
      renameReqZ,
      renameResZ,
    );
  }

  async setData(pid: Key, data: UnknownRecord): Promise<void> {
    await sendRequired<typeof setDataReqZ, typeof setDataResZ>(
      this.client,
      SET_DATA_ENDPOINT,
      { key: pid, data: JSON.stringify(data) },
      setDataReqZ,
      renameResZ,
    );
  }
}
