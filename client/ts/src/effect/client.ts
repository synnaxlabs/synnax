// Copyright 2025 Synnax Labs, Inc.
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

import { type Effect, effectZ, type Key, keyZ, type New, newZ } from "@/effect/payload";
import { type framer } from "@/framer";
import { slate } from "@/slate";
import { checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";

export const STATUS_CHANNEL_NAME = "sy_effect_status";
export const SET_CHANNEL_NAME = "sy_effect_set";
export const DELETE_CHANNEL_NAME = "sy_effect_delete";

const CREATE_ENDPOINT = "/effect/create";
const DELETE_ENDPOINT = "/effect/delete";
const RETRIEVE_ENDPOINT = "/effect/retrieve";
const VALIDATE_ENDPOINT = "/effect/validate";

const createReqZ = z.object({ effects: z.array(newZ) });
const createResZ = z.object({ effects: z.array(effectZ) });
const deleteReqZ = z.object({ keys: z.array(keyZ) });
const retrieveReqZ = z.object({
  keys: z.array(keyZ).optional(),
  term: z.string().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  includeStatus: z.boolean().optional(),
  includeLabels: z.boolean().optional(),
});
const retrieveResZ = z.object({ effects: nullableArrayZ(effectZ) });
const emptyResZ = z.object({});

export const validateReqZ = z.object({ graph: slate.graphZ });

export type RetrieveRequest = z.input<typeof retrieveReqZ>;

const keyRetrieveRequestZ = z
  .object({
    key: keyZ,
    includeStatus: z.boolean().optional(),
    includeLabels: z.boolean().optional(),
  })
  .transform(({ key, includeStatus, includeLabels }) => ({
    keys: [key],
    includeStatus,
    includeLabels,
  }));

export type KeyRetrieveRequest = z.input<typeof keyRetrieveRequestZ>;

const retrieveArgsZ = z.union([keyRetrieveRequestZ, retrieveReqZ]);

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

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
      { effects: array.toArray(effects) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.effects : res.effects[0];
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired(
      this.client,
      DELETE_ENDPOINT,
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }

  async retrieve(args: KeyRetrieveRequest): Promise<Effect>;
  async retrieve(args: RetrieveArgs): Promise<Effect[]>;
  async retrieve(args: RetrieveArgs): Promise<Effect | Effect[]> {
    const isSingle = "key" in args;
    const res = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      args,
      retrieveArgsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Effect", args, res.effects, isSingle);
    return isSingle ? res.effects[0] : res.effects;
  }

  async validate(graph: slate.Graph): Promise<void> {
    await sendRequired(
      this.client,
      VALIDATE_ENDPOINT,
      { graph },
      validateReqZ,
      emptyResZ,
    );
  }
}
