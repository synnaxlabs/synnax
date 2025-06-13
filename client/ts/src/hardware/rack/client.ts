// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { type UnknownRecord } from "@synnaxlabs/x";
import { array } from "@synnaxlabs/x/array";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { z } from "zod/v4";

import { framer } from "@/framer";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  ONTOLOGY_TYPE,
  type Payload,
  rackZ,
  type State,
  stateZ,
} from "@/hardware/rack/payload";
import { type task } from "@/hardware/task";
import { ontology } from "@/ontology";
import { analyzeParams, checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";

const RETRIEVE_ENDPOINT = "/hardware/rack/retrieve";
const CREATE_ENDPOINT = "/hardware/rack/create";
const DELETE_ENDPOINT = "/hardware/rack/delete";

export const STATE_CHANNEL_NAME = "sy_rack_state";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  search: z.string().optional(),
  embedded: z.boolean().optional(),
  hostIsNode: z.boolean().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  includeState: z.boolean().optional(),
});

const retrieveResZ = z.object({ racks: nullableArrayZ(rackZ) });

const createReqZ = z.object({ racks: newZ.array() });

const createResZ = z.object({ racks: rackZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });

const deleteResZ = z.object({});

export interface RetrieveOptions {
  includeState?: boolean;
}

export class Client implements AsyncTermSearcher<string, Key, Payload> {
  readonly type = ONTOLOGY_TYPE;
  private readonly client: UnaryClient;
  private readonly tasks: task.Client;
  private readonly frameClient: framer.Client;

  constructor(
    client: UnaryClient,
    taskClient: task.Client,
    frameClient: framer.Client,
  ) {
    this.client = client;
    this.tasks = taskClient;
    this.frameClient = frameClient;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: array.toArray(keys) },
      deleteReqZ,
      deleteResZ,
    );
  }

  async create(rack: New): Promise<Rack>;
  async create(racks: New[]): Promise<Rack[]>;
  async create(rack: New | New[]): Promise<Rack | Rack[]> {
    const isSingle = !Array.isArray(rack);
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { racks: array.toArray(rack) },
      createReqZ,
      createResZ,
    );
    const sugared = this.sugar(res.racks);
    if (isSingle) return sugared[0];
    return sugared;
  }

  async search(term: string): Promise<Rack[]> {
    const res = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      { search: term },
      retrieveReqZ,
      retrieveResZ,
    );
    return this.sugar(res.racks);
  }

  async page(offset: number, limit: number): Promise<Rack[]> {
    const res = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      { offset, limit },
      retrieveReqZ,
      retrieveResZ,
    );
    return this.sugar(res.racks);
  }

  async retrieve(key: string | Key, options?: RetrieveOptions): Promise<Rack>;
  async retrieve(keys: Key[], options?: RetrieveOptions): Promise<Rack[]>;
  async retrieve(
    racks: string | Key | Key[],
    options?: RetrieveOptions,
  ): Promise<Rack | Rack[]> {
    const { variant, normalized, single } = analyzeParams(racks, {
      string: "names",
      number: "keys",
    });
    const res = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      {
        [variant]: normalized,
        includeState: options?.includeState,
      },
      retrieveReqZ,
      retrieveResZ,
    );
    const sugared = this.sugar(res.racks);
    checkForMultipleOrNoResults("Rack", racks, sugared, single);
    return single ? sugared[0] : sugared;
  }

  async openStateObserver(): Promise<framer.ObservableStreamer<State[]>> {
    return new framer.ObservableStreamer<State[]>(
      await this.frameClient.openStreamer(STATE_CHANNEL_NAME),
      (fr) => {
        const data = fr.get(STATE_CHANNEL_NAME);
        if (data.length === 0) return [[], false];
        const states = data.parseJSON(stateZ);
        return [states, true];
      },
    );
  }

  private sugar(payloads: Payload[]): Rack[] {
    return payloads.map(
      ({ key, name, state }) => new Rack(key, name, this.tasks, state),
    );
  }
}

export class Rack {
  key: Key;
  name: string;
  state?: State;
  private readonly tasks: task.Client;

  constructor(key: Key, name: string, taskClient: task.Client, state?: State) {
    this.key = key;
    this.name = name;
    this.tasks = taskClient;
    this.state = state;
  }

  async listTasks(): Promise<task.Task[]> {
    return await this.tasks.retrieve(this.key);
  }

  async retrieveTaskByName(name: string): Promise<task.Task> {
    return await this.tasks.retrieveByName(name, this.key);
  }

  async retrieveTaskByType(type: string): Promise<task.Task[]> {
    return await this.tasks.retrieveByType(type, this.key);
  }

  async createTask<
    Config extends UnknownRecord,
    Details extends {} = UnknownRecord,
    Type extends string = string,
  >(task: task.New<Config, Type>): Promise<task.Task<Config, Details, Type>> {
    task.key = (
      (BigInt(this.key) << 32n) +
      (BigInt(task.key ?? 0) & 0xffffffffn)
    ).toString();
    return await this.tasks.create<Config, Details, Type>(task);
  }

  async deleteTask(task: bigint): Promise<void> {
    await this.tasks.delete([task]);
  }
}

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key: key.toString() });
