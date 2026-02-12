// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array, record } from "@synnaxlabs/x";
import { z } from "zod";

import { ontology } from "@/ontology";
import { keyZ as userKeyZ } from "@/user/payload";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  type Workspace,
  workspaceZ,
} from "@/workspace/payload";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  author: userKeyZ.optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
});
export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}
const createReqZ = z.object({ workspaces: newZ.array() });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const setLayoutReqZ = z.object({
  key: keyZ,
  layout: record.unknownZ(),
});
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveResZ = z.object({ workspaces: array.nullishToEmpty(workspaceZ) });
const createResZ = z.object({ workspaces: workspaceZ.array() });
const emptyResZ = z.object({});

export const SET_CHANNEL_NAME = "sy_workspace_set";
export const DELETE_CHANNEL_NAME = "sy_workspace_delete";

export interface SetLayoutArgs extends z.input<typeof setLayoutReqZ> {}

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(workspace: New): Promise<Workspace>;
  async create(workspaces: New[]): Promise<Workspace[]>;
  async create(workspaces: New | New[]): Promise<Workspace | Workspace[]> {
    const isMany = Array.isArray(workspaces);
    const res = await sendRequired(
      this.client,
      "/workspace/create",
      { workspaces: array.toArray(workspaces) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.workspaces : res.workspaces[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await sendRequired(
      this.client,
      "/workspace/rename",
      { key, name },
      renameReqZ,
      emptyResZ,
    );
  }

  async setLayout(key: Key, layout: record.Unknown): Promise<void> {
    await sendRequired(
      this.client,
      "/workspace/set-layout",
      { key, layout },
      setLayoutReqZ,
      emptyResZ,
    );
  }

  async retrieve(key: Key): Promise<Workspace>;
  async retrieve(keys: Key[]): Promise<Workspace[]>;
  async retrieve(req: RetrieveRequest): Promise<Workspace[]>;
  async retrieve(
    keys: Key | Key[] | RetrieveRequest,
  ): Promise<Workspace | Workspace[]> {
    let req: RetrieveRequest;
    const isMany: boolean = typeof keys !== "string";
    if (typeof keys === "string" || Array.isArray(keys))
      req = { keys: array.toArray(keys) };
    else req = keys;
    const res = await sendRequired(
      this.client,
      "/workspace/retrieve",
      req,
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.workspaces : res.workspaces[0];
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired(
      this.client,
      "/workspace/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}
