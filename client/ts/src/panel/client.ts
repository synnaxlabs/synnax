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

import { type Action, actionZ, rename as renameAction } from "@/panel/actions.gen";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  type Panel,
  panelZ,
} from "@/panel/types.gen";
import { keyZ as projectKeyZ } from "@/project/types.gen";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  project: projectKeyZ.optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
});
export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}
const createReqZ = z.object({ panels: newZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveResZ = z.object({ panels: array.nullishToEmpty(panelZ) });
const createResZ = z.object({ panels: panelZ.array() });
const emptyResZ = z.object({});

const dispatchReqZ = z.object({
  key: keyZ,
  session_key: z.string(),
  actions: actionZ.array(),
});

// The server emits the action frame as snake_case JSON, but the framer's JSON
// codec runs snakeToCamel before handing the value to the schema, so this stays
// camelCase. seq is the server's monotonic high-water mark used by the store to
// drop stale echoes; it defaults to 0 to keep frames from servers that predate
// the field parseable.
export const scopedActionZ = z.object({
  key: keyZ,
  sessionKey: z.string(),
  seq: z.number().int().nonnegative().default(0),
  actions: actionZ.array(),
});
export interface ScopedAction extends z.infer<typeof scopedActionZ> {}

export const SET_CHANNEL_NAME = "sy_panel_set";
export const DELETE_CHANNEL_NAME = "sy_panel_delete";

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(panel: New): Promise<Panel>;
  async create(panels: New[]): Promise<Panel[]>;
  async create(panels: New | New[]): Promise<Panel | Panel[]> {
    const isMany = Array.isArray(panels);
    const res = await sendRequired(
      this.client,
      "/panel/create",
      { panels: array.toArray(panels) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.panels : res.panels[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    // Rename routes through dispatch so the action channel broadcasts the change
    // to other connected clients (and so the Rename action handler can promote
    // a draft to project ownership in the same transaction, see panel.actions
    // on the server). The /panel/rename REST endpoint remains available for
    // callers that do not want to construct an action vector themselves.
    await this.dispatch(key, "", [renameAction({ name })]);
  }

  async dispatch(key: Key, sessionKey: string, actions: Action[]): Promise<void> {
    await sendRequired(
      this.client,
      "/panel/dispatch",
      { key, session_key: sessionKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async retrieve(key: Key): Promise<Panel>;
  async retrieve(keys: Key[]): Promise<Panel[]>;
  async retrieve(req: RetrieveRequest): Promise<Panel[]>;
  async retrieve(keys: Key | Key[] | RetrieveRequest): Promise<Panel | Panel[]> {
    let req: RetrieveRequest;
    const isMany: boolean = typeof keys !== "string";
    if (typeof keys === "string" || Array.isArray(keys))
      req = { keys: array.toArray(keys) };
    else req = keys;
    const res = await sendRequired(
      this.client,
      "/panel/retrieve",
      req,
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.panels : res.panels[0];
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired(
      this.client,
      "/panel/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}
