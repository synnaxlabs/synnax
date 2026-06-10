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
  type Action,
  dispatchReqZ,
  rename as renameAction,
} from "@/lineplot/actions.gen";
import {
  type Key,
  keyZ,
  type LinePlot,
  linePlotZ,
  type New,
  newZ,
} from "@/lineplot/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { project } from "@/project";

export const SET_CHANNEL_NAME = "sy_lineplot_set";

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

const createReqZ = z.object({ project: project.keyZ, linePlots: newZ.array() });
const createResZ = z.object({ linePlots: linePlotZ.array() });

const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(project: project.Key, linePlot: New): Promise<LinePlot>;
  async create(project: project.Key, linePlots: New[]): Promise<LinePlot[]>;
  async create(
    project: project.Key,
    linePlots: New | New[],
  ): Promise<LinePlot | LinePlot[]> {
    const isMany = Array.isArray(linePlots);
    const res = await this.client.send(
      "/lineplot/create",
      { project, linePlots: array.toArray(linePlots) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.linePlots : res.linePlots[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatch(key, "", [renameAction({ name })]);
  }

  async dispatch(key: Key, dispatchKey: string, actions: Action[]): Promise<void> {
    await this.client.send(
      "/lineplot/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
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
