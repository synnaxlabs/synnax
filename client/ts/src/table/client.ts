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
import { type Action, dispatchReqZ, rename as renameAction } from "@/table/actions.gen";
import { bindStore, STORE_KEY } from "@/table/store";
import { type Key, keyZ, type New, type Table, tableZ } from "@/table/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveReqZ = z.object({ keys: keyZ.array() });
const singleRetrieveParamsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

export const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveReqZ]);
export type RetrieveParams = z.input<typeof retrieveParamsZ>;
export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveReqZ>;

const retrieveResZ = z.object({ tables: tableZ.array().default(() => []) });

const createReqZ = z.object({ project: project.keyZ, tables: tableZ.array() });
const createResZ = z.object({ tables: tableZ.array() });

const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;
  private readonly store_?: cache.Store<Key, Table>;
  private readonly dispatcher_?: dispatch.Controller<Key, Table, Action>;

  constructor(client: UnaryClient, engine?: cache.Engine) {
    this.client = client;
    if (engine == null) return;
    this.dispatcher_ = bindStore(engine);
    this.store_ = engine.store(STORE_KEY);
  }

  /**
   * Read surface of the table cache.
   * @throws when the cache was disabled at client construction.
   */
  get store(): cache.Store<Key, Table> {
    if (this.store_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.store_;
  }

  /**
   * Action-dispatch controller over the table cache.
   * @throws when the cache was disabled at client construction.
   */
  get dispatcher(): dispatch.Controller<Key, Table, Action> {
    if (this.dispatcher_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.dispatcher_;
  }

  async create(project: project.Key, table: New): Promise<Table>;
  async create(project: project.Key, tables: New[]): Promise<Table[]>;
  async create(project: project.Key, tables: New | New[]): Promise<Table | Table[]> {
    const isMany = Array.isArray(tables);
    const res = await this.client.send(
      "/table/create",
      { project, tables: array.toArray(tables) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.tables : res.tables[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatch(key, "", [renameAction({ name })]);
  }

  async dispatch(key: Key, dispatchKey: string, actions: Action[]): Promise<void> {
    await this.client.send(
      "/table/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async retrieve(params: RetrieveSingleParams): Promise<Table>;
  async retrieve(params: RetrieveMultipleParams): Promise<Table[]>;
  async retrieve(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<Table | Table[]> {
    const isSingle = singleRetrieveParamsZ.safeParse(params).success;
    const res = await this.client.send(
      "/table/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Table", params, res.tables, isSingle);
    return isSingle ? res.tables[0] : res.tables;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await this.client.send(
      "/table/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}
