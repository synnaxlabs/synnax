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

import { type Workspace, workspaceZ, keyZ, workspaceRemoteZ } from "./payload";

const crudeWorkspaceZ = workspaceZ.partial({ key: true });
const workspaceWriteZ = workspaceRemoteZ.partial({ key: true });

export type CrudeWorkspace = z.infer<typeof crudeWorkspaceZ>;

const createReqZ = z.object({
  workspaces: workspaceWriteZ.partial({ key: true }).array(),
});

const createResZ = z.object({
  workspaces: workspaceRemoteZ.array(),
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

const setLayoutReqZ = z.object({
  key: keyZ,
  layout: z.string(),
});

const setLayoutResZ = z.object({});

export type Response = z.infer<typeof createResZ>;

const CREATE_ENDPOINT = "/workspace/create";
const DELETE_ENDPOINT = "/workspace/delete";
const RENAME_ENDPOINT = "/workspace/rename";
const SET_LAYOUT_ENDPOINT = "/workspace/set-layout";

export class Writer {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspaces: CrudeWorkspace | CrudeWorkspace[]): Promise<Workspace[]> {
    const ws = toArray(workspaces).map((w) => ({
      ...w,
      layout: JSON.stringify(w.layout),
    }));
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { workspaces: ws },
      createResZ,
    );
    return res.workspaces;
  }

  async delete(keys: string | string[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: toArray(keys) },
      deleteResZ,
    );
  }

  async rename(key: string, name: string): Promise<void> {
    await sendRequired<typeof renameReqZ, typeof renameResZ>(
      this.client,
      RENAME_ENDPOINT,
      { key, name },
      renameResZ,
    );
  }

  async setLayout(key: string, layout: UnknownRecord): Promise<void> {
    await sendRequired<typeof setLayoutReqZ, typeof setLayoutResZ>(
      this.client,
      SET_LAYOUT_ENDPOINT,
      { key, layout: JSON.stringify(layout) },
      setLayoutResZ,
    );
  }
}
