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

import { type Action, dispatchReqZ, rename as renameAction } from "@/panel/actions.gen";
import { type Key, keyZ, type New, type Panel, panelZ } from "@/panel/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
});
export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}
const createReqZ = z.object({ panels: panelZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveResZ = z.object({ panels: panelZ.array().default(() => []) });
const createResZ = z.object({ panels: panelZ.array() });
const emptyResZ = z.object({});

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
    const res = await this.client.send(
      "/panel/create",
      { panels: array.toArray(panels) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.panels : res.panels[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    // Rename routes through dispatch so the action channel broadcasts the change
    // to other connected clients.
    await this.dispatch(key, "", [renameAction({ name })]);
  }

  async dispatch(key: Key, dispatchKey: string, actions: Action[]): Promise<void> {
    await this.client.send(
      "/panel/dispatch",
      { key, dispatchKey, actions },
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
    const res = await this.client.send(
      "/panel/retrieve",
      req,
      retrieveReqZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Panel", keys, res.panels, !isMany);
    return isMany ? res.panels : res.panels[0];
  }

  async delete(key: Key): Promise<void>;
  async delete(keys: Key[]): Promise<void>;
  async delete(keys: Key | Key[]): Promise<void> {
    await this.client.send(
      "/panel/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}
