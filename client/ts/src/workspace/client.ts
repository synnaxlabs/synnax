// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { toArray, type UnknownRecord } from "@synnaxlabs/x";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { z } from "zod";

import { ontology } from "@/ontology";
import { type Key as UserKey, keyZ as userKeyZ } from "@/user/payload";
import { nullableArrayZ } from "@/util/zod";
import { linePlot } from "@/workspace/lineplot";
import { log } from "@/workspace/log";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  ONTOLOGY_TYPE,
  type Params,
  remoteZ,
  type Workspace,
  workspaceZ,
} from "@/workspace/payload";
import { schematic } from "@/workspace/schematic";
import { table } from "@/workspace/table";

const RETRIEVE_ENDPOINT = "/workspace/retrieve";
const CREATE_ENDPOINT = "/workspace/create";
const RENAME_ENDPOINT = "/workspace/rename";
const SET_LAYOUT_ENDPOINT = "/workspace/set-layout";
const DELETE_ENDPOINT = "/workspace/delete";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  search: z.string().optional(),
  author: userKeyZ.optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});
const createReqZ = z.object({ workspaces: newZ.array() });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const setLayoutReqZ = z.object({ key: keyZ, layout: z.string() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveResZ = z.object({ workspaces: nullableArrayZ(workspaceZ) });
const createResZ = z.object({ workspaces: remoteZ.array() });
const emptyResZ = z.object({});

export class Client implements AsyncTermSearcher<string, Key, Workspace> {
  readonly type = ONTOLOGY_TYPE;
  readonly schematic: schematic.Client;
  readonly linePlot: linePlot.Client;
  readonly log: log.Client;
  readonly table: table.Client;
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
    this.schematic = new schematic.Client(client);
    this.linePlot = new linePlot.Client(client);
    this.log = new log.Client(client);
    this.table = new table.Client(client);
  }

  async create(workspace: New): Promise<Workspace>;
  async create(workspaces: New[]): Promise<Workspace[]>;
  async create(workspaces: New | New[]): Promise<Workspace | Workspace[]> {
    const isMany = Array.isArray(workspaces);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { workspaces: toArray(workspaces) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.workspaces : res.workspaces[0];
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

  async setLayout(key: Key, layout: UnknownRecord): Promise<void> {
    await sendRequired(
      this.client,
      SET_LAYOUT_ENDPOINT,
      { key, layout: JSON.stringify(layout) },
      setLayoutReqZ,
      emptyResZ,
    );
  }

  async retrieve(key: Key): Promise<Workspace>;
  async retrieve(keys: Key[]): Promise<Workspace[]>;
  async retrieve(keys: Params): Promise<Workspace | Workspace[]> {
    const isMany = Array.isArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: toArray(keys) },
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.workspaces : res.workspaces[0];
  }

  async retrieveByAuthor(author: UserKey): Promise<Workspace[]> {
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { author },
      retrieveReqZ,
      retrieveResZ,
    );
    return res.workspaces;
  }

  async search(term: string): Promise<Workspace[]> {
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { search: term },
      retrieveReqZ,
      retrieveResZ,
    );
    return res.workspaces;
  }

  async page(offset: number, limit: number): Promise<Workspace[]> {
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { offset, limit },
      retrieveReqZ,
      retrieveResZ,
    );
    return res.workspaces;
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
}

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key });
