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
import {
  type Key,
  keyZ,
  type LinePlot,
  linePlotZ,
  type New,
  newZ,
  type Params,
} from "@/workspace/lineplot/payload";
import { type Key as WorkspaceKey, keyZ as workspaceKeyZ } from "@/workspace/payload";

const RETRIEVE_ENDPOINT = "/workspace/lineplot/retrieve";
const CREATE_ENDPOINT = "/workspace/lineplot/create";
const RENAME_ENDPOINT = "/workspace/lineplot/rename";
const SET_DATA_ENDPOINT = "/workspace/lineplot/set-data";
const DELETE_ENDPOINT = "/workspace/lineplot/delete";

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

const retrieveResZ = z.object({ linePlots: array.nullableZ(linePlotZ) });

const createReqZ = z.object({ workspace: workspaceKeyZ, linePlots: newZ.array() });
const createResZ = z.object({ linePlots: linePlotZ.array() });

const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: WorkspaceKey, linePlot: New): Promise<LinePlot>;
  async create(workspace: WorkspaceKey, linePlots: New[]): Promise<LinePlot[]>;
  async create(
    workspace: WorkspaceKey,
    linePlots: New | New[],
  ): Promise<LinePlot | LinePlot[]> {
    const isMany = Array.isArray(linePlots);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { workspace, linePlots: array.toArray(linePlots) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.linePlots : res.linePlots[0];
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

  async retrieve(args: RetrieveSingleParams): Promise<LinePlot>;
  async retrieve(args: RetrieveMultipleParams): Promise<LinePlot[]>;
  async retrieve(
    args: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<LinePlot | LinePlot[]> {
    const isSingle = singleRetrieveArgsZ.safeParse(args).success;
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("LinePlot", args, res.linePlots, isSingle);
    return isSingle ? res.linePlots[0] : res.linePlots;
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
}

export const ontologyID = (key: Key): ontology.ID => ({ type: "lineplot", key });
