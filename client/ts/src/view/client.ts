// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { ontology } from "@/ontology";
import { query } from "@/query";
import { type Condition } from "@/view/builder.gen";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  type View,
  viewZ,
} from "@/view/types.gen";
import { matches, serverComputed } from "@/view/where.gen";

export const SET_CHANNEL_NAME = "sy_view_set";
export const DELETE_CHANNEL_NAME = "sy_view_delete";

const createReqZ = z.object({ views: viewZ.array() });
const createResZ = z.object({ views: viewZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});
const renameReqZ = z.object({ key: keyZ, name: z.string() });

const retrieveMultiParamsZ = z.union([query.keysRequestZ(keyZ), retrieveRequestZ]);

export type RetrieveSingleParams = { key: Key };
export type RetrieveMultipleParams = RetrieveRequest;

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  ontology: ontology.Client;
}

export class Client extends query.Retriever<
  Condition,
  typeof retrieveMultiParamsZ,
  Key,
  View
> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, View>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, View>({
      name: "views",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, viewZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "view",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (view, req) => matches(view, req.where),
        serverComputed: (req) => serverComputed(req.where),
      },
    });
    this.cfg = cfg;
    this.store = store;
  }

  async create(view: New): Promise<View>;
  async create(views: New[]): Promise<View[]>;
  async create(views: New | New[]): Promise<View | View[]> {
    const isMany = Array.isArray(views);
    const res = await this.cfg.unary.send(
      "/view/create",
      { views: array.toArray(views) },
      createReqZ,
      createResZ,
    );
    this.store.set(res.views);
    return isMany ? res.views : res.views[0];
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      query.partialUpdate(this.store, key, { name }),
      this.cfg.ontology.cache.renameResource(ontologyID(key), name),
    ];
    await query.optimistic({
      rollbacks: rename(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send("/view/rename", { key, name }, renameReqZ, emptyResZ),
    });
    rename();
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const ids = ontologyID(keysArr);
    const drop = () => [
      this.cfg.ontology.cache.deleteRelationships(ids),
      this.store.delete(keysArr),
      this.cfg.ontology.cache.resources.delete(ontology.idToString(ids)),
    ];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/view/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  /** Subscribes to every view set delivered to the cache. */
  onSet(handler: (view: View) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "set") handler(event.value);
    });
  }

  /** Subscribes to every view delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  private async execRetrieve(req: RetrieveRequest): Promise<View[]> {
    const res = await this.cfg.unary.send(
      "/view/retrieve",
      req,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.views;
  }
}
