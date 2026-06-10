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

import { type Action, dispatchReqZ, rename as renameAction } from "@/log/actions.gen";
import { type Key, keyZ, type Log, logZ, type New, newZ } from "@/log/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { project } from "@/project";

export const SET_CHANNEL_NAME = "sy_log_set";

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

const createReqZ = z.object({ project: project.keyZ, logs: newZ.array() });
const createResZ = z.object({ logs: logZ.array() });

const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(project: project.Key, log: New): Promise<Log>;
  async create(project: project.Key, logs: New[]): Promise<Log[]>;
  async create(project: project.Key, logs: New | New[]): Promise<Log | Log[]> {
    const isMany = Array.isArray(logs);
    const res = await this.client.send(
      "/log/create",
      { project, logs: array.toArray(logs) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.logs : res.logs[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatch(key, "", [renameAction({ name })]);
  }

  async dispatch(key: Key, dispatchKey: string, actions: Action[]): Promise<void> {
    await this.client.send(
      "/log/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async retrieve(args: RetrieveSingleParams): Promise<Log>;
  async retrieve(args: RetrieveMultipleParams): Promise<Log[]>;
  async retrieve(
    args: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<Log | Log[]> {
    const isSingle = singleRetrieveArgsZ.safeParse(args).success;
    const res = await this.client.send(
      "/log/retrieve",
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Log", args, res.logs, isSingle);
    return isSingle ? res.logs[0] : res.logs;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await this.client.send(
      "/log/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}
