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
import { z } from "zod/v4";

import { ontology } from "@/ontology";
import {
  type Key,
  keyZ,
  type Log,
  logZ,
  type New,
  newZ,
  ONTOLOGY_TYPE,
  type Params,
} from "@/workspace/log/payload";
import { type Key as WorkspaceKey, keyZ as workspaceKeyZ } from "@/workspace/payload";

const RETRIEVE_ENDPOINT = "/workspace/log/retrieve";
const CREATE_ENDPOINT = "/workspace/log/create";
const RENAME_ENDPOINT = "/workspace/log/rename";
const SET_DATA_ENDPOINT = "/workspace/log/set-data";
const DELETE_ENDPOINT = "/workspace/log/delete";

const retrieveReqZ = z.object({ keys: keyZ.array() });
const createReqZ = z.object({ workspace: workspaceKeyZ, logs: newZ.array() });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const setDataReqZ = z.object({ key: keyZ, data: z.string() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveResZ = z.object({ logs: logZ.array() });
const createResZ = z.object({ logs: logZ.array() });
const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: WorkspaceKey, log: New): Promise<Log>;
  async create(workspace: WorkspaceKey, logs: New[]): Promise<Log[]>;
  async create(workspace: WorkspaceKey, logs: New | New[]): Promise<Log | Log[]> {
    const isMany = Array.isArray(logs);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { workspace, logs: array.toArray(logs) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.logs : res.logs[0];
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

  async retrieve(key: Key): Promise<Log>;
  async retrieve(keys: Key[]): Promise<Log[]>;
  async retrieve(keys: Params): Promise<Log | Log[]> {
    const isMany = Array.isArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: array.toArray(keys) },
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.logs : res.logs[0];
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
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

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });
