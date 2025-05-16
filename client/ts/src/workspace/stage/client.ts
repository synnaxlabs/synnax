// Copyright 2025 Synnax Labs, Inc.
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

import { ontology } from "@/ontology";
import { nullableArrayZ } from "@/util/zod";
import { type Key as WorkspaceKey, keyZ as workspaceKeyZ } from "@/workspace/payload";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  ONTOLOGY_TYPE,
  type Params,
  remoteZ,
  type Stage,
  stageZ,
} from "@/workspace/stage/payload";

const RETRIEVE_ENDPOINT = "/workspace/stage/retrieve";
const CREATE_ENDPOINT = "/workspace/stage/create";
const RENAME_ENDPOINT = "/workspace/stage/rename";
const SET_DATA_ENDPOINT = "/workspace/stage/set-data";
const DELETE_ENDPOINT = "/workspace/stage/delete";
const COPY_ENDPOINT = "/workspace/stage/copy";

const retrieveReqZ = z.object({ keys: keyZ.array() });
const createReqZ = z.object({ workspace: workspaceKeyZ, stages: newZ.array() });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const setDataReqZ = z.object({ key: keyZ, data: z.string() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const copyReqZ = z.object({ key: keyZ, name: z.string(), snapshot: z.boolean() });

const retrieveResZ = z.object({ stages: nullableArrayZ(remoteZ) });
const createResZ = z.object({ stages: remoteZ.array() });
const copyResZ = z.object({ stage: stageZ });
const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: WorkspaceKey, stage: New): Promise<Stage>;
  async create(workspace: WorkspaceKey, stages: New[]): Promise<Stage[]>;
  async create(
    workspace: WorkspaceKey,
    stages: New | New[],
  ): Promise<Stage | Stage[]> {
    const isMany = Array.isArray(stages);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { workspace, stages: toArray(stages) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.stages : res.stages[0];
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

  async setData(key: Key, data: UnknownRecord): Promise<void> {
    await sendRequired(
      this.client,
      SET_DATA_ENDPOINT,
      { key, data: JSON.stringify(data) },
      setDataReqZ,
      emptyResZ,
    );
  }

  async retrieve(key: Key): Promise<Stage>;
  async retrieve(keys: Key[]): Promise<Stage[]>;
  async retrieve(keys: Params): Promise<Stage | Stage[]> {
    const isMany = Array.isArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: toArray(keys) },
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.stages : res.stages[0];
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
  async delete(keys: Params): Promise<void> {
    await sendRequired(
      this.client,
      DELETE_ENDPOINT,
      { keys: toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }

  async copy(key: Key, name: string, snapshot: boolean): Promise<Stage> {
    const res = await sendRequired(
      this.client,
      COPY_ENDPOINT,
      { key, name, snapshot },
      copyReqZ,
      copyResZ,
    );
    return res.stage;
  }
}

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });
