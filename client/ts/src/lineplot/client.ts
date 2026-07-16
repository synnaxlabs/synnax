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
import {
  type Action,
  dispatchReqZ,
  rename as renameAction,
} from "@/lineplot/actions.gen";
import { bindStore, STORE_KEY } from "@/lineplot/store";
import {
  type Key,
  keyZ,
  type LinePlot,
  linePlotZ,
  type New,
} from "@/lineplot/types.gen";
import { project } from "@/project";
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

const retrieveResZ = z.object({ linePlots: linePlotZ.array().default(() => []) });

const createReqZ = z.object({ project: project.keyZ, linePlots: linePlotZ.array() });
const createResZ = z.object({ linePlots: linePlotZ.array() });

const emptyResZ = z.object({});

export class Client {
  private readonly client: UnaryClient;
  private readonly store_?: cache.Store<Key, LinePlot>;
  private readonly dispatcher_?: dispatch.Controller<Key, LinePlot, Action>;

  constructor(client: UnaryClient, engine?: cache.Engine) {
    this.client = client;
    if (engine == null) return;
    this.dispatcher_ = bindStore(engine);
    this.store_ = engine.store(STORE_KEY);
  }

  /**
   * Read surface of the line plot cache.
   * @throws when the cache was disabled at client construction.
   */
  get store(): cache.Store<Key, LinePlot> {
    if (this.store_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.store_;
  }

  /**
   * Action-dispatch controller over the line plot cache.
   * @throws when the cache was disabled at client construction.
   */
  get dispatcher(): dispatch.Controller<Key, LinePlot, Action> {
    if (this.dispatcher_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.dispatcher_;
  }

  async create(project: project.Key, linePlot: New): Promise<LinePlot>;
  async create(project: project.Key, linePlots: New[]): Promise<LinePlot[]>;
  async create(
    project: project.Key,
    linePlots: New | New[],
  ): Promise<LinePlot | LinePlot[]> {
    const isMany = Array.isArray(linePlots);
    const res = await this.client.send(
      "/lineplot/create",
      { project, linePlots: array.toArray(linePlots) },
      createReqZ,
      createResZ,
    );
    return isMany ? res.linePlots : res.linePlots[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatch(key, "", [renameAction({ name })]);
  }

  async dispatch(key: Key, dispatchKey: string, actions: Action[]): Promise<void> {
    await this.client.send(
      "/lineplot/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  async retrieve(params: RetrieveSingleParams): Promise<LinePlot>;
  async retrieve(params: RetrieveMultipleParams): Promise<LinePlot[]>;
  async retrieve(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): Promise<LinePlot | LinePlot[]> {
    const isSingle = singleRetrieveParamsZ.safeParse(params).success;
    const res = await this.client.send(
      "/lineplot/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("LinePlot", params, res.linePlots, isSingle);
    return isSingle ? res.linePlots[0] : res.linePlots;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await this.client.send(
      "/lineplot/delete",
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }
}
