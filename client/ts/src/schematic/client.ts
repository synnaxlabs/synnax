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

import { type cache } from "@/cache";
import { type dispatch } from "@/dispatch";
import { project } from "@/project";
import {
  type Action,
  dispatchReqZ,
  rename as renameAction,
} from "@/schematic/actions.gen";
import { bindStore, STORE_KEY } from "@/schematic/store";
import { symbol } from "@/schematic/symbol";
import {
  type Key,
  keyZ,
  type New,
  type Schematic,
  schematicZ,
} from "@/schematic/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

const deleteReqZ = z.object({ keys: keyZ.array() });

const copyReqZ = z.object({
  key: keyZ,
  name: z.string(),
  snapshot: z.boolean(),
});

const retrieveReqZ = z.object({ keys: keyZ.array() });
const singleRetrieveParamsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

export const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveReqZ]);
export type RetrieveParams = z.input<typeof retrieveParamsZ>;
export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveReqZ>;
export type CopyParams = z.input<typeof copyReqZ>;

const retrieveResZ = z.object({ schematics: schematicZ.array() });

const createReqZ = z.object({
  project: project.keyZ,
  schematics: schematicZ.array(),
});
const createResZ = z.object({ schematics: schematicZ.array() });

const copyResZ = z.object({ schematic: schematicZ });
const emptyResZ = z.object({});

export class Client {
  readonly symbols: symbol.Client;
  private readonly client: UnaryClient;
  private readonly store_?: cache.Store<Key, Schematic>;
  private readonly dispatcher_?: dispatch.Controller<Key, Schematic, Action>;

  constructor(client: UnaryClient, engine?: cache.Engine) {
    this.client = client;
    this.symbols = new symbol.Client(client, engine);
    if (engine == null) return;
    this.dispatcher_ = bindStore(engine);
    this.store_ = engine.store(STORE_KEY);
  }

  /**
   * Read surface of the schematic cache.
   * @throws when the cache was disabled at client construction.
   */
  get store(): cache.Store<Key, Schematic> {
    if (this.store_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.store_;
  }

  /**
   * Action-dispatch controller over the schematic cache.
   * @throws when the cache was disabled at client construction.
   */
  get dispatcher(): dispatch.Controller<Key, Schematic, Action> {
    if (this.dispatcher_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.dispatcher_;
  }

  async create(project: project.Key, schematic: New): Promise<Schematic>;
  async create(project: project.Key, schematics: New[]): Promise<Schematic[]>;
  async create(
    project: project.Key,
    schematics: New | New[],
  ): Promise<Schematic | Schematic[]> {
    const isMany = Array.isArray(schematics);
    const res = await this.client.send(
      "/schematic/create",
      { project, schematics: array.toArray(schematics) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.schematics : res.schematics[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatch(key, "", [renameAction({ name })]);
  }

  async dispatch(key: Key, dispatchKey: string, actions: Action[]): Promise<void> {
    await this.client.send(
      "/schematic/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async retrieve(params: RetrieveSingleParams): Promise<Schematic>;
  async retrieve(params: RetrieveMultipleParams): Promise<Schematic[]>;
  async retrieve(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<Schematic | Schematic[]> {
    const isSingle = singleRetrieveParamsZ.safeParse(params).success;
    const res = await this.client.send(
      "/schematic/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Schematic", params, res.schematics, isSingle);
    return isSingle ? res.schematics[0] : res.schematics;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await this.client.send(
      "/schematic/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }

  async copy(params: CopyParams): Promise<Schematic> {
    const res = await this.client.send("/schematic/copy", params, copyReqZ, copyResZ);
    return res.schematic;
  }
}
