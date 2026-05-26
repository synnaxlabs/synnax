// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Axis,
  type AxisKey,
  type Key,
  keyZ,
  type LinePlot,
  linePlotZ,
  type New,
  newZ,
} from "@/lineplot/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { workspace } from "@/workspace";

const renameReqZ = z.object({ key: keyZ, name: z.string() });

export type SetDataBody = Omit<LinePlot, "key" | "name">;
const setDataBodyZ = linePlotZ.omit({ key: true, name: true });
const setDataReqZ = z.object({ key: keyZ, data: setDataBodyZ });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveReqZ = z.object({ keys: keyZ.array() });
const singleRetrieveArgsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

export const retrieveArgsZ = z.union([singleRetrieveArgsZ, retrieveReqZ]);
export type RetrieveArgs = z.input<typeof retrieveArgsZ>;
export type RetrieveSingleParams = z.input<typeof singleRetrieveArgsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveReqZ>;

const retrieveResZ = z.object({ linePlots: array.nullishToEmpty(linePlotZ) });

const createReqZ = z.object({ workspace: workspace.keyZ, linePlots: newZ.array() });
const createResZ = z.object({ linePlots: linePlotZ.array() });

const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: workspace.Key, linePlot: New): Promise<LinePlot>;
  async create(workspace: workspace.Key, linePlots: New[]): Promise<LinePlot[]>;
  async create(
    workspace: workspace.Key,
    linePlots: New | New[],
  ): Promise<LinePlot | LinePlot[]> {
    const isMany = Array.isArray(linePlots);
    const res = await this.client.send(
      "/lineplot/create",
      { workspace, linePlots: array.toArray(linePlots) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.linePlots : res.linePlots[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.client.send("/lineplot/rename", { key, name }, renameReqZ, emptyResZ);
  }

  async setData(key: Key, data: SetDataBody): Promise<void> {
    await this.client.send("/lineplot/set-data", { key, data }, setDataReqZ, emptyResZ);
  }

  async retrieve(args: RetrieveSingleParams): Promise<LinePlot>;
  async retrieve(args: RetrieveMultipleParams): Promise<LinePlot[]>;
  async retrieve(
    args: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<LinePlot | LinePlot[]> {
    const isSingle = singleRetrieveArgsZ.safeParse(args).success;
    const res = await this.client.send(
      "/lineplot/retrieve",
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("LinePlot", args, res.linePlots, isSingle);
    return isSingle ? res.linePlots[0] : res.linePlots;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await this.client.send(
      "/lineplot/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}

const zeroAxis = (key: AxisKey): Axis => ({
  key,
  label: "",
  labelDirection: key.startsWith("y") ? "y" : "x",
  labelLevel: "small",
  bounds: { lower: 0, upper: 0 },
  autoBounds: { lower: true, upper: true },
  tickSpacing: 75,
  ...(key.startsWith("x") ? ({ type: "time" } as const) : {}),
});

// ZERO_NEW is a fully-populated New payload with every required field set to
// its zero value. Callers spread it and override only the fields they care
// about: `client.lineplots.create(ws, { ...ZERO_NEW, name: "My Plot" })`.
export const ZERO_NEW: New = {
  name: "",
  title: { level: "h4", visible: false },
  legend: {
    visible: true,
    position: {
      x: 50,
      y: 50,
      root: { x: "left", y: "top" },
      units: { x: "px", y: "px" },
    },
  },
  channels: { x1: 0, x2: 0, y1: [], y2: [], y3: [], y4: [] },
  ranges: { x1: [], x2: [] },
  axes: {
    x1: zeroAxis("x1"),
    x2: zeroAxis("x2"),
    y1: zeroAxis("y1"),
    y2: zeroAxis("y2"),
    y3: zeroAxis("y3"),
    y4: zeroAxis("y4"),
  },
  lines: [],
  rules: [],
};
