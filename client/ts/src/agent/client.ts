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
import { z } from "zod/v4";

import { type Agent, agentZ, type Key, keyZ, type New, newZ, type Params } from "@/agent/payload";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_agent_set";
export const DELETE_CHANNEL_NAME = "sy_agent_delete";

const createReqZ = z.object({ agents: newZ.array() });
const createResZ = z.object({ agents: agentZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const sendReqZ = z.object({ key: keyZ, content: z.string() });
const sendResZ = z.object({ agent: agentZ });

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
});
const retrieveResZ = z.object({ agents: array.nullableZ(agentZ) });

export type RetrieveRequest = z.input<typeof retrieveReqZ>;

const keyRetrieveRequestZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

export type SingleRetrieveArgs = z.input<typeof keyRetrieveRequestZ>;

const retrieveArgsZ = z.union([keyRetrieveRequestZ, retrieveReqZ]);
export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(agent: New): Promise<Agent>;
  async create(agents: New[]): Promise<Agent[]>;
  async create(agents: New | New[]): Promise<Agent | Agent[]> {
    const isMany = Array.isArray(agents);
    const res = await sendRequired(
      this.client,
      "/agent/create",
      { agents: array.toArray(agents) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.agents : res.agents[0];
  }

  async send(key: Key, content: string): Promise<Agent> {
    const res = await sendRequired(
      this.client,
      "/agent/send",
      { key, content },
      sendReqZ,
      sendResZ,
    );
    return res.agent;
  }

  async retrieve(key: Key): Promise<Agent>;
  async retrieve(keys: Params): Promise<Agent[]>;
  async retrieve(req: RetrieveRequest): Promise<Agent[]>;
  async retrieve(keys: Params | RetrieveRequest): Promise<Agent | Agent[]> {
    const isSingle = typeof keys === "string";
    const req =
      typeof keys === "string" || Array.isArray(keys)
        ? { keys: array.toArray(keys) }
        : keys;
    const res = await sendRequired(
      this.client,
      "/agent/retrieve",
      req,
      retrieveArgsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Agent", req, res.agents, isSingle);
    return isSingle ? res.agents[0] : res.agents;
  }

  async delete(keys: Params): Promise<void> {
    await sendRequired(
      this.client,
      "/agent/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}
