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
  type LinePlot,
  linePlotZ,
  type Params,
} from "@/workspace/lineplot/payload";
import { keyZ as workspaceKeyZ } from "@/workspace/payload";

export const newLinePlotZ = linePlotZ.partial({ key: true }).transform((p) => ({
  ...p,
  data: JSON.stringify(p.data),
}));

export type NewLinePlot = z.input<typeof newLinePlotZ>;

const createReqZ = z.object({
  workspace: workspaceKeyZ,
  linePlots: newLinePlotZ.array(),
});

const createResZ = z.object({
  linePlots: linePlotZ.array(),
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

  async create(workspace: string, plot: NewLinePlot): Promise<LinePlot> {
    const linePlot = { ...plot, data: JSON.stringify(plot.data) };
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { workspace, linePlots: [linePlot] },
      createReqZ,
      createResZ,
    );

    return res.linePlots[0];
  }

  async delete(workspaces: Params): Promise<void> {
    const normalized = toArray(workspaces);
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: normalized },
      deleteReqZ,
      deleteResZ,
    );
  }

  async rename(plot: Key, name: string): Promise<void> {
    await sendRequired<typeof renameReqZ, typeof renameResZ>(
      this.client,
      RENAME_ENDPOINT,
      { key: plot, name },
      renameReqZ,
      renameResZ,
    );
  }

  async setData(plot: Key, data: UnknownRecord): Promise<void> {
    await sendRequired<typeof setDataReqZ, typeof setDataResZ>(
      this.client,
      SET_DATA_ENDPOINT,
      { key: plot, data: JSON.stringify(data) },
      setDataReqZ,
      setDataResZ,
    );
  }
}
