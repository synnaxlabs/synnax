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
import { unknownRecordZ } from "@synnaxlabs/x/record";
import { z } from "zod";

import { ontology } from "@/ontology";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;
export type Params = Key | Key[];

export const logZ = z.object({
  key: z.string(),
  name: z.string(),
  data: unknownRecordZ.or(z.string().transform((s) => JSON.parse(s) as UnknownRecord)),
});

export type Log = z.infer<typeof logZ>;

export const ONTOLOGY_TYPE: ontology.ResourceType = "log";

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });

const RETRIEVE_ENDPOINT = "/workspace/log/retrieve";
const CREATE_ENDPOINT = "/workspace/log/create";
const RENAME_ENDPOINT = "/workspace/log/rename";
const SET_DATA_ENDPOINT = "/workspace/log/set-data";
const DELETE_ENDPOINT = "/workspace/log/delete";

export const newLogZ = logZ.partial({ key: true }).transform((p) => ({
  ...p,
  data: JSON.stringify(p.data),
}));

export type NewLog = z.input<typeof newLogZ>;

const retrieveReqZ = z.object({ keys: z.string().array() });
const createReqZ = z.object({ workspace: z.string(), logs: newLogZ.array() });
const renameReqZ = z.object({ key: z.string(), name: z.string() });
const setDataReqZ = z.object({ key: z.string(), data: z.string() });
const deleteReqZ = z.object({ keys: z.string().array() });

const retrieveResZ = z.object({ logs: logZ.array() });
const createResZ = z.object({ logs: logZ.array() });
const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: string, log: NewLog): Promise<Log>;
  async create(workspace: string, logs: NewLog[]): Promise<Log[]>;
  async create(workspace: string, logs: NewLog | NewLog[]): Promise<Log | Log[]> {
    const isMany = Array.isArray(logs);
    const normalized = toArray(logs);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { workspace, logs: normalized },
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

  async setData(key: Key, data: UnknownRecord): Promise<void> {
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
    const normalized = toArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: normalized },
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.logs : res.logs[0];
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
  async delete(keys: Params): Promise<void> {
    const normalized = toArray(keys);
    await sendRequired(
      this.client,
      DELETE_ENDPOINT,
      { keys: normalized },
      deleteReqZ,
      emptyResZ,
    );
  }
}
