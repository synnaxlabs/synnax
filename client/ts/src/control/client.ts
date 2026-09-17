// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { type channel } from "@/channel";
import { keyZ } from "@/channel/types.gen";
import { type KeyedState, keyedStateZ, updateZ } from "@/control/state";
import { query } from "@/query";

/** The free virtual channel carrying control updates for the entire cluster. */
const CHANNEL_NAME = "sy_control";

const RETRIEVE_ENDPOINT = "/control/retrieve";

const retrieveRequestZ = z.object({ keys: keyZ.array().optional() });
const retrieveMultiParamsZ = retrieveRequestZ.or(query.keyListZ(keyZ));
const retrieveResZ = z.object({ states: keyedStateZ.array().default(() => []) });

export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;

interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
}

/**
 * Reads the control state of channels across the cluster. A channel is in the table
 * exactly while a subject holds it, so absence is the uncontrolled state: a release
 * deletes the entry instead of storing a sentinel.
 */
export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  channel.Key,
  KeyedState
> {
  readonly type: string = "control";
  /** The control state table, keyed by the channel under control. */
  readonly store: query.Table<channel.Key, KeyedState>;
  private readonly cfg: ClientConfig;

  constructor(cfg: ClientConfig) {
    const store = cfg.cache.createTable<channel.Key, KeyedState>({
      name: "control",
      fetch: async (keys) => await this.execRetrieve({ keys }),
      listen: [transferListener()],
    });
    super(cfg.cache, {
      name: "control",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (state, req) => req.keys == null || req.keys.includes(state.key),
      },
    });
    this.cfg = cfg;
    this.store = store;
  }

  private async execRetrieve(req: RetrieveRequest): Promise<KeyedState[]> {
    // An empty key set reads the whole cluster, so it must not reach the Core here.
    if (req.keys != null && req.keys.length === 0) return [];
    const { states } = await this.cfg.unary.send(
      RETRIEVE_ENDPOINT,
      { keys: req.keys ?? [] },
      retrieveRequestZ,
      retrieveResZ,
    );
    return states;
  }
}

/**
 * Mirrors the cluster's control transfers into the table: the `to` end of a transfer
 * sets the channel's holder, and a transfer without one deletes the entry.
 */
const transferListener = (): query.ListenerSpec<channel.Key, KeyedState> => ({
  bind: (table): query.Listener<typeof updateZ> => ({
    channel: CHANNEL_NAME,
    schema: updateZ,
    onChange: (updates) =>
      table.batch(() =>
        updates
          .flatMap((u) => u.transfers)
          .forEach(({ from, to }) => {
            if (to == null) table.delete(from.resource);
            else table.set({ key: to.resource, ...to });
          }),
      ),
  }),
});
