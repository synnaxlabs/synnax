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

import {
  linePlotZ,
  type LinePlot,
  type Params,
  keyZ,
  type Key,
  linePlotRemoteZ,
} from "@/workspace/lineplot/payload";
import { keyZ as workspaceKeyZ } from "@/workspace/payload";

export const crudeLinePlotZ = linePlotZ.partial({ key: true });
export const linePlotWriteZ = linePlotRemoteZ.partial({ key: true });

export type CrudeLinePlot = z.infer<typeof crudeLinePlotZ>;

const createReqZ = z.object({
  workspace: workspaceKeyZ,
  linePlots: linePlotWriteZ.array(),
});

const createResZ = z.object({
  linePlots: linePlotRemoteZ.array(),
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

const CREATE_ENDPOINT = "/workspace/lineplot/create";
const DELETE_ENDPOINT = "/workspace/lineplot/delete";
const RENAME_ENDPOINT = "/workspace/lineplot/rename";
const SET_DATA_ENDPOINT = "/workspace/lineplot/set-data";

export class Writer {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: string, plot: CrudeLinePlot): Promise<LinePlot> {
    const pid_ = { ...plot, data: JSON.stringify(plot.data) };
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { workspace, linePlots: [pid_] },
      createResZ,
    );

    return res.linePlots[0];
  }

  async delete(params: Params): Promise<void> {
    const normalized = toArray(params);
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: normalized },
      deleteResZ,
    );
  }

  async rename(pid: Key, name: string): Promise<void> {
    await sendRequired<typeof renameReqZ, typeof renameResZ>(
      this.client,
      RENAME_ENDPOINT,
      { key: pid, name },
      renameResZ,
    );
  }

  async setData(pid: Key, data: UnknownRecord): Promise<void> {
    await sendRequired<typeof setDataReqZ, typeof setDataResZ>(
      this.client,
      SET_DATA_ENDPOINT,
      { key: pid, data: JSON.stringify(data) },
      renameResZ,
    );
  }
}
