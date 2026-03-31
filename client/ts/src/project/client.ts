// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Key,
  keyZ,
  type New,
  newZ,
  type Project,
  projectZ,
} from "@/project/types.gen";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
});
export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}
const createReqZ = z.object({ projects: newZ.array() });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveResZ = z.object({ projects: array.nullishToEmpty(projectZ) });
const createResZ = z.object({ projects: projectZ.array() });
const emptyResZ = z.object({});

export const SET_CHANNEL_NAME = "sy_project_set";
export const DELETE_CHANNEL_NAME = "sy_project_delete";

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(project: New): Promise<Project>;
  async create(projects: New[]): Promise<Project[]>;
  async create(projects: New | New[]): Promise<Project | Project[]> {
    const isMany = Array.isArray(projects);
    const res = await sendRequired(
      this.client,
      "/project/create",
      { projects: array.toArray(projects) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.projects : res.projects[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await sendRequired(
      this.client,
      "/project/rename",
      { key, name },
      renameReqZ,
      emptyResZ,
    );
  }

  async retrieve(key: Key): Promise<Project>;
  async retrieve(keys: Key[]): Promise<Project[]>;
  async retrieve(req: RetrieveRequest): Promise<Project[]>;
  async retrieve(keys: Key | Key[] | RetrieveRequest): Promise<Project | Project[]> {
    let req: RetrieveRequest;
    const isMany: boolean = typeof keys !== "string";
    if (typeof keys === "string" || Array.isArray(keys))
      req = { keys: array.toArray(keys) };
    else req = keys;
    const res = await sendRequired(
      this.client,
      "/project/retrieve",
      req,
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.projects : res.projects[0];
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired(
      this.client,
      "/project/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}
