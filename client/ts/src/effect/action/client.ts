// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { toArray } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Action,
  actionZ,
  type Key,
  keyZ,
  type New,
  newZ,
  type Params,
} from "@/effect/action/payload";

const CREATE_ENDPOINT = "/effect/action/create";
const DELETE_ENDPOINT = "/effect/action/delete";
const RETRIEVE_ENDPOINT = "/effect/action/retrieve";

const createReqZ = z.object({ actions: z.array(newZ) });
const createResZ = z.object({ actions: z.array(actionZ) });
const deleteReqZ = z.object({ actions: z.array(actionZ) });
const retrieveReqZ = z.object({ keys: z.array(keyZ) });
const retrieveResZ = z.object({ actions: z.array(actionZ) });
const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(action: New): Promise<Action>;
  async create(actions: New[]): Promise<Action[]>;
  async create(actions: New | New[]): Promise<Action | Action[]> {
    const isMany = Array.isArray(actions);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { actions: toArray(actions) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.actions : res.actions[0];
  }

  async delete(action: Action): Promise<void>;
  async delete(actions: Action[]): Promise<void>;
  async delete(actions: Action | Action[]): Promise<void> {
    await sendRequired(
      this.client,
      DELETE_ENDPOINT,
      { actions: toArray(actions) },
      deleteReqZ,
      emptyResZ,
    );
  }

  async retrieve(key: Key): Promise<Action>;
  async retrieve(keys: Key[]): Promise<Action[]>;
  async retrieve(keys: Params): Promise<Action | Action[]> {
    const isMany = Array.isArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: toArray(keys) },
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.actions : res.actions[0];
  }
}
