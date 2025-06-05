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
  type Effect,
  effectZ,
  type Key,
  keyZ,
  type New,
  newZ,
  type Params,
  type State,
  stateZ,
} from "@/effect/payload";
import { framer } from "@/framer";

const STATE_CHANNEL_NAME = "sy_effect_state";

const CREATE_ENDPOINT = "/effect/create";
const DELETE_ENDPOINT = "/effect/delete";
const RETRIEVE_ENDPOINT = "/effect/retrieve";

const createReqZ = z.object({ effects: z.array(newZ) });
const createResZ = z.object({ effects: z.array(effectZ) });
const deleteReqZ = z.object({ effects: z.array(effectZ) });
const retrieveReqZ = z.object({ keys: z.array(keyZ) });
const retrieveResZ = z.object({ effects: z.array(effectZ) });
const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;
  private readonly frameClient: framer.Client;

  constructor(client: UnaryClient, frameClient: framer.Client) {
    this.client = client;
    this.frameClient = frameClient;
  }

  async create(effect: New): Promise<Effect>;
  async create(effects: New[]): Promise<Effect[]>;
  async create(effects: New | New[]): Promise<Effect | Effect[]> {
    const isMany = Array.isArray(effects);
    const res = await sendRequired(
      this.client,
      CREATE_ENDPOINT,
      { effects: toArray(effects) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.effects : res.effects[0];
  }

  async delete(effect: Effect): Promise<void>;
  async delete(effects: Effect[]): Promise<void>;
  async delete(effects: Effect | Effect[]): Promise<void> {
    await sendRequired(
      this.client,
      DELETE_ENDPOINT,
      { effects: toArray(effects) },
      deleteReqZ,
      emptyResZ,
    );
  }

  async retrieve(key: Key): Promise<Effect>;
  async retrieve(keys: Key[]): Promise<Effect[]>;
  async retrieve(keys: Params): Promise<Effect | Effect[]> {
    const isMany = Array.isArray(keys);
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      { keys: toArray(keys) },
      retrieveReqZ,
      retrieveResZ,
    );
    return isMany ? res.effects : res.effects[0];
  }

  async openStateObserver(): Promise<framer.ObservableStreamer<State[]>> {
    return new framer.ObservableStreamer<State[]>(
      await this.frameClient.openStreamer(STATE_CHANNEL_NAME),
      (frame) => {
        const s = frame.get(STATE_CHANNEL_NAME);
        if (s.length === 0) return [null, false];
        const states = s.parseJSON(stateZ);
        return [states as State[], true];
      },
    );
  }
}
