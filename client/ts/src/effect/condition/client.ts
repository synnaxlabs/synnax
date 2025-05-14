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
  type Condition,
  conditionZ,
  type Key,
  keyZ,
  type New,
  newZ,
  type Params,
} from "@/effect/condition/payload";

const CREATE_ENDPOINT = "/effect/condition/create";
const DELETE_ENDPOINT = "/effect/condition/delete";
const RETRIEVE_ENDPOINT = "/effect/condition/retrieve";

const createReqZ = z.object({ conditions: z.array(newZ) });
const createResZ = z.object({ conditions: z.array(conditionZ) });
const deleteReqZ = z.object({ conditions: z.array(conditionZ) });
const retrieveReqZ = z.object({ keys: z.array(keyZ) });
const retrieveResZ = z.object({ conditions: z.array(conditionZ) });
const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(condition: New): Promise<Condition>;
  async create(conditions: New[]): Promise<Condition[]>;
  async create(conditions: New | New[]): Promise<Condition | Condition[]> {
    const isMany = Array.isArray(conditions);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { conditions: toArray(conditions) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.conditions : res.conditions[0];
  }

  async delete(condition: Condition): Promise<void>;
  async delete(conditions: Condition[]): Promise<void>;
  async delete(conditions: Condition | Condition[]): Promise<void> {
    await sendRequired(
      this.client,
      DELETE_ENDPOINT,
      { conditions: toArray(conditions) },
      deleteReqZ,
      emptyResZ,
    );
  }

  async retrieve(key: Key): Promise<Condition>;
  async retrieve(keys: Key[]): Promise<Condition[]>;
  async retrieve(keys: Params): Promise<Condition | Condition[]> {
    const isMany = Array.isArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: toArray(keys) },
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.conditions : res.conditions[0];
  }
}
