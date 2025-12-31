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

import { checkForMultipleOrNoResults } from "@/util/retrieve";
import {
  type Key,
  keyZ,
  type Log,
  logZ,
  type New,
  newZ,
} from "@/workspace/log/types.gen";
import { type Key as WorkspaceKey, keyZ as workspaceKeyZ } from "@/workspace/types.gen";

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

const retrieveResZ = z.object({ logs: array.nullishToEmpty(logZ) });

const createReqZ = z.object({ workspace: workspaceKeyZ, logs: newZ.array() });
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
      "/workspace/log/create",
      { workspace, logs: array.toArray(logs) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.logs : res.logs[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await sendRequired(
      this.client,
      "/workspace/log/rename",
      { key, name },
      renameReqZ,
      emptyResZ,
    );
  }

  async setData(key: Key, data: record.Unknown): Promise<void> {
    await sendRequired(
      this.client,
      "/workspace/log/set-data",
      { key, data: JSON.stringify(data) },
      setDataReqZ,
      emptyResZ,
    );
  }

  async retrieve(args: RetrieveSingleParams): Promise<Log>;
  async retrieve(args: RetrieveMultipleParams): Promise<Log[]>;
  async retrieve(
    args: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<Log | Log[]> {
    const isSingle = singleRetrieveArgsZ.safeParse(args).success;
    const res = await sendRequired(
      this.client,
      "/workspace/log/retrieve",
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Log", args, res.logs, isSingle);
    return isSingle ? res.logs[0] : res.logs;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired(
      this.client,
      "/workspace/log/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}
