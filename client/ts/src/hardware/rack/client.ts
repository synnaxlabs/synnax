// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { type UnknownRecord } from "@synnaxlabs/x";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { toArray } from "@synnaxlabs/x/toArray";
import { z } from "zod";

import {
  type Key,
  keyZ,
  type New,
  newZ,
  ONTOLOGY_TYPE,
  type Payload,
  rackZ,
} from "@/hardware/rack/payload";
import { type task } from "@/hardware/task";
import { ontology } from "@/ontology";
import { analyzeParams, checkForMultipleOrNoResults } from "@/util/retrieve";
import { nullableArrayZ } from "@/util/zod";

const RETRIEVE_ENDPOINT = "/hardware/rack/retrieve";
const CREATE_ENDPOINT = "/hardware/rack/create";
const DELETE_ENDPOINT = "/hardware/rack/delete";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  search: z.string().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

const retrieveResZ = z.object({ racks: nullableArrayZ(rackZ) });

const createReqZ = z.object({ racks: newZ.array() });

const createResZ = z.object({ racks: rackZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });

const deleteResZ = z.object({});

export class Client implements AsyncTermSearcher<string, Key, Payload> {
  readonly type = ONTOLOGY_TYPE;
  private readonly client: UnaryClient;
  private readonly tasks: task.Client;

  constructor(client: UnaryClient, taskClient: task.Client) {
    this.client = client;
    this.tasks = taskClient;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: toArray(keys) },
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
      { racks: toArray(rack) },
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

  async retrieve(key: string | Key): Promise<Rack>;
  async retrieve(keys: Key[]): Promise<Rack[]>;
  async retrieve(racks: string | Key | Key[]): Promise<Rack | Rack[]> {
    const { variant, normalized, single } = analyzeParams(racks, {
      string: "names",
      number: "keys",
    });
    const res = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.client,
      RETRIEVE_ENDPOINT,
      { [variant]: normalized },
      retrieveReqZ,
      retrieveResZ,
    );
    const sugared = this.sugar(res.racks);
    checkForMultipleOrNoResults("Rack", racks, sugared, single);
    return single ? sugared[0] : sugared;
  }

  private sugar(payloads: Payload[]): Rack[] {
    return payloads.map(({ key, name }) => new Rack(key, name, this.tasks));
  }
}

export class Rack {
  key: Key;
  name: string;
  private readonly tasks: task.Client;

  constructor(key: Key, name: string, taskClient: task.Client) {
    this.key = key;
    this.name = name;
    this.tasks = taskClient;
  }

  async listTasks(): Promise<task.Task[]> {
    return await this.tasks.retrieve(this.key);
  }

  async retrieveTaskByName(name: string): Promise<task.Task> {
    return await this.tasks.retrieveByName(name, this.key);
  }

  async createTask<
    C extends UnknownRecord,
    D extends {} = UnknownRecord,
    T extends string = string,
  >(task: task.New<C, T>): Promise<task.Task<C, D, T>> {
    task.key = (
      (BigInt(this.key) << 32n) +
      (BigInt(task.key ?? 0) & 0xffffffffn)
    ).toString();
    return await this.tasks.create<C, D, T>(task);
  }

  async deleteTask(task: bigint): Promise<void> {
    await this.tasks.delete([task]);
  }
}

export const ontologyID = (key: Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key: key.toString() });
